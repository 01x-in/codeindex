package indexer

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/01x-in/codeindex/internal/graph"
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

	// Go-specific patterns.
	// Support generic declarations: func Map[T any](...), type Set[T comparable] struct{}
	goFuncNameRe   = regexp.MustCompile(`func\s+(\w+)\s*(?:\[|[\(])`)
	goMethodNameRe = regexp.MustCompile(`func\s+\([^)]+\)\s+(\w+)\s*(?:\[|[\(])`)
	goTypeNameRe   = regexp.MustCompile(`type\s+(\w+)\s*(?:\[.*?\]\s+)?`)
	goImportPathRe = regexp.MustCompile(`"([^"]+)"`)
	goCallNameRe   = regexp.MustCompile(`^(\w+(?:\.\w+)*)\s*(?:\[.*?\])?\s*\(`)

	// Go type discriminator patterns (support generics with type params).
	goStructRe    = regexp.MustCompile(`\bstruct\s*\{`)
	goInterfaceRe = regexp.MustCompile(`\binterface\s*\{`)

	// Python-specific patterns.
	pyFuncNameRe   = regexp.MustCompile(`def\s+(\w+)\s*\(`)
	pyClassNameRe  = regexp.MustCompile(`class\s+(\w+)`)
	pyImportNameRe = regexp.MustCompile(`^import\s+(\S+)`)
	pyFromImportRe = regexp.MustCompile(`from\s+(\S+)\s+import`)
	pyCallNameRe   = regexp.MustCompile(`^(\w+(?:\.\w+)*)\s*\(`)

	// Rust-specific patterns.
	// Support generic declarations: fn process<T>(...)
	rustFuncNameRe   = regexp.MustCompile(`fn\s+(\w+)\s*(?:<[^>]*)?\s*\(`)
	rustStructNameRe = regexp.MustCompile(`(?:pub\s+)?struct\s+(\w+)`)
	rustEnumNameRe   = regexp.MustCompile(`(?:pub\s+)?enum\s+(\w+)`)
	rustTraitNameRe  = regexp.MustCompile(`(?:pub\s+)?trait\s+(\w+)`)
	rustUsePathRe    = regexp.MustCompile(`use\s+([\w:]+)(?:::\{([^}]+)\})?`)
	rustCallNameRe   = regexp.MustCompile(`^(\w+(?:::\w+)*)\s*(?:::<[^>]*)?\s*\(`)
)

// nodeRuleIDs are the rule IDs that produce nodes (symbol definitions).
var nodeRuleIDs = map[string]bool{
	"ts-function-def":  true,
	"ts-class-def":     true,
	"ts-interface-def": true,
	"ts-type-def":      true,
	"go-function-def":  true,
	"go-method-def":    true,
	"go-type-decl":     true,
}

// edgeRuleIDs are the rule IDs that produce edges (relationships).
var edgeRuleIDs = map[string]bool{
	"ts-import":    true,
	"ts-call-expr": true,
	"go-import":    true,
	"go-call-expr": true,
}

