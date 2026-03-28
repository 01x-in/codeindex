package indexer

import (
	"regexp"
	"strings"

	"github.com/01x/codeindex/internal/graph"
)

// ParseResult holds the parsed nodes and edges from ast-grep output.
type ParseResult struct {
	Nodes []graph.Node
	Edges []ParsedEdge
}

// ParsedEdge is an edge with source/target names (not IDs) for resolution.
type ParsedEdge struct {
	SourceName string
	TargetName string
	Kind       string // calls, imports, references
	FilePath   string
	Line       int
}

// Patterns for extracting names from AST match text.
var (
	funcNameRe      = regexp.MustCompile(`(?:async\s+)?function\s+(\w+)`)
	classNameRe     = regexp.MustCompile(`class\s+(\w+)`)
	interfaceNameRe = regexp.MustCompile(`interface\s+(\w+)`)
	typeNameRe      = regexp.MustCompile(`type\s+(\w+)`)
	importFromRe    = regexp.MustCompile(`from\s+['"]([^'"]+)['"]`)
	importNamesRe   = regexp.MustCompile(`import\s+\{([^}]+)\}`)
	callNameRe      = regexp.MustCompile(`^(\w+(?:\.\w+)*)\s*\(`)
	exportFuncRe    = regexp.MustCompile(`export\s+(?:async\s+)?function\s+(\w+)`)
	exportClassRe   = regexp.MustCompile(`export\s+class\s+(\w+)`)
	exportInterfRe  = regexp.MustCompile(`export\s+interface\s+(\w+)`)
	exportTypeRe    = regexp.MustCompile(`export\s+type\s+(\w+)`)
)

// nodeRuleIDs are the rule IDs that produce nodes (symbol definitions).
var nodeRuleIDs = map[string]bool{
	"ts-function-def":  true,
	"ts-class-def":     true,
	"ts-interface-def": true,
	"ts-type-def":      true,
}

// edgeRuleIDs are the rule IDs that produce edges (relationships).
var edgeRuleIDs = map[string]bool{
	"ts-import":    true,
	"ts-call-expr": true,
}

// ParseMatches converts ast-grep matches into graph nodes and edges.
func ParseMatches(matches []AstGrepMatch, filePath string, language string) ParseResult {
	var result ParseResult

	for _, m := range matches {
		switch m.RuleID {
		case "ts-function-def":
			if node := parseFunctionDef(m, filePath, language); node != nil {
				result.Nodes = append(result.Nodes, *node)
			}

		case "ts-class-def":
			if node := parseClassDef(m, filePath, language); node != nil {
				result.Nodes = append(result.Nodes, *node)
			}

		case "ts-interface-def":
			if node := parseInterfaceDef(m, filePath, language); node != nil {
				result.Nodes = append(result.Nodes, *node)
			}

		case "ts-type-def":
			if node := parseTypeDef(m, filePath, language); node != nil {
				result.Nodes = append(result.Nodes, *node)
			}

		case "ts-export-stmt":
			// Export statements can contain function, class, interface, or type defs.
			// Parse the inner definition and mark as exported.
			if node := parseExportDef(m, filePath, language); node != nil {
				result.Nodes = append(result.Nodes, *node)
			}

		case "ts-import":
			edges := parseImport(m, filePath)
			result.Edges = append(result.Edges, edges...)

		case "ts-call-expr":
			if edge := parseCall(m, filePath); edge != nil {
				result.Edges = append(result.Edges, *edge)
			}
		}
	}

	return result
}

func parseFunctionDef(m AstGrepMatch, filePath string, language string) *graph.Node {
	match := funcNameRe.FindStringSubmatch(m.Text)
	if match == nil {
		return nil
	}

	exported := isExported(m)
	sig := extractFunctionSignature(m.Text)

	return &graph.Node{
		Name:      match[1],
		Kind:      "fn",
		FilePath:  filePath,
		LineStart: m.Range.Start.Line + 1, // Convert 0-indexed to 1-indexed.
		LineEnd:   m.Range.End.Line + 1,
		ColStart:  m.Range.Start.Column,
		ColEnd:    m.Range.End.Column,
		Exported:  exported,
		Language:  language,
		Signature: sig,
	}
}

func parseClassDef(m AstGrepMatch, filePath string, language string) *graph.Node {
	match := classNameRe.FindStringSubmatch(m.Text)
	if match == nil {
		return nil
	}

	return &graph.Node{
		Name:      match[1],
		Kind:      "class",
		FilePath:  filePath,
		LineStart: m.Range.Start.Line + 1,
		LineEnd:   m.Range.End.Line + 1,
		ColStart:  m.Range.Start.Column,
		ColEnd:    m.Range.End.Column,
		Exported:  isExported(m),
		Language:  language,
	}
}

func parseInterfaceDef(m AstGrepMatch, filePath string, language string) *graph.Node {
	match := interfaceNameRe.FindStringSubmatch(m.Text)
	if match == nil {
		return nil
	}

	return &graph.Node{
		Name:      match[1],
		Kind:      "interface",
		FilePath:  filePath,
		LineStart: m.Range.Start.Line + 1,
		LineEnd:   m.Range.End.Line + 1,
		ColStart:  m.Range.Start.Column,
		ColEnd:    m.Range.End.Column,
		Exported:  isExported(m),
		Language:  language,
	}
}

