package indexer_test

import (
	"testing"

	"github.com/01x-in/codeindex/internal/indexer"
	"github.com/stretchr/testify/assert"
)

func TestParseMatches_GoFunctionDef(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "func FormatName(u *User) string {\n\tif u == nil {\n\t\treturn \"\"\n\t}\n\treturn u.Name\n}",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 30, Column: 0}, End: indexer.Position{Line: 35, Column: 1}},
			File:   "/repo/pkg/models/user.go",
			Lines:  "func FormatName(u *User) string {\n\tif u == nil {\n\t\treturn \"\"\n\t}\n\treturn u.Name\n}",
			RuleID: "go-function-def",
		},
	}

	result := indexer.ParseMatches(matches, "pkg/models/user.go", "go")

	assert.Len(t, result.Nodes, 1)
	assert.Equal(t, "FormatName", result.Nodes[0].Name)
	assert.Equal(t, "fn", result.Nodes[0].Kind)
	assert.True(t, result.Nodes[0].Exported)
	assert.Equal(t, "go", result.Nodes[0].Language)
	assert.Equal(t, 31, result.Nodes[0].LineStart)
	assert.Contains(t, result.Nodes[0].Signature, "(u *User)")
}

func TestParseMatches_GoUnexportedFunction(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "func generateID() string {\n\treturn \"id-001\"\n}",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 55, Column: 0}, End: indexer.Position{Line: 57, Column: 1}},
			File:   "/repo/pkg/service/user_service.go",
			Lines:  "func generateID() string {\n\treturn \"id-001\"\n}",
			RuleID: "go-function-def",
		},
	}

	result := indexer.ParseMatches(matches, "pkg/service/user_service.go", "go")

	assert.Len(t, result.Nodes, 1)
	assert.Equal(t, "generateID", result.Nodes[0].Name)
	assert.False(t, result.Nodes[0].Exported)
}

func TestParseMatches_GoMethodDef(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "func (u *User) Validate() error {\n\tif u.Name == \"\" {\n\t\treturn ErrEmptyName\n\t}\n\treturn nil\n}",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 22, Column: 0}, End: indexer.Position{Line: 27, Column: 1}},
			File:   "/repo/pkg/models/user.go",
			Lines:  "func (u *User) Validate() error {\n\tif u.Name == \"\" {\n\t\treturn ErrEmptyName\n\t}\n\treturn nil\n}",
			RuleID: "go-method-def",
		},
	}

	result := indexer.ParseMatches(matches, "pkg/models/user.go", "go")

	assert.Len(t, result.Nodes, 1)
	assert.Equal(t, "Validate", result.Nodes[0].Name)
	assert.Equal(t, "fn", result.Nodes[0].Kind)
	assert.True(t, result.Nodes[0].Exported)
	assert.Equal(t, "User", result.Nodes[0].Scope, "receiver type should be set as scope")
}

func TestParseMatches_GoStructDef(t *testing.T) {
	// With type_spec rule, the match text is the type spec itself (no "type" prefix).
	matches := []indexer.AstGrepMatch{
		{
			Text:   "User struct {\n\tID    string\n\tName  string\n\tEmail string\n}",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 4, Column: 5}, End: indexer.Position{Line: 8, Column: 1}},
			File:   "/repo/pkg/models/user.go",
			Lines:  "type User struct {\n\tID    string\n\tName  string\n\tEmail string\n}",
			RuleID: "go-type-decl",
		},
	}

	result := indexer.ParseMatches(matches, "pkg/models/user.go", "go")

	assert.Len(t, result.Nodes, 1)
	assert.Equal(t, "User", result.Nodes[0].Name)
	assert.Equal(t, "class", result.Nodes[0].Kind)
	assert.True(t, result.Nodes[0].Exported)
}

func TestParseMatches_GoInterfaceDef(t *testing.T) {
	// With type_spec rule, the match text is the type spec itself (no "type" prefix).
	matches := []indexer.AstGrepMatch{
		{
			Text:   "Validatable interface {\n\tValidate() error\n}",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 18, Column: 5}, End: indexer.Position{Line: 20, Column: 1}},
			File:   "/repo/pkg/models/user.go",
			Lines:  "type Validatable interface {\n\tValidate() error\n}",
			RuleID: "go-type-decl",
		},
	}

	result := indexer.ParseMatches(matches, "pkg/models/user.go", "go")

	assert.Len(t, result.Nodes, 1)
	assert.Equal(t, "Validatable", result.Nodes[0].Name)
	assert.Equal(t, "interface", result.Nodes[0].Kind)
	assert.True(t, result.Nodes[0].Exported)
}

func TestParseMatches_GoImport(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "import (\n\t\"fmt\"\n\n\t\"example.com/testproject/pkg/service\"\n)",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 2, Column: 0}, End: indexer.Position{Line: 6, Column: 1}},
			File:   "/repo/main.go",
			Lines:  "import (\n\t\"fmt\"\n\n\t\"example.com/testproject/pkg/service\"\n)",
			RuleID: "go-import",
		},
	}

	result := indexer.ParseMatches(matches, "main.go", "go")

	assert.Len(t, result.Edges, 2)
	assert.Equal(t, "fmt", result.Edges[0].TargetName)
	assert.Equal(t, "imports", result.Edges[0].Kind)
	assert.Equal(t, "service", result.Edges[1].TargetName)
	assert.Equal(t, "imports", result.Edges[1].Kind)
}