// ParseMatches converts ast-grep matches into graph nodes and edges.
func ParseMatches(matches []AstGrepMatch, filePath string, language string) ParseResult {
	var result ParseResult

	for _, m := range matches {
		switch m.RuleID {
		// TypeScript rules.
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

		// Go rules.
		case "go-function-def":
			if node := parseGoFunctionDef(m, filePath, language); node != nil {
				result.Nodes = append(result.Nodes, *node)
			}

		case "go-method-def":
			if node := parseGoMethodDef(m, filePath, language); node != nil {
				result.Nodes = append(result.Nodes, *node)
			}

		case "go-type-decl":
			// Handles type_spec nodes: individual type specs within grouped or standalone declarations.
			// Differentiates struct vs interface based on text content.
			nodes := parseGoTypeDecl(m, filePath, language)
			result.Nodes = append(result.Nodes, nodes...)

		case "go-import":
			edges := parseGoImport(m, filePath)
			result.Edges = append(result.Edges, edges...)

		case "go-call-expr":
			if edge := parseGoCall(m, filePath); edge != nil {
				result.Edges = append(result.Edges, *edge)
			}

		// Python rules.
		case "python-func-def":
			if node := parsePythonFuncDef(m, filePath, language); node != nil {
				result.Nodes = append(result.Nodes, *node)
			}

		case "python-class-def":
			if node := parsePythonClassDef(m, filePath, language); node != nil {
				result.Nodes = append(result.Nodes, *node)
			}

		case "python-import":
			edges := parsePythonImport(m, filePath)
			result.Edges = append(result.Edges, edges...)

		case "python-from-import":
			edges := parsePythonFromImport(m, filePath)
			result.Edges = append(result.Edges, edges...)

		case "python-call-expr":
			if edge := parsePythonCall(m, filePath); edge != nil {
				result.Edges = append(result.Edges, *edge)
			}

		// Rust rules.
		case "rust-func-def":
			if node := parseRustFuncDef(m, filePath, language); node != nil {
				result.Nodes = append(result.Nodes, *node)
			}

		case "rust-struct-def":
			if node := parseRustStructDef(m, filePath, language); node != nil {
				result.Nodes = append(result.Nodes, *node)
			}

		case "rust-enum-def":
			if node := parseRustEnumDef(m, filePath, language); node != nil {
				result.Nodes = append(result.Nodes, *node)
			}

		case "rust-trait-def":
			if node := parseRustTraitDef(m, filePath, language); node != nil {
				result.Nodes = append(result.Nodes, *node)
			}

		case "rust-use-stmt":
			edges := parseRustUse(m, filePath)
			result.Edges = append(result.Edges, edges...)

		case "rust-call-expr":
			if edge := parseRustCall(m, filePath); edge != nil {
				result.Edges = append(result.Edges, *edge)
			}
		}
	}

	return result
}

// --- TypeScript parsers ---

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

// --- Go parsers ---

func parseGoFunctionDef(m AstGrepMatch, filePath string, language string) *graph.Node {
	match := goFuncNameRe.FindStringSubmatch(m.Text)
	if match == nil {
		return nil
	}

	name := match[1]
	exported := isGoExported(name)
	sig := extractGoFunctionSignature(m.Text)

	return &graph.Node{
		Name:      name,
		Kind:      "fn",
		FilePath:  filePath,
		LineStart: m.Range.Start.Line + 1,
		LineEnd:   m.Range.End.Line + 1,
		ColStart:  m.Range.Start.Column,
		ColEnd:    m.Range.End.Column,
		Exported:  exported,
		Language:  language,
		Signature: sig,
	}
}

func parseGoMethodDef(m AstGrepMatch, filePath string, language string) *graph.Node {
	match := goMethodNameRe.FindStringSubmatch(m.Text)
	if match == nil {
		return nil
	}

	name := match[1]
	exported := isGoExported(name)
	sig := extractGoFunctionSignature(m.Text)

	// Extract receiver type for scope.
	scope := extractGoReceiver(m.Text)

	return &graph.Node{
		Name:      name,
		Kind:      "fn",
		FilePath:  filePath,
		LineStart: m.Range.Start.Line + 1,
		LineEnd:   m.Range.End.Line + 1,
		ColStart:  m.Range.Start.Column,
		ColEnd:    m.Range.End.Column,
		Exported:  exported,
		Language:  language,
		Signature: sig,
		Scope:     scope,
	}
}

// parseGoTypeDecl handles Go type_spec nodes. With the type_spec rule, each type in
// a grouped `type ( ... )` block is matched individually.
func parseGoTypeDecl(m AstGrepMatch, filePath string, language string) []graph.Node {
	// type_spec text looks like: "Name struct { ... }" or "Name[T any] interface { ... }" or "Name = int"
	// Extract the name (first word).
	text := strings.TrimSpace(m.Text)
	if text == "" {
		return nil
	}

	// Extract name: first contiguous word characters.
	nameRe := regexp.MustCompile(`^(\w+)`)
	nameMatch := nameRe.FindStringSubmatch(text)
	if nameMatch == nil {
		return nil
	}

	name := nameMatch[1]
	exported := isGoExported(name)

	// Determine the kind from text content.
	kind := "type" // default for type aliases
	if goStructRe.MatchString(text) {
		kind = "class" // struct -> class in the generic kind system
	} else if goInterfaceRe.MatchString(text) {
		kind = "interface"
	}

	return []graph.Node{
		{
			Name:      name,
			Kind:      kind,
			FilePath:  filePath,
			LineStart: m.Range.Start.Line + 1,
			LineEnd:   m.Range.End.Line + 1,
			ColStart:  m.Range.Start.Column,
			ColEnd:    m.Range.End.Column,
			Exported:  exported,
			Language:  language,
		},
	}
}

