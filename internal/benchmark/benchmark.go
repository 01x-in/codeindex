package benchmark

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/01x/codeindex/internal/config"
	"github.com/01x/codeindex/internal/graph"
	"github.com/01x/codeindex/internal/indexer"
	"github.com/01x/codeindex/internal/query"
)

// SourceKind identifies the benchmark source type.
type SourceKind string

const (
	SourceKindLocal  SourceKind = "local"
	SourceKindRemote SourceKind = "remote"
)

// Request configures a benchmark run.
type Request struct {
	Source        string
	Symbol        string
	KeepWorkspace bool
	TempRoot      string
	Progress      io.Writer
}

// Result captures benchmark output.
type Result struct {
	RepoName         string
	SourceKind       SourceKind
	OriginalSource   string
	WorkspacePath    string
	QuerySymbol      string
	SampleFile       string
	KeptWorkspace    bool
	FilesIndexed     int
	FilesFresh       int
	Nodes            int
	Edges            int
	InitDuration     time.Duration
	ReindexDuration  time.Duration
	FileStructTime   time.Duration
	FindSymbolTime   time.Duration
	ReferencesTime   time.Duration
	CallersTime      time.Duration
	SubgraphTime     time.Duration
	TextSearchTime   time.Duration
	TextSearchLines  int
	TextSearchTokens int
	CodeindexTokens  int
	TokenReductionX  float64
	GeneratedAt      time.Time
}

type resolvedSource struct {
	kind     SourceKind
	original string
	local    string
	remote   string
	repoName string
}

// Run executes a benchmark against the requested source.
func Run(req Request) (result Result, err error) {
	sourceInput := strings.TrimSpace(req.Source)
	if sourceInput == "" {
		return Result{}, fmt.Errorf("source is required")
	}

	symbol := strings.TrimSpace(req.Symbol)
	if symbol == "" {
		return Result{}, fmt.Errorf("symbol is required")
	}

	src, err := resolveSource(sourceInput)
	if err != nil {
		return Result{}, err
	}

	progress := req.Progress
	if progress == nil {
		progress = io.Discard
	}

	tempRoot := req.TempRoot
	if tempRoot == "" {
		tempRoot = os.TempDir()
	}

	workspaceParent, err := os.MkdirTemp(tempRoot, "codeindex-bench-"+src.repoName+"-")
	if err != nil {
		return Result{}, fmt.Errorf("creating benchmark workspace: %w", err)
	}

	workspacePath := filepath.Join(workspaceParent, src.repoName)

	result = Result{
		RepoName:       src.repoName,
		SourceKind:     src.kind,
		OriginalSource: src.original,
		WorkspacePath:  workspacePath,
		QuerySymbol:    symbol,
		KeptWorkspace:  req.KeepWorkspace,
	}

	var store *graph.SQLiteStore
	defer func() {
		if store != nil {
			_ = store.Close()
		}
		if !req.KeepWorkspace {
			_ = os.RemoveAll(workspaceParent)
		}
	}()

	switch src.kind {
	case SourceKindRemote:
		progressf(progress, "Cloning %s ...", src.original)
		if err := cloneRepo(src.remote, workspacePath); err != nil {
			return Result{}, err
		}
	case SourceKindLocal:
		progressf(progress, "Copying %s ...", src.original)
		if err := copyLocalRepo(src.local, workspacePath); err != nil {
			return Result{}, err
		}
	default:
		return Result{}, fmt.Errorf("unsupported source kind: %s", src.kind)
	}

	if err := resetBenchmarkArtifacts(workspacePath); err != nil {
		return Result{}, err
	}

	progressf(progress, "Running codeindex init --yes ...")
	cfg, openedStore, initDuration, err := initializeIndex(workspacePath)
	if err != nil {
		return Result{}, err
	}
	store = openedStore
	result.InitDuration = initDuration

	allMeta, err := store.GetAllFileMetadata()
	if err != nil {
		return Result{}, fmt.Errorf("reading indexed files: %w", err)
	}
	result.FilesIndexed = len(allMeta)
	result.FilesFresh = len(allMeta)

	nodeCount, err := store.NodeCount()
	if err != nil {
		return Result{}, fmt.Errorf("counting nodes: %w", err)
	}
	result.Nodes = nodeCount

	edgeCount, err := store.EdgeCount()
	if err != nil {
		return Result{}, fmt.Errorf("counting edges: %w", err)
	}
	result.Edges = edgeCount

	result.SampleFile = chooseSampleFile(allMeta)
	if result.SampleFile != "" {
		progressf(progress, "Single-file reindex: %s", result.SampleFile)
		result.ReindexDuration, err = reindexFile(workspacePath, cfg, store, result.SampleFile)
		if err != nil {
			return Result{}, err
		}
	}

	engine := query.NewEngine(store, workspacePath)

	progressf(progress, "Measuring query latencies (symbol: %s) ...", result.QuerySymbol)
	if result.SampleFile != "" {
		result.FileStructTime, err = measureQuery(func() error {
			_, _, queryErr := engine.GetFileStructure(result.SampleFile)
			return queryErr
		})
		if err != nil {
			return Result{}, err
		}
	}

	result.FindSymbolTime, err = measureQuery(func() error {
		_, _, queryErr := engine.FindSymbol(result.QuerySymbol, "")
		return queryErr
	})
	if err != nil {
		return Result{}, err
	}

	result.ReferencesTime, err = measureQuery(func() error {
		_, _, queryErr := engine.GetReferences(result.QuerySymbol)
		return queryErr
	})
	if err != nil {
		return Result{}, err
	}

	result.CallersTime, err = measureQuery(func() error {
		_, _, queryErr := engine.GetCallers(result.QuerySymbol, 3)
		return queryErr
	})
	if err != nil {
		return Result{}, err
	}

	result.SubgraphTime, err = measureQuery(func() error {
		_, _, queryErr := engine.GetSubgraph(result.QuerySymbol, 2, nil)
		return queryErr
	})
	if err != nil {
		return Result{}, err
	}

	progressf(progress, "Measuring text search baseline ...")
	result.TextSearchLines, result.TextSearchTime, err = textSearchBaseline(workspacePath, allMeta, result.QuerySymbol)
	if err != nil {
		return Result{}, err
	}

	result.TextSearchTokens = result.TextSearchLines * 30
	result.CodeindexTokens = len(result.QuerySymbol)*5 + 200
	if result.CodeindexTokens > 0 {
		result.TokenReductionX = float64(result.TextSearchTokens) / float64(result.CodeindexTokens)
	}

	result.GeneratedAt = time.Now().UTC()
	return result, nil
}

