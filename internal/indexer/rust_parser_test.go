package indexer_test

import (
	"testing"

	"github.com/01x-in/codeindex/internal/indexer"
	"github.com/stretchr/testify/assert"
)

func TestParseRustFunc(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "fn generate_id() -> String {\n    format!(\"{}\", 42)\n}",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 24, Column: 0}, End: indexer.Position{Line: 27, Column: 1}},
			File:   "/repo/src/service.rs",
			Lines:  "fn generate_id() -> String {",
			RuleID: "rust-func-def",
		},
	}

	result := indexer.ParseMatches(matches, "src/service.rs", "rust")

	assert.Len(t, result.Nodes, 1)
	assert.Equal(t, "generate_id", result.Nodes[0].Name)
	assert.Equal(t, "fn", result.Nodes[0].Kind)
	assert.False(t, result.Nodes[0].Exported, "fn without pub should not be exported")
	assert.Equal(t, "rust", result.Nodes[0].Language)
	assert.Equal(t, 25, result.Nodes[0].LineStart)
}

func TestParseRustStruct(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "pub struct User {\n    pub id: String,\n    pub name: String,\n}",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 1, Column: 0}, End: indexer.Position{Line: 5, Column: 1}},
			File:   "/repo/src/models.rs",
			Lines:  "pub struct User {",
			RuleID: "rust-struct-def",
		},
	}

	result := indexer.ParseMatches(matches, "src/models.rs", "rust")

	assert.Len(t, result.Nodes, 1)
	assert.Equal(t, "User", result.Nodes[0].Name)
	assert.Equal(t, "class", result.Nodes[0].Kind)
	assert.True(t, result.Nodes[0].Exported)
	assert.Equal(t, "rust", result.Nodes[0].Language)
}

func TestParseRustEnum(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "pub enum Status {\n    Active,\n    Inactive,\n}",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 14, Column: 0}, End: indexer.Position{Line: 17, Column: 1}},
			File:   "/repo/src/models.rs",
			Lines:  "pub enum Status {",
			RuleID: "rust-enum-def",
		},
	}

	result := indexer.ParseMatches(matches, "src/models.rs", "rust")

	assert.Len(t, result.Nodes, 1)
	assert.Equal(t, "Status", result.Nodes[0].Name)
	assert.Equal(t, "class", result.Nodes[0].Kind)
	assert.True(t, result.Nodes[0].Exported)
	assert.Equal(t, "rust", result.Nodes[0].Language)
}

func TestParseRustTrait(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "pub trait Repository {\n    fn find_by_id(&self, id: &str) -> Option<User>;\n}",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 19, Column: 0}, End: indexer.Position{Line: 22, Column: 1}},
			File:   "/repo/src/models.rs",
			Lines:  "pub trait Repository {",
			RuleID: "rust-trait-def",
		},
	}

	result := indexer.ParseMatches(matches, "src/models.rs", "rust")

	assert.Len(t, result.Nodes, 1)
	assert.Equal(t, "Repository", result.Nodes[0].Name)
	assert.Equal(t, "type", result.Nodes[0].Kind)
	assert.True(t, result.Nodes[0].Exported)
	assert.Equal(t, "rust", result.Nodes[0].Language)
}

func TestParseRustUse(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "use crate::models::User",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 0, Column: 0}, End: indexer.Position{Line: 0, Column: 23}},
			File:   "/repo/src/service.rs",
			Lines:  "use crate::models::User;",
			RuleID: "rust-use-stmt",
		},
	}

	result := indexer.ParseMatches(matches, "src/service.rs", "rust")

	assert.Len(t, result.Edges, 1)
	assert.Equal(t, "imports", result.Edges[0].Kind)
	assert.Equal(t, "src/service.rs", result.Edges[0].FilePath)
}

func TestParseRustExportDetection(t *testing.T) {
	t.Run("pub fn is exported", func(t *testing.T) {
		matches := []indexer.AstGrepMatch{
			{
				Text:   "pub fn new() -> Self {\n    UserService { users: HashMap::new() }\n}",
				Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 7, Column: 4}, End: indexer.Position{Line: 9, Column: 5}},
				RuleID: "rust-func-def",
			},
		}
		result := indexer.ParseMatches(matches, "src/service.rs", "rust")
		assert.Len(t, result.Nodes, 1)
		assert.Equal(t, "new", result.Nodes[0].Name)
		assert.True(t, result.Nodes[0].Exported)
	})

	t.Run("private fn is not exported", func(t *testing.T) {
		matches := []indexer.AstGrepMatch{
			{
				Text:   "fn generate_id() -> String {\n    format!(\"{}\", 42)\n}",
				Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 24, Column: 0}, End: indexer.Position{Line: 26, Column: 1}},
				RuleID: "rust-func-def",
			},
		}
		result := indexer.ParseMatches(matches, "src/service.rs", "rust")
		assert.Len(t, result.Nodes, 1)
		assert.Equal(t, "generate_id", result.Nodes[0].Name)
		assert.False(t, result.Nodes[0].Exported)
	})

	t.Run("private struct is not exported", func(t *testing.T) {
		matches := []indexer.AstGrepMatch{
			{
				Text:   "struct InternalState {\n    count: u32,\n}",
				Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 0, Column: 0}, End: indexer.Position{Line: 2, Column: 1}},
				RuleID: "rust-struct-def",
			},
		}
		result := indexer.ParseMatches(matches, "src/internal.rs", "rust")
		assert.Len(t, result.Nodes, 1)
		assert.Equal(t, "InternalState", result.Nodes[0].Name)
		assert.False(t, result.Nodes[0].Exported)
	})
}

func TestParseRustBuiltinCallsFiltered(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "println!(\"hello\")",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 0, Column: 4}, End: indexer.Position{Line: 0, Column: 21}},
			RuleID: "rust-call-expr",
		},
		{
			Text:   "vec![1, 2, 3]",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 1, Column: 4}, End: indexer.Position{Line: 1, Column: 17}},
			RuleID: "rust-call-expr",
		},
		{
			Text:   "format!(\"{}\", x)",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 2, Column: 4}, End: indexer.Position{Line: 2, Column: 20}},
			RuleID: "rust-call-expr",
		},
	}

	result := indexer.ParseMatches(matches, "src/main.rs", "rust")
	assert.Len(t, result.Edges, 0, "Rust macro/builtin calls should be filtered out")
}