func parseGoImport(m AstGrepMatch, filePath string) []ParsedEdge {
	// Go imports are paths, not symbol names. Extract all quoted paths.
	paths := goImportPathRe.FindAllStringSubmatch(m.Text, -1)
	if len(paths) == 0 {
		return nil
	}

	var edges []ParsedEdge
	for _, p := range paths {
		importPath := p[1]
		// Use the last segment of the import path as the target name.
		parts := strings.Split(importPath, "/")
		targetName := parts[len(parts)-1]

		edges = append(edges, ParsedEdge{
			SourceName: "",
			TargetName: targetName,
			Kind:       "imports",
			FilePath:   filePath,
			Line:       m.Range.Start.Line + 1,
		})
	}

	return edges
}

func parseGoCall(m AstGrepMatch, filePath string) *ParsedEdge {
	match := goCallNameRe.FindStringSubmatch(m.Text)
	if match == nil {
		return nil
	}

	calledName := match[1]
	if isGoBuiltinCall(calledName) {
		return nil
	}

	return &ParsedEdge{
		SourceName: "",
		TargetName: calledName,
		Kind:       "calls",
		FilePath:   filePath,
		Line:       m.Range.Start.Line + 1,
	}
}

// --- Shared helpers ---

// isExported checks if the match text is preceded by "export" in the Lines field.
func isExported(m AstGrepMatch) bool {
	return strings.Contains(m.Lines, "export ")
}

// isGoExported checks if a Go identifier is exported (starts with uppercase).
func isGoExported(name string) bool {
	if len(name) == 0 {
		return false
	}
	return unicode.IsUpper(rune(name[0]))
}

// extractFunctionSignature extracts the parameter and return type signature (TypeScript).
func extractFunctionSignature(text string) string {
	start := strings.Index(text, "(")
	if start < 0 {
		return ""
	}

	paramEnd := findMatchingDelimiter(text, start, '(', ')')
	if paramEnd < 0 {
		return strings.TrimSpace(text[start:])
	}

	bodyStart := findFunctionBodyStart(text, paramEnd+1)
	if bodyStart < 0 {
		bodyStart = len(text)
	}

	return strings.TrimSpace(text[start:bodyStart])
}

func findMatchingDelimiter(text string, start int, open byte, close byte) int {
	depth := 0

	for i := start; i < len(text); i++ {
		switch text[i] {
		case '\'', '"', '`':
			var next int
			next, _ = skipQuotedLiteral(text, i)
			i = next - 1
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return i
			}
		}
	}

	return -1
}

func findFunctionBodyStart(text string, start int) int {
	parenDepth := 0
	bracketDepth := 0
	angleDepth := 0
	braceDepth := 0
	lastWord := ""
	lastToken := ""
	inReturnType := false

	for i := start; i < len(text); {
		ch := text[i]

		if ch == '\'' || ch == '"' || ch == '`' {
			next, _ := skipQuotedLiteral(text, i)
			i = next
			continue
		}

		if unicode.IsSpace(rune(ch)) {
			i++
			continue
		}

		if isIdentifierStart(ch) {
			j := i + 1
			for j < len(text) && isIdentifierPart(text[j]) {
				j++
			}
			lastWord = text[i:j]
			lastToken = lastWord
			i = j
			continue
		}

		switch ch {
		case ':':
			inReturnType = true
			lastToken = ":"
		case '(':
			parenDepth++
			lastToken = "("
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
			lastToken = ")"
		case '[':
			bracketDepth++
			lastToken = "["
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
			lastToken = "]"
		case '<':
			angleDepth++
			lastToken = "<"
		case '>':
			if lastToken == "=" {
				lastToken = "=>"
				i++
				continue
			}
			if angleDepth > 0 {
				angleDepth--
			}
			lastToken = ">"
		case '=':
			lastToken = "="
		case ',':
			lastToken = ","
		case '&':
			lastToken = "&"
		case '|':
			lastToken = "|"
		case '{':
			if parenDepth == 0 && bracketDepth == 0 && angleDepth == 0 && braceDepth == 0 {
				if !inReturnType || !startsTypeLiteral(lastToken, lastWord) {
					return i
				}
			}
			braceDepth++
			lastToken = "{"
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
			lastToken = "}"
		default:
			lastToken = string(ch)
		}

		i++
	}

	return -1
}