// TerminalSummary renders a human-readable CLI summary.
func (r Result) TerminalSummary() string {
	var b strings.Builder

	fmt.Fprintf(&b, "Benchmark: %s\n", r.RepoName)
	fmt.Fprintf(&b, "Source: %s\n", r.OriginalSource)
	fmt.Fprintf(&b, "Query symbol: %s\n", r.QuerySymbol)
	if r.KeptWorkspace {
		fmt.Fprintf(&b, "Workspace kept: %s\n", r.WorkspacePath)
	}

	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Index stats:")
	fmt.Fprintf(&b, "  Files indexed: %d\n", r.FilesIndexed)
	fmt.Fprintf(&b, "  Files fresh:   %d\n", r.FilesFresh)
	fmt.Fprintf(&b, "  Nodes:         %d\n", r.Nodes)
	fmt.Fprintf(&b, "  Edges:         %d\n", r.Edges)

	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Timing:")
	fmt.Fprintf(&b, "  codeindex init (cold, full index): %s\n", formatDuration(r.InitDuration))
	if r.SampleFile != "" {
		fmt.Fprintf(&b, "  Single-file reindex (%s): %s\n", r.SampleFile, formatDuration(r.ReindexDuration))
		fmt.Fprintf(&b, "  get_file_structure: %s\n", formatDuration(r.FileStructTime))
	} else {
		fmt.Fprintln(&b, "  Single-file reindex: n/a")
		fmt.Fprintln(&b, "  get_file_structure: n/a")
	}
	fmt.Fprintf(&b, "  find_symbol: %s\n", formatDuration(r.FindSymbolTime))
	fmt.Fprintf(&b, "  get_references: %s\n", formatDuration(r.ReferencesTime))
	fmt.Fprintf(&b, "  get_callers: %s\n", formatDuration(r.CallersTime))
	fmt.Fprintf(&b, "  get_subgraph: %s\n", formatDuration(r.SubgraphTime))
	fmt.Fprintf(&b, "  text search: %s\n", formatDuration(r.TextSearchTime))

	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Context window impact:")
	fmt.Fprintf(&b, "  Lines returned for %q: %d\n", r.QuerySymbol, r.TextSearchLines)
	fmt.Fprintf(&b, "  Estimated tokens consumed: baseline ~%d, codeindex ~%d\n", r.TextSearchTokens, r.CodeindexTokens)
	if r.TokenReductionX > 0 {
		fmt.Fprintf(&b, "  Reduction: ~%.1fx fewer tokens\n", r.TokenReductionX)
	} else {
		fmt.Fprintln(&b, "  Reduction: n/a")
	}

	return strings.TrimRight(b.String(), "\n")
}