func parseTypeDef(m AstGrepMatch, filePath string, language string) *graph.Node {
	match := typeNameRe.FindStringSubmatch(m.Text)
	if match == nil {
		return nil
	}

	return &graph.Node{
		Name:      match[1],
		Kind:      "type",
		FilePath:  filePath,
		LineStart: m.Range.Start.Line + 1,
		LineEnd:   m.Range.End.Line + 1,
		ColStart:  m.Range.Start.Column,
		ColEnd:    m.Range.End.Column,
		Exported:  isExported(m),
		Language:  language,
	}
}

func parseExportDef(m AstGrepMatch, filePath string, language string) *graph.Node {
	text := m.Text

	// Try each export pattern.
	if match := exportFuncRe.FindStringSubmatch(text); match != nil {
		sig := extractFunctionSignature(text)
		return &graph.Node{
			Name:      match[1],
			Kind:      "fn",
			FilePath:  filePath,
			LineStart: m.Range.Start.Line + 1,
			LineEnd:   m.Range.End.Line + 1,
			ColStart:  m.Range.Start.Column,
			ColEnd:    m.Range.End.Column,
			Exported:  true,
			Language:  language,
			Signature: sig,
		}
	}

	if match := exportClassRe.FindStringSubmatch(text); match != nil {
		return &graph.Node{
			Name:      match[1],
			Kind:      "class",
			FilePath:  filePath,
			LineStart: m.Range.Start.Line + 1,
			LineEnd:   m.Range.End.Line + 1,
			ColStart:  m.Range.Start.Column,
			ColEnd:    m.Range.End.Column,
			Exported:  true,
			Language:  language,
		}
	}

	if match := exportInterfRe.FindStringSubmatch(text); match != nil {
		return &graph.Node{
			Name:      match[1],
			Kind:      "interface",
			FilePath:  filePath,
			LineStart: m.Range.Start.Line + 1,
			LineEnd:   m.Range.End.Line + 1,
			ColStart:  m.Range.Start.Column,
			ColEnd:    m.Range.End.Column,
			Exported:  true,
			Language:  language,
		}
	}

	if match := exportTypeRe.FindStringSubmatch(text); match != nil {
		return &graph.Node{
			Name:      match[1],
			Kind:      "type",
			FilePath:  filePath,
			LineStart: m.Range.Start.Line + 1,
			LineEnd:   m.Range.End.Line + 1,
			ColStart:  m.Range.Start.Column,
			ColEnd:    m.Range.End.Column,
			Exported:  true,
			Language:  language,
		}
	}

	return nil
}

func parseImport(m AstGrepMatch, filePath string) []ParsedEdge {
	text := m.Text

	// Extract "from" path.
	fromMatch := importFromRe.FindStringSubmatch(text)
	if fromMatch == nil {
		return nil
	}
	fromPath := fromMatch[1]

	// Extract imported names.
	namesMatch := importNamesRe.FindStringSubmatch(text)
	if namesMatch == nil {
		return nil
	}

	var edges []ParsedEdge
	names := strings.Split(namesMatch[1], ",")
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		// Handle "Name as Alias" — use the original name.
		parts := strings.Fields(name)
		originalName := parts[0]

		edges = append(edges, ParsedEdge{
			SourceName: "", // Will be resolved by the indexer (the importing file).
			TargetName: originalName,
			Kind:       "imports",
			FilePath:   filePath,
			Line:       m.Range.Start.Line + 1,
		})
		_ = fromPath // Available for path-based resolution if needed.
	}

	return edges
}

func parseCall(m AstGrepMatch, filePath string) *ParsedEdge {
	match := callNameRe.FindStringSubmatch(m.Text)
	if match == nil {
		return nil
	}

	calledName := match[1]
	// Skip common built-in / noise calls.
	if isBuiltinCall(calledName) {
		return nil
	}

	return &ParsedEdge{
		SourceName: "", // Will be resolved by the indexer.
		TargetName: calledName,
		Kind:       "calls",
		FilePath:   filePath,
		Line:       m.Range.Start.Line + 1,
	}
}

// isExported checks if the match text is preceded by "export" in the Lines field.
func isExported(m AstGrepMatch) bool {
	return strings.Contains(m.Lines, "export ")
}

// extractFunctionSignature extracts the parameter and return type signature.
func extractFunctionSignature(text string) string {
	// Find the part between the function name and the opening brace.
	idx := strings.Index(text, "(")
	if idx < 0 {
		return ""
	}
	braceIdx := strings.Index(text, "{")
	if braceIdx < 0 {
		braceIdx = len(text)
	}
	sig := strings.TrimSpace(text[idx:braceIdx])
	return sig
}

// isBuiltinCall returns true for common built-in calls that shouldn't be edges.
func isBuiltinCall(name string) bool {
	builtins := map[string]bool{
		"console.log":    true,
		"console.error":  true,
		"console.warn":   true,
		"console.info":   true,
		"parseInt":       true,
		"parseFloat":     true,
		"String":         true,
		"Number":         true,
		"Boolean":        true,
		"Array":          true,
		"Object":         true,
		"JSON.parse":     true,
		"JSON.stringify": true,
		"Date.now":       true,
		"Math.floor":     true,
		"Math.ceil":      true,
		"Math.round":     true,
		"Math.random":    true,
		"require":        true,
		"setTimeout":     true,
		"setInterval":    true,
		"clearTimeout":   true,
		"clearInterval":  true,
		"Promise.resolve": true,
		"Promise.reject":  true,
		"Promise.all":     true,
	}
	return builtins[name]
}