func startsTypeLiteral(lastToken string, lastWord string) bool {
	switch lastToken {
	case ":", "|", "&", ",", "=>", "=", "(", "[", "<":
		return true
	}

	switch lastWord {
	case "is", "extends", "infer":
		return true
	}

	return false
}

func skipQuotedLiteral(text string, start int) (int, bool) {
	quote := text[start]
	escaped := false

	for i := start + 1; i < len(text); i++ {
		ch := text[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == quote {
			return i + 1, true
		}
	}

	return len(text), false
}

func isIdentifierStart(ch byte) bool {
	r, _ := utf8.DecodeRune([]byte{ch})
	return ch == '_' || unicode.IsLetter(r)
}

func isIdentifierPart(ch byte) bool {
	r, _ := utf8.DecodeRune([]byte{ch})
	return ch == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// extractGoFunctionSignature extracts the Go function signature from the declaration text.
func extractGoFunctionSignature(text string) string {
	// Find "func" keyword position.
	funcIdx := strings.Index(text, "func")
	if funcIdx < 0 {
		return ""
	}

	rest := text[funcIdx+4:]

	// Skip receiver if present.
	rest = strings.TrimSpace(rest)
	if strings.HasPrefix(rest, "(") {
		// This is a receiver — find its closing paren.
		depth := 0
		for i, ch := range rest {
			if ch == '(' {
				depth++
			} else if ch == ')' {
				depth--
				if depth == 0 {
					rest = strings.TrimSpace(rest[i+1:])
					break
				}
			}
		}
	}

	// Skip the function name (and optional type params).
	nameEnd := strings.Index(rest, "(")
	if nameEnd < 0 {
		return ""
	}
	// Check for type params before the paren.
	bracketIdx := strings.Index(rest[:nameEnd], "[")
	if bracketIdx >= 0 {
		// Skip past the type params to find the actual param list.
		depth := 0
		for i := bracketIdx; i < len(rest); i++ {
			if rest[i] == '[' {
				depth++
			} else if rest[i] == ']' {
				depth--
				if depth == 0 {
					rest = rest[i+1:]
					break
				}
			}
		}
		nameEnd = strings.Index(rest, "(")
		if nameEnd < 0 {
			return ""
		}
	}
	rest = rest[nameEnd:]

	// Find the opening brace.
	braceIdx := strings.Index(rest, "{")
	if braceIdx < 0 {
		braceIdx = len(rest)
	}

	sig := strings.TrimSpace(rest[:braceIdx])
	return sig
}

// extractGoReceiver extracts the receiver type name from a method declaration.
func extractGoReceiver(text string) string {
	re := regexp.MustCompile(`func\s+\(\s*\w+\s+\*?(\w+)`)
	match := re.FindStringSubmatch(text)
	if match == nil {
		return ""
	}
	return match[1]
}

// isBuiltinCall returns true for common built-in calls that shouldn't be edges (TypeScript).
func isBuiltinCall(name string) bool {
	builtins := map[string]bool{
		"console.log":     true,
		"console.error":   true,
		"console.warn":    true,
		"console.info":    true,
		"parseInt":        true,
		"parseFloat":      true,
		"String":          true,
		"Number":          true,
		"Boolean":         true,
		"Array":           true,
		"Object":          true,
		"JSON.parse":      true,
		"JSON.stringify":  true,
		"Date.now":        true,
		"Math.floor":      true,
		"Math.ceil":       true,
		"Math.round":      true,
		"Math.random":     true,
		"require":         true,
		"setTimeout":      true,
		"setInterval":     true,
		"clearTimeout":    true,
		"clearInterval":   true,
		"Promise.resolve": true,
		"Promise.reject":  true,
		"Promise.all":     true,
	}
	return builtins[name]
}

// --- Python parsers ---

func parsePythonFuncDef(m AstGrepMatch, filePath string, language string) *graph.Node {
	match := pyFuncNameRe.FindStringSubmatch(m.Text)
	if match == nil {
		return nil
	}

	name := match[1]
	exported := !strings.HasPrefix(name, "_")

	return &graph.Node{
		Name:      name,
		Kind:      "fn",
		FilePath:  filePath,
		LineStart: m.Range.Start.Line + 1,
		LineEnd:   m.Range.End.Line + 1,
		ColStart:  m.Range.Start.Column,
		ColEnd:    m.Range.End.Column,
		Exported:  exported,
		Language:  language,
	}
}

func parsePythonClassDef(m AstGrepMatch, filePath string, language string) *graph.Node {
	match := pyClassNameRe.FindStringSubmatch(m.Text)
	if match == nil {
		return nil
	}

	name := match[1]
	exported := !strings.HasPrefix(name, "_")

	return &graph.Node{
		Name:      name,
		Kind:      "class",
		FilePath:  filePath,
		LineStart: m.Range.Start.Line + 1,
		LineEnd:   m.Range.End.Line + 1,
		ColStart:  m.Range.Start.Column,
		ColEnd:    m.Range.End.Column,
		Exported:  exported,
		Language:  language,
	}
}

func parsePythonImport(m AstGrepMatch, filePath string) []ParsedEdge {
	// "import foo" or "import foo.bar" or "import foo, bar"
	text := strings.TrimSpace(m.Text)
	// Strip "import " prefix and split by comma.
	withoutKeyword := strings.TrimPrefix(text, "import ")
	parts := strings.Split(withoutKeyword, ",")

	var edges []ParsedEdge
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		// For dotted imports (foo.bar), use the top-level module.
		segments := strings.Split(name, ".")
		targetName := segments[0]

		edges = append(edges, ParsedEdge{
			SourceName: "",
			TargetName: targetName,
			Kind:       "imports",
			FilePath:   filePath,
			Line:       m.Range.Start.Line + 1,
		})
	}
	return edges
}

func parsePythonFromImport(m AstGrepMatch, filePath string) []ParsedEdge {
	// "from foo import bar, baz" or "from foo.bar import Baz"
	text := strings.TrimSpace(m.Text)

	fromMatch := pyFromImportRe.FindStringSubmatch(text)
	if fromMatch == nil {
		return nil
	}
	_ = fromMatch[1] // module path — available for future resolution

	// Extract imported names after "import".
	importIdx := strings.Index(text, " import ")
	if importIdx < 0 {
		return nil
	}
	namesPart := strings.TrimSpace(text[importIdx+8:])
	// Strip parentheses if present.
	namesPart = strings.Trim(namesPart, "()")

	var edges []ParsedEdge
	for _, name := range strings.Split(namesPart, ",") {
		name = strings.TrimSpace(name)
		if name == "" || name == "*" {
			continue
		}
		// Handle "Name as Alias".
		parts := strings.Fields(name)
		targetName := parts[0]

		edges = append(edges, ParsedEdge{
			SourceName: "",
			TargetName: targetName,
			Kind:       "imports",
			FilePath:   filePath,
			Line:       m.Range.Start.Line + 1,
		})
	}
	return edges
}

func parsePythonCall(m AstGrepMatch, filePath string) *ParsedEdge {
	match := pyCallNameRe.FindStringSubmatch(m.Text)
	if match == nil {
		return nil
	}

	calledName := match[1]
	if isPythonBuiltinCall(calledName) {
		return nil
	}

	return &ParsedEdge{
		SourceName: "",
		TargetName: calledName,
		Kind:       "calls",
		FilePath:   filePath,
		Line:       m.Range.Start.Line + 1,
	}
}

// isPythonBuiltinCall returns true for Python built-in calls that shouldn't be edges.
func isPythonBuiltinCall(name string) bool {
	builtins := map[string]bool{
		"print":      true,
		"len":        true,
		"range":      true,
		"str":        true,
		"int":        true,
		"float":      true,
		"bool":       true,
		"list":       true,
		"dict":       true,
		"set":        true,
		"tuple":      true,
		"type":       true,
		"isinstance": true,
		"issubclass": true,
		"hasattr":    true,
		"getattr":    true,
		"setattr":    true,
		"super":      true,
		"enumerate":  true,
		"zip":        true,
		"map":        true,
		"filter":     true,
		"sorted":     true,
		"reversed":   true,
		"open":       true,
		"repr":       true,
		"vars":       true,
		"dir":        true,
		"id":         true,
	}
	return builtins[name]
}

// --- Rust parsers ---

func parseRustFuncDef(m AstGrepMatch, filePath string, language string) *graph.Node {
	match := rustFuncNameRe.FindStringSubmatch(m.Text)
	if match == nil {
		return nil
	}

	name := match[1]
	exported := strings.Contains(m.Text, "pub ")

	return &graph.Node{
		Name:      name,
		Kind:      "fn",
		FilePath:  filePath,
		LineStart: m.Range.Start.Line + 1,
		LineEnd:   m.Range.End.Line + 1,
		ColStart:  m.Range.Start.Column,
		ColEnd:    m.Range.End.Column,
		Exported:  exported,
		Language:  language,
	}
}

func parseRustStructDef(m AstGrepMatch, filePath string, language string) *graph.Node {
	match := rustStructNameRe.FindStringSubmatch(m.Text)
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
		Exported:  strings.Contains(m.Text, "pub "),
		Language:  language,
	}
}