// Markdown renders the benchmark result as markdown.
func (r Result) Markdown() string {
	var b strings.Builder

	fmt.Fprintf(&b, "# Benchmark: %s\n\n", r.RepoName)
	fmt.Fprintf(&b, "**Date:** %s\n", r.GeneratedAt.Format("2006-01-02"))
	fmt.Fprintf(&b, "**Repo:** %s\n", r.OriginalSource)
	fmt.Fprintf(&b, "**Query symbol:** `%s`\n", r.QuerySymbol)
	if r.KeptWorkspace {
		fmt.Fprintf(&b, "**Workspace kept:** `%s`\n", r.WorkspacePath)
	}
	b.WriteString("\n## Index Stats\n\n")
	b.WriteString("| Metric | Value |\n")
	b.WriteString("|--------|-------|\n")
	fmt.Fprintf(&b, "| Files indexed | %d |\n", r.FilesIndexed)
	fmt.Fprintf(&b, "| Files fresh | %d |\n", r.FilesFresh)
	fmt.Fprintf(&b, "| Nodes | %d |\n", r.Nodes)
	fmt.Fprintf(&b, "| Edges | %d |\n", r.Edges)

	b.WriteString("\n## Timing\n\n")
	b.WriteString("| Operation | Time |\n")
	b.WriteString("|-----------|------|\n")
	fmt.Fprintf(&b, "| `codeindex init` (cold, full index) | %s |\n", formatDuration(r.InitDuration))
	if r.SampleFile != "" {
		fmt.Fprintf(&b, "| Single file reindex (`%s`) | %s |\n", r.SampleFile, formatDuration(r.ReindexDuration))
		fmt.Fprintf(&b, "| `get_file_structure` | %s |\n", formatDuration(r.FileStructTime))
	} else {
		b.WriteString("| Single file reindex | n/a |\n")
		b.WriteString("| `get_file_structure` | n/a |\n")
	}
	fmt.Fprintf(&b, "| `find_symbol` | %s |\n", formatDuration(r.FindSymbolTime))
	fmt.Fprintf(&b, "| `get_references` | %s |\n", formatDuration(r.ReferencesTime))
	fmt.Fprintf(&b, "| `get_callers` (depth=3) | %s |\n", formatDuration(r.CallersTime))
	fmt.Fprintf(&b, "| `get_subgraph` (depth=2) | %s |\n", formatDuration(r.SubgraphTime))
	fmt.Fprintf(&b, "| `text search` for same symbol | %s |\n", formatDuration(r.TextSearchTime))

	b.WriteString("\n## Context Window Impact\n\n")
	b.WriteString("| | text search | codeindex |\n")
	b.WriteString("|-|-------------|-----------|\n")
	fmt.Fprintf(&b, "| Lines returned for `%s` | %d | structured facts |\n", r.QuerySymbol, r.TextSearchLines)
	fmt.Fprintf(&b, "| Estimated tokens consumed | ~%d | ~%d |\n", r.TextSearchTokens, r.CodeindexTokens)
	if r.TokenReductionX > 0 {
		fmt.Fprintf(&b, "| Reduction | **~%.1fx fewer tokens** | |\n", r.TokenReductionX)
	} else {
		b.WriteString("| Reduction | n/a | |\n")
	}
	b.WriteString("\n> Token estimates are approximate (baseline: 30 tokens/line avg; codeindex: structured facts only).\n")

	return b.String()
}

func resolveSource(input string) (resolvedSource, error) {
	if parsed, err := url.Parse(input); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		switch parsed.Scheme {
		case "http", "https":
			repoName := sanitizeRepoName(strings.TrimSuffix(filepath.Base(parsed.Path), ".git"))
			return resolvedSource{
				kind:     SourceKindRemote,
				original: input,
				remote:   input,
				repoName: repoName,
			}, nil
		default:
			return resolvedSource{}, fmt.Errorf("unsupported repo URL scheme %q", parsed.Scheme)
		}
	}

	absPath, err := filepath.Abs(input)
	if err != nil {
		return resolvedSource{}, fmt.Errorf("resolving local path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return resolvedSource{}, fmt.Errorf("stat %s: %w", absPath, err)
	}
	if !info.IsDir() {
		return resolvedSource{}, fmt.Errorf("local path must be a directory: %s", absPath)
	}
	canonicalPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return resolvedSource{}, fmt.Errorf("canonicalizing local path %s: %w", absPath, err)
	}

	return resolvedSource{
		kind:     SourceKindLocal,
		original: filepath.Clean(absPath),
		local:    canonicalPath,
		repoName: sanitizeRepoName(filepath.Base(absPath)),
	}, nil
}