func TestParseMatches_GoCallExpression(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "service.NewUserService()",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 9, Column: 8}, End: indexer.Position{Line: 9, Column: 31}},
			File:   "/repo/main.go",
			Lines:  "\tsvc := service.NewUserService()",
			RuleID: "go-call-expr",
		},
	}

	result := indexer.ParseMatches(matches, "main.go", "go")

	assert.Len(t, result.Edges, 1)
	assert.Equal(t, "service.NewUserService", result.Edges[0].TargetName)
	assert.Equal(t, "calls", result.Edges[0].Kind)
}

func TestParseMatches_GoBuiltinCallsFiltered(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "make(map[string]*models.User)",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 15, Column: 10}, End: indexer.Position{Line: 15, Column: 38}},
			File:   "/repo/pkg/service/user_service.go",
			Lines:  "\t\tusers: make(map[string]*models.User),",
			RuleID: "go-call-expr",
		},
		{
			Text:   "append(result, u)",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 35, Column: 13}, End: indexer.Position{Line: 35, Column: 30}},
			File:   "/repo/pkg/service/user_service.go",
			Lines:  "\t\t\tresult = append(result, u)",
			RuleID: "go-call-expr",
		},
	}

	result := indexer.ParseMatches(matches, "pkg/service/user_service.go", "go")

	assert.Len(t, result.Edges, 0, "Go built-in calls should be filtered out")
}

func TestParseMatches_GoExportDetection(t *testing.T) {
	// Lowercase function: unexported.
	unexported := []indexer.AstGrepMatch{
		{
			Text:   "func helper() {}",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 0, Column: 0}, End: indexer.Position{Line: 0, Column: 16}},
			RuleID: "go-function-def",
		},
	}
	r1 := indexer.ParseMatches(unexported, "test.go", "go")
	assert.Len(t, r1.Nodes, 1)
	assert.False(t, r1.Nodes[0].Exported)

	// Uppercase function: exported.
	exported := []indexer.AstGrepMatch{
		{
			Text:   "func Helper() {}",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 0, Column: 0}, End: indexer.Position{Line: 0, Column: 16}},
			RuleID: "go-function-def",
		},
	}
	r2 := indexer.ParseMatches(exported, "test.go", "go")
	assert.Len(t, r2.Nodes, 1)
	assert.True(t, r2.Nodes[0].Exported)
}

func TestParseMatches_GoTypeAlias(t *testing.T) {
	// With type_spec rule, text is "ID string" (no "type" prefix).
	matches := []indexer.AstGrepMatch{
		{
			Text:   "ID string",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 2, Column: 5}, End: indexer.Position{Line: 2, Column: 14}},
			Lines:  "type ID string",
			RuleID: "go-type-decl",
		},
	}
	result := indexer.ParseMatches(matches, "types.go", "go")
	assert.Len(t, result.Nodes, 1)
	assert.Equal(t, "ID", result.Nodes[0].Name)
	assert.Equal(t, "type", result.Nodes[0].Kind)
	assert.True(t, result.Nodes[0].Exported)
}

func TestParseMatches_GoGroupedTypeDecl(t *testing.T) {
	// Grouped type declaration: type ( A struct{}; B interface{} )
	// With type_spec rule, each spec is a separate match.
	matches := []indexer.AstGrepMatch{
		{
			Text:   "Config struct {\n\tHost string\n\tPort int\n}",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 3, Column: 1}, End: indexer.Position{Line: 6, Column: 1}},
			Lines:  "\tConfig struct {\n\t\tHost string\n\t\tPort int\n\t}",
			RuleID: "go-type-decl",
		},
		{
			Text:   "Logger interface {\n\tLog(msg string)\n}",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 7, Column: 1}, End: indexer.Position{Line: 9, Column: 1}},
			Lines:  "\tLogger interface {\n\t\tLog(msg string)\n\t}",
			RuleID: "go-type-decl",
		},
	}

	result := indexer.ParseMatches(matches, "types.go", "go")

	assert.Len(t, result.Nodes, 2)
	assert.Equal(t, "Config", result.Nodes[0].Name)
	assert.Equal(t, "class", result.Nodes[0].Kind)
	assert.True(t, result.Nodes[0].Exported)
	assert.Equal(t, "Logger", result.Nodes[1].Name)
	assert.Equal(t, "interface", result.Nodes[1].Kind)
	assert.True(t, result.Nodes[1].Exported)
}

func TestParseMatches_GoGenericFunction(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "func Map[T any, U any](slice []T, fn func(T) U) []U {\n\tresult := make([]U, len(slice))\n\tfor i, v := range slice {\n\t\tresult[i] = fn(v)\n\t}\n\treturn result\n}",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 10, Column: 0}, End: indexer.Position{Line: 16, Column: 1}},
			RuleID: "go-function-def",
		},
	}

	result := indexer.ParseMatches(matches, "generic.go", "go")

	assert.Len(t, result.Nodes, 1)
	assert.Equal(t, "Map", result.Nodes[0].Name)
	assert.Equal(t, "fn", result.Nodes[0].Kind)
	assert.True(t, result.Nodes[0].Exported)
}

func TestParseMatches_GoGenericType(t *testing.T) {
	// With type_spec: "Set[T comparable] struct { items map[T]struct{} }"
	matches := []indexer.AstGrepMatch{
		{
			Text:   "Set[T comparable] struct {\n\titems map[T]struct{}\n}",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 5, Column: 5}, End: indexer.Position{Line: 7, Column: 1}},
			RuleID: "go-type-decl",
		},
	}

	result := indexer.ParseMatches(matches, "generic.go", "go")

	assert.Len(t, result.Nodes, 1)
	assert.Equal(t, "Set", result.Nodes[0].Name)
	assert.Equal(t, "class", result.Nodes[0].Kind)
	assert.True(t, result.Nodes[0].Exported)
}
