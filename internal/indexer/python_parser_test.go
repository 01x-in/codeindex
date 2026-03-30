package indexer_test

import (
	"testing"

	"github.com/01x/codeindex/internal/indexer"
	"github.com/stretchr/testify/assert"
)

func TestParsePythonFunc(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "def create_user(name: str, email: str) -> \"User\":\n    return User(id=\"\", name=name, email=email)",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 19, Column: 0}, End: indexer.Position{Line: 20, Column: 42}},
			File:   "/repo/src/models.py",
			Lines:  "def create_user(name: str, email: str) -> \"User\":\n    return User(id=\"\", name=name, email=email)",
			RuleID: "python-func-def",
		},
	}

	result := indexer.ParseMatches(matches, "src/models.py", "python")

	assert.Len(t, result.Nodes, 1)
	assert.Equal(t, "create_user", result.Nodes[0].Name)
	assert.Equal(t, "fn", result.Nodes[0].Kind)
	assert.True(t, result.Nodes[0].Exported)
	assert.Equal(t, "python", result.Nodes[0].Language)
	assert.Equal(t, 20, result.Nodes[0].LineStart)
}

func TestParsePythonClass(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "class User:\n    id: str\n    name: str",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 5, Column: 0}, End: indexer.Position{Line: 10, Column: 0}},
			File:   "/repo/src/models.py",
			Lines:  "class User:\n    id: str\n    name: str",
			RuleID: "python-class-def",
		},
	}

	result := indexer.ParseMatches(matches, "src/models.py", "python")

	assert.Len(t, result.Nodes, 1)
	assert.Equal(t, "User", result.Nodes[0].Name)
	assert.Equal(t, "class", result.Nodes[0].Kind)
	assert.True(t, result.Nodes[0].Exported)
	assert.Equal(t, "python", result.Nodes[0].Language)
}

func TestParsePythonImport(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "import uuid",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 0, Column: 0}, End: indexer.Position{Line: 0, Column: 11}},
			File:   "/repo/src/utils.py",
			Lines:  "import uuid",
			RuleID: "python-import",
		},
	}

	result := indexer.ParseMatches(matches, "src/utils.py", "python")

	assert.Len(t, result.Edges, 1)
	assert.Equal(t, "uuid", result.Edges[0].TargetName)
	assert.Equal(t, "imports", result.Edges[0].Kind)
}

func TestParsePythonImport_MultipleModules(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "import os, sys",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 0, Column: 0}, End: indexer.Position{Line: 0, Column: 14}},
			File:   "/repo/src/utils.py",
			Lines:  "import os, sys",
			RuleID: "python-import",
		},
	}

	result := indexer.ParseMatches(matches, "src/utils.py", "python")

	assert.Len(t, result.Edges, 2)
	assert.Equal(t, "os", result.Edges[0].TargetName)
	assert.Equal(t, "sys", result.Edges[1].TargetName)
}

func TestParsePythonFromImport(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "from src.models import User, Product, create_user",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 0, Column: 0}, End: indexer.Position{Line: 0, Column: 49}},
			File:   "/repo/src/service.py",
			Lines:  "from src.models import User, Product, create_user",
			RuleID: "python-from-import",
		},
	}

	result := indexer.ParseMatches(matches, "src/service.py", "python")

	assert.GreaterOrEqual(t, len(result.Edges), 3)
	names := make([]string, len(result.Edges))
	for i, e := range result.Edges {
		names[i] = e.TargetName
		assert.Equal(t, "imports", e.Kind)
	}
	assert.Contains(t, names, "User")
	assert.Contains(t, names, "Product")
	assert.Contains(t, names, "create_user")
}

func TestParsePythonCall(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "generate_id()",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 10, Column: 18}, End: indexer.Position{Line: 10, Column: 31}},
			File:   "/repo/src/service.py",
			Lines:  "        user_id = generate_id()",
			RuleID: "python-call-expr",
		},
	}

	result := indexer.ParseMatches(matches, "src/service.py", "python")

	assert.Len(t, result.Edges, 1)
	assert.Equal(t, "generate_id", result.Edges[0].TargetName)
	assert.Equal(t, "calls", result.Edges[0].Kind)
}

func TestParsePythonExportDetection(t *testing.T) {
	t.Run("public function is exported", func(t *testing.T) {
		matches := []indexer.AstGrepMatch{
			{
				Text:   "def generate_id() -> str:\n    return str(uuid.uuid4())",
				Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 6, Column: 0}, End: indexer.Position{Line: 7, Column: 28}},
				RuleID: "python-func-def",
			},
		}
		result := indexer.ParseMatches(matches, "src/utils.py", "python")
		assert.Len(t, result.Nodes, 1)
		assert.Equal(t, "generate_id", result.Nodes[0].Name)
		assert.True(t, result.Nodes[0].Exported)
	})

	t.Run("private function (underscore prefix) is not exported", func(t *testing.T) {
		matches := []indexer.AstGrepMatch{
			{
				Text:   "def _internal_helper(x):\n    return x",
				Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 22, Column: 0}, End: indexer.Position{Line: 23, Column: 12}},
				RuleID: "python-func-def",
			},
		}
		result := indexer.ParseMatches(matches, "src/models.py", "python")
		assert.Len(t, result.Nodes, 1)
		assert.Equal(t, "_internal_helper", result.Nodes[0].Name)
		assert.False(t, result.Nodes[0].Exported)
	})

	t.Run("private class (underscore prefix) is not exported", func(t *testing.T) {
		matches := []indexer.AstGrepMatch{
			{
				Text:   "class _InternalBase:\n    pass",
				Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 0, Column: 0}, End: indexer.Position{Line: 1, Column: 8}},
				RuleID: "python-class-def",
			},
		}
		result := indexer.ParseMatches(matches, "src/models.py", "python")
		assert.Len(t, result.Nodes, 1)
		assert.Equal(t, "_InternalBase", result.Nodes[0].Name)
		assert.False(t, result.Nodes[0].Exported)
	})

	t.Run("public class is exported", func(t *testing.T) {
		matches := []indexer.AstGrepMatch{
			{
				Text:   "class UserService:\n    pass",
				Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 5, Column: 0}, End: indexer.Position{Line: 6, Column: 8}},
				RuleID: "python-class-def",
			},
		}
		result := indexer.ParseMatches(matches, "src/service.py", "python")
		assert.Len(t, result.Nodes, 1)
		assert.True(t, result.Nodes[0].Exported)
	})
}

func TestParsePythonBuiltinCallsFiltered(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "print(\"hello\")",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 0, Column: 0}, End: indexer.Position{Line: 0, Column: 14}},
			RuleID: "python-call-expr",
		},
		{
			Text:   "len(items)",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 1, Column: 0}, End: indexer.Position{Line: 1, Column: 10}},
			RuleID: "python-call-expr",
		},
	}

	result := indexer.ParseMatches(matches, "src/utils.py", "python")
	assert.Len(t, result.Edges, 0, "Python built-in calls should be filtered out")
}