func parseRustEnumDef(m AstGrepMatch, filePath string, language string) *graph.Node {
	match := rustEnumNameRe.FindStringSubmatch(m.Text)
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
		Exported:  strings.Contains(m.Text, "pub "),
		Language:  language,
	}
}

func parseRustTraitDef(m AstGrepMatch, filePath string, language string) *graph.Node {
	match := rustTraitNameRe.FindStringSubmatch(m.Text)
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
		Exported:  strings.Contains(m.Text, "pub "),
		Language:  language,
	}
}

func parseRustUse(m AstGrepMatch, filePath string) []ParsedEdge {
	match := rustUsePathRe.FindStringSubmatch(m.Text)
	if match == nil {
		return nil
	}

	line := m.Range.Start.Line + 1

	// Grouped import: use crate::module::{Name1, Name2}
	if match[2] != "" {
		var edges []ParsedEdge
		for _, item := range strings.Split(match[2], ",") {
			name := strings.TrimSpace(item)
			// Handle aliased imports: SomeName as Alias
			if idx := strings.Index(name, " as "); idx >= 0 {
				name = strings.TrimSpace(name[idx+4:])
			}
			if name != "" && name != "_" {
				edges = append(edges, ParsedEdge{
					SourceName: "",
					TargetName: name,
					Kind:       "imports",
					FilePath:   filePath,
					Line:       line,
				})
			}
		}
		return edges
	}

	// Simple import: use crate::module::Name
	path := match[1]
	parts := strings.Split(path, "::")
	targetName := parts[len(parts)-1]
	if targetName == "" {
		targetName = path
	}

	return []ParsedEdge{
		{
			SourceName: "",
			TargetName: targetName,
			Kind:       "imports",
			FilePath:   filePath,
			Line:       line,
		},
	}
}

