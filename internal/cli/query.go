package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/01x/codeindex/internal/config"
	"github.com/01x/codeindex/internal/graph"
	"github.com/01x/codeindex/internal/query"
	"github.com/spf13/cobra"
)

// queryResponse is implemented by all query result types.
// It exists so writeQueryJSON has an explicit type signature instead of any.
type queryResponse interface {
	isQueryResponse()
}

// --- response types ---

type fileStructureResponse struct {
	query.FileStructure
	Metadata query.QueryMetadata `json:"metadata"`
}

func (fileStructureResponse) isQueryResponse() {}

type findSymbolResponse struct {
	Results  []query.SymbolResult `json:"results"`
	Metadata query.QueryMetadata  `json:"metadata"`
}

func (findSymbolResponse) isQueryResponse() {}

type referencesResponse struct {
	Results  []query.ReferenceResult `json:"results"`
	Metadata query.QueryMetadata     `json:"metadata"`
}

func (referencesResponse) isQueryResponse() {}

type callersResponse struct {
	Results  []query.CallerResult `json:"results"`
	Metadata query.QueryMetadata  `json:"metadata"`
}

func (callersResponse) isQueryResponse() {}

type subgraphResponse struct {
	Nodes    []query.SubgraphNode `json:"nodes"`
	Edges    []query.SubgraphEdge `json:"edges"`
	Metadata query.QueryMetadata  `json:"metadata"`
}

func (subgraphResponse) isQueryResponse() {}

// --- commands ---

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query the code index",
	Long: `Run structural queries against the code index knowledge graph.

All subcommands output JSON to stdout. Use these commands directly from
coding agents via Bash — no MCP server required.

Examples:
  codeindex query file-structure src/api.ts
  codeindex query find-symbol handleRequest --kind fn
  codeindex query references handleRequest
  codeindex query callers handleRequest --depth 5
  codeindex query subgraph handleRequest --depth 2`,
}

func init() {
	queryCmd.AddCommand(queryFileStructureCmd)
	queryCmd.AddCommand(queryFindSymbolCmd)
	queryCmd.AddCommand(queryReferencesCmd)
	queryCmd.AddCommand(queryCallersCmd)
	queryCmd.AddCommand(querySubgraphCmd)

	queryFindSymbolCmd.Flags().String("kind", "", "Filter by symbol kind: fn, class, type, interface, var")
	queryCallersCmd.Flags().Int("depth", 3, "Max call graph depth (1-10)")
	querySubgraphCmd.Flags().Int("depth", 2, "Max graph depth (1-10)")
	querySubgraphCmd.Flags().StringSlice("edge-kinds", nil, "Filter by edge kind: calls, imports, references, implements, extends")
}

// --- file-structure ---

var queryFileStructureCmd = &cobra.Command{
	Use:   "file-structure <path>",
	Short: "Get the structural skeleton of a file (symbols, imports)",
	Long: `Output the structural skeleton of a file: symbols, functions, classes, types, and imports.

The response includes a 'stale' flag. If true, run 'codeindex reindex <path>'
before trusting the data.

Examples:
  codeindex query file-structure src/api.ts`,
	Args: cobra.ExactArgs(1),
	RunE: runQueryFileStructure,
}

func runQueryFileStructure(cmd *cobra.Command, args []string) error {
	store, engine, err := openQueryEngine()
	if err != nil {
		return err
	}
	defer store.Close()

	start := time.Now()
	result, meta, err := engine.GetFileStructure(args[0])
	if err != nil {
		return fmt.Errorf("file-structure query: %w", err)
	}
	meta.QueryDurationMs = time.Since(start).Milliseconds()

	return writeQueryJSON(cmd, fileStructureResponse{result, meta})
}

// --- find-symbol ---

var queryFindSymbolCmd = &cobra.Command{
	Use:   "find-symbol <name>",
	Short: "Find where a symbol is defined",
	Long: `Locate the definition of a symbol across the codebase.

Examples:
  codeindex query find-symbol handleRequest
  codeindex query find-symbol User --kind class`,
	Args: cobra.ExactArgs(1),
	RunE: runQueryFindSymbol,
}