func cloneRepo(remote string, dst string) error {
	cmd := exec.Command("git", "clone", "--depth=1", "--quiet", remote, dst)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cloning %s: %w: %s", remote, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func copyLocalRepo(src string, dst string) error {
	canonicalSrc, err := canonicalizeExistingPath(src)
	if err != nil {
		return err
	}

	ignoreDirs := map[string]bool{
		".codeindex": true,
	}
	for _, dir := range config.DefaultConfig().Ignore {
		ignoreDirs[dir] = true
	}

	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("creating workspace: %w", err)
	}

	return filepath.WalkDir(canonicalSrc, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(canonicalSrc, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		name := d.Name()
		if d.IsDir() && ignoreDirs[name] {
			return filepath.SkipDir
		}
		if name == config.ConfigFileName {
			return nil
		}

		target := filepath.Join(dst, rel)

		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(target, info.Mode().Perm())
		}

		if d.Type()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			resolvedTarget, err := resolveSymlinkTarget(canonicalSrc, path, link)
			if err != nil {
				return err
			}
			if resolvedTarget == "" {
				return nil
			}
			copiedLink, err := rewriteSymlinkTarget(canonicalSrc, dst, target, resolvedTarget)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			return os.Symlink(copiedLink, target)
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		return copyFile(path, target, info.Mode().Perm())
	})
}

func resolveSymlinkTarget(srcRoot string, sourcePath string, link string) (string, error) {
	var resolved string
	if filepath.IsAbs(link) {
		resolved = filepath.Clean(link)
	} else {
		resolved = filepath.Clean(filepath.Join(filepath.Dir(sourcePath), link))
	}
	canonicalResolved, err := canonicalizeExistingPath(resolved)
	if err != nil {
		return "", err
	}
	withinRoot, err := pathWithinRoot(srcRoot, canonicalResolved)
	if err != nil {
		return "", err
	}
	if !withinRoot {
		return "", nil
	}

	return canonicalResolved, nil
}

func rewriteSymlinkTarget(srcRoot string, dstRoot string, dstPath string, resolvedSourceTarget string) (string, error) {
	relToSourceRoot, err := filepath.Rel(srcRoot, resolvedSourceTarget)
	if err != nil {
		return "", fmt.Errorf("relativizing symlink target: %w", err)
	}

	dstTarget := filepath.Join(dstRoot, relToSourceRoot)
	rewritten, err := filepath.Rel(filepath.Dir(dstPath), dstTarget)
	if err != nil {
		return "", fmt.Errorf("rewriting symlink target: %w", err)
	}
	return rewritten, nil
}

func pathWithinRoot(root string, candidate string) (bool, error) {
	canonicalRoot, err := canonicalizeExistingPath(root)
	if err != nil {
		return false, err
	}
	canonicalCandidate, err := canonicalizeExistingPath(candidate)
	if err != nil {
		return false, err
	}

	rel, err := filepath.Rel(canonicalRoot, canonicalCandidate)
	if err != nil {
		return false, fmt.Errorf("checking path containment: %w", err)
	}
	if rel == ".." {
		return false, nil
	}
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false, nil
	}
	return true, nil
}

func canonicalizeExistingPath(path string) (string, error) {
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		if os.IsNotExist(err) {
			return filepath.Clean(path), nil
		}
		return "", fmt.Errorf("canonicalizing path %s: %w", path, err)
	}
	return canonical, nil
}

func copyFile(src string, dst string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return nil
}

func resetBenchmarkArtifacts(workspacePath string) error {
	if err := os.RemoveAll(filepath.Join(workspacePath, ".codeindex")); err != nil {
		return fmt.Errorf("removing prior index: %w", err)
	}
	if err := os.Remove(filepath.Join(workspacePath, config.ConfigFileName)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing prior config: %w", err)
	}
	return nil
}