func parseRustCall(m AstGrepMatch, filePath string) *ParsedEdge {
	match := rustCallNameRe.FindStringSubmatch(m.Text)
	if match == nil {
		return nil
	}

	calledName := match[1]
	if isRustBuiltinCall(calledName) {
		return nil
	}

	return &ParsedEdge{
		SourceName: "",
		TargetName: calledName,
		Kind:       "calls",
		FilePath:   filePath,
		Line:       m.Range.Start.Line + 1,
	}
}

// isRustBuiltinCall returns true for Rust built-in/macro calls that shouldn't be edges.
func isRustBuiltinCall(name string) bool {
	builtins := map[string]bool{
		"println":       true,
		"print":         true,
		"eprintln":      true,
		"eprint":        true,
		"vec":           true,
		"format":        true,
		"assert":        true,
		"assert_eq":     true,
		"assert_ne":     true,
		"panic":         true,
		"todo":          true,
		"unimplemented": true,
		"unreachable":   true,
		"dbg":           true,
		"write":         true,
		"writeln":       true,
		"Some":          true,
		"None":          true,
		"Ok":            true,
		"Err":           true,
	}
	return builtins[name]
}

// isGoBuiltinCall returns true for Go built-in/noise calls that shouldn't be edges.
func isGoBuiltinCall(name string) bool {
	builtins := map[string]bool{
		"make":    true,
		"len":     true,
		"cap":     true,
		"append":  true,
		"copy":    true,
		"delete":  true,
		"close":   true,
		"panic":   true,
		"recover": true,
		"print":   true,
		"println": true,
		"new":     true,
		"complex": true,
		"real":    true,
		"imag":    true,
		"error":   true,
	}
	return builtins[name]
}