func runQueryFindSymbol(cmd *cobra.Command, args []string) error {
	store, engine, err := openQueryEngine()
	if err != nil {
		return err
	}
	defer store.Close()

	kind, _ := cmd.Flags().GetString("kind")

	start := time.Now()
	results, meta, err := engine.FindSymbol(args[0], kind)
	if err != nil {
		return fmt.Errorf("find-symbol query: %w", err)
	}
	meta.QueryDurationMs = time.Since(start).Milliseconds()

	return writeQueryJSON(cmd, findSymbolResponse{results, meta})
}

// --- references ---

var queryReferencesCmd = &cobra.Command{
	Use:   "references <symbol>",
	Short: "Find every usage of a symbol",
	Long: `Find every file and line that uses a symbol (calls, imports, references).

Examples:
  codeindex query references handleRequest`,
	Args: cobra.ExactArgs(1),
	RunE: runQueryReferences,
}

func runQueryReferences(cmd *cobra.Command, args []string) error {
	store, engine, err := openQueryEngine()
	if err != nil {
		return err
	}
	defer store.Close()

	start := time.Now()
	results, meta, err := engine.GetReferences(args[0])
	if err != nil {
		return fmt.Errorf("references query: %w", err)
	}
	meta.QueryDurationMs = time.Since(start).Milliseconds()

	return writeQueryJSON(cmd, referencesResponse{results, meta})
}

// --- callers ---

var queryCallersCmd = &cobra.Command{
	Use:   "callers <symbol>",
	Short: "Trace the call graph upstream from a symbol",
	Long: `Walk the call graph upstream from a symbol up to a given depth.

Examples:
  codeindex query callers handleRequest
  codeindex query callers handleRequest --depth 5`,
	Args: cobra.ExactArgs(1),
	RunE: runQueryCallers,
}

func runQueryCallers(cmd *cobra.Command, args []string) error {
	store, engine, err := openQueryEngine()
	if err != nil {
		return err
	}
	defer store.Close()

	depth, _ := cmd.Flags().GetInt("depth")

	start := time.Now()
	results, meta, err := engine.GetCallers(args[0], depth)
	if err != nil {
		return fmt.Errorf("callers query: %w", err)
	}
	meta.QueryDurationMs = time.Since(start).Milliseconds()

	return writeQueryJSON(cmd, callersResponse{results, meta})
}

// --- subgraph ---

var querySubgraphCmd = &cobra.Command{
	Use:   "subgraph <symbol>",
	Short: "Get the graph neighborhood around a symbol",
	Long: `Return all nodes and edges within N hops of a symbol.

Examples:
  codeindex query subgraph handleRequest
  codeindex query subgraph handleRequest --depth 3 --edge-kinds calls,imports`,
	Args: cobra.ExactArgs(1),
	RunE: runQuerySubgraph,
}

func runQuerySubgraph(cmd *cobra.Command, args []string) error {
	store, engine, err := openQueryEngine()
	if err != nil {
		return err
	}
	defer store.Close()

	depth, _ := cmd.Flags().GetInt("depth")
	edgeKinds, _ := cmd.Flags().GetStringSlice("edge-kinds")

	start := time.Now()
	result, meta, err := engine.GetSubgraph(args[0], depth, edgeKinds)
	if err != nil {
		return fmt.Errorf("subgraph query: %w", err)
	}
	meta.QueryDurationMs = time.Since(start).Milliseconds()

	return writeQueryJSON(cmd, subgraphResponse{result.Nodes, result.Edges, meta})
}

// --- shared helpers ---

// openQueryEngine loads config, opens the SQLite store, and returns a ready
// query engine. The caller must defer store.Close().
func openQueryEngine() (*graph.SQLiteStore, *query.Engine, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, nil, fmt.Errorf("getting working directory: %w", err)
	}

	cfg, _, err := config.LoadOrDetect(dir)
	if err != nil {
		return nil, nil, ErrConfigInvalid(err.Error())
	}

	dbPath := filepath.Join(dir, cfg.IndexPath, "graph.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("no index found — run 'codeindex init' to get started")
	}

	store, err := graph.NewSQLiteStore(dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("opening graph store: %w", err)
	}

	if err := store.Migrate(); err != nil {
		store.Close()
		return nil, nil, fmt.Errorf("migrating schema: %w", err)
	}

	return store, query.NewEngine(store, dir), nil
}

// writeQueryJSON serializes resp as indented JSON and writes it to cmd's stdout.
func writeQueryJSON(cmd *cobra.Command, resp queryResponse) error {
	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling response: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}