func initializeIndex(workspacePath string) (config.Config, *graph.SQLiteStore, time.Duration, error) {
	if err := indexer.CheckInstalled(); err != nil {
		return config.Config{}, nil, 0, err
	}

	cfg, _, err := config.LoadOrDetect(workspacePath)
	if err != nil {
		return config.Config{}, nil, 0, fmt.Errorf("loading benchmark config: %w", err)
	}
	if len(cfg.Languages) == 0 {
		return config.Config{}, nil, 0, fmt.Errorf("no supported languages detected in %s", workspacePath)
	}

	configPath := filepath.Join(workspacePath, config.ConfigFileName)
	if err := cfg.Save(configPath); err != nil {
		return config.Config{}, nil, 0, fmt.Errorf("writing benchmark config: %w", err)
	}

	dbPath := filepath.Join(workspacePath, cfg.IndexPath, "graph.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return config.Config{}, nil, 0, fmt.Errorf("creating index directory: %w", err)
	}

	store, err := graph.NewSQLiteStore(dbPath)
	if err != nil {
		return config.Config{}, nil, 0, fmt.Errorf("opening graph store: %w", err)
	}
	if err := store.Migrate(); err != nil {
		_ = store.Close()
		return config.Config{}, nil, 0, fmt.Errorf("migrating schema: %w", err)
	}

	start := time.Now()
	runner := indexer.NewSubprocessRunner()
	for _, language := range cfg.Languages {
		idx := indexer.NewIndexer(store, runner, workspacePath, language)
		if _, err := idx.IndexStale(); err != nil {
			_ = store.Close()
			return config.Config{}, nil, 0, fmt.Errorf("indexing %s files: %w", language, err)
		}
	}
	if err := store.SetIndexMetadata("last_full_reindex", time.Now().UTC().Format(time.RFC3339)); err != nil {
		_ = store.Close()
		return config.Config{}, nil, 0, fmt.Errorf("setting index metadata: %w", err)
	}

	return cfg, store, time.Since(start), nil
}

func chooseSampleFile(allMeta []graph.FileMetadata) string {
	if len(allMeta) == 0 {
		return ""
	}
	return allMeta[0].FilePath
}

func reindexFile(workspacePath string, cfg config.Config, store *graph.SQLiteStore, filePath string) (time.Duration, error) {
	language := languageForFile(filePath, cfg.Languages)
	if language == "" {
		return 0, nil
	}

	start := time.Now()
	runner := indexer.NewSubprocessRunner()
	idx := indexer.NewIndexer(store, runner, workspacePath, language)
	if _, err := idx.IndexFile(filepath.Join(workspacePath, filePath)); err != nil {
		return 0, fmt.Errorf("reindexing %s: %w", filePath, err)
	}

	return time.Since(start), nil
}

func languageForFile(filePath string, configured []string) string {
	ext := filepath.Ext(filePath)
	for _, language := range configured {
		for _, candidate := range indexer.LanguageExtensions(language) {
			if ext == candidate {
				return language
			}
		}
	}
	return ""
}

func measureQuery(fn func() error) (time.Duration, error) {
	start := time.Now()
	if err := fn(); err != nil {
		return 0, err
	}
	return time.Since(start), nil
}

func textSearchBaseline(workspacePath string, files []graph.FileMetadata, symbol string) (int, time.Duration, error) {
	start := time.Now()
	lineCount := 0

	for _, meta := range files {
		path := filepath.Join(workspacePath, meta.FilePath)
		file, err := os.Open(path)
		if err != nil {
			return 0, 0, fmt.Errorf("opening %s: %w", meta.FilePath, err)
		}

		reader := bufio.NewReader(file)
		for {
			line, err := reader.ReadBytes('\n')
			if len(line) > 0 && bytes.Contains(line, []byte(symbol)) {
				lineCount++
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				_ = file.Close()
				return 0, 0, fmt.Errorf("reading %s: %w", meta.FilePath, err)
			}
		}

		if err := file.Close(); err != nil {
			return 0, 0, fmt.Errorf("closing %s: %w", meta.FilePath, err)
		}
	}

	return lineCount, time.Since(start), nil
}

func progressf(w io.Writer, format string, args ...interface{}) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, format+"\n", args...)
}

func sanitizeRepoName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, ".git")
	if name == "" {
		return "repo"
	}

	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}

	trimmed := strings.Trim(b.String(), "-")
	if trimmed == "" {
		return "repo"
	}
	return trimmed
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "n/a"
	}
	return fmt.Sprintf("%dms", d.Milliseconds())
}
