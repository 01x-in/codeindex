package indexer_test

import (
	"testing"

	"github.com/01x-in/codeindex/internal/indexer"
	"github.com/stretchr/testify/assert"
)

func TestParseMatches_FunctionDef(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "function handleRequest(id: string): ResponsePayload {\n  const numId = parseId(id);\n}",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 12, Column: 7}, End: indexer.Position{Line: 16, Column: 1}},
			File:   "/repo/src/api/handler.ts",
			Lines:  "export function handleRequest(id: string): ResponsePayload {\n  const numId = parseId(id);\n}",
			RuleID: "ts-function-def",
		},
	}

	result := indexer.ParseMatches(matches, "src/api/handler.ts", "typescript")

	assert.Len(t, result.Nodes, 1)
	assert.Equal(t, "handleRequest", result.Nodes[0].Name)
	assert.Equal(t, "fn", result.Nodes[0].Kind)
	assert.Equal(t, 13, result.Nodes[0].LineStart) // 0-indexed line 12 -> 1-indexed 13
	assert.True(t, result.Nodes[0].Exported)       // "export" in Lines
	assert.Contains(t, result.Nodes[0].Signature, "(id: string)")
}

func TestParseMatches_ClassDef(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "class UserService {\n  getUser(id: number): User {}\n}",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 8, Column: 7}, End: indexer.Position{Line: 24, Column: 1}},
			File:   "/repo/src/models/user.ts",
			Lines:  "export class UserService {\n  getUser(id: number): User {}\n}",
			RuleID: "ts-class-def",
		},
	}

	result := indexer.ParseMatches(matches, "src/models/user.ts", "typescript")

	assert.Len(t, result.Nodes, 1)
	assert.Equal(t, "UserService", result.Nodes[0].Name)
	assert.Equal(t, "class", result.Nodes[0].Kind)
	assert.True(t, result.Nodes[0].Exported)
}

func TestParseMatches_InterfaceDef(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "interface Logger {\n  info(msg: string): void;\n}",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 13, Column: 7}, End: indexer.Position{Line: 16, Column: 1}},
			File:   "/repo/src/utils.ts",
			Lines:  "export interface Logger {\n  info(msg: string): void;\n}",
			RuleID: "ts-interface-def",
		},
	}

	result := indexer.ParseMatches(matches, "src/utils.ts", "typescript")

	assert.Len(t, result.Nodes, 1)
	assert.Equal(t, "Logger", result.Nodes[0].Name)
	assert.Equal(t, "interface", result.Nodes[0].Kind)
	assert.True(t, result.Nodes[0].Exported)
}

func TestParseMatches_TypeDef(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "type Config = {\n  port: number;\n  host: string;\n};",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 8, Column: 7}, End: indexer.Position{Line: 11, Column: 2}},
			File:   "/repo/src/utils.ts",
			Lines:  "export type Config = {\n  port: number;\n  host: string;\n};",
			RuleID: "ts-type-def",
		},
	}

	result := indexer.ParseMatches(matches, "src/utils.ts", "typescript")

	assert.Len(t, result.Nodes, 1)
	assert.Equal(t, "Config", result.Nodes[0].Name)
	assert.Equal(t, "type", result.Nodes[0].Kind)
	assert.True(t, result.Nodes[0].Exported)
}

func TestParseMatches_Import(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "import { formatDate, parseId } from '../utils';",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 0, Column: 0}, End: indexer.Position{Line: 0, Column: 47}},
			File:   "/repo/src/api/handler.ts",
			Lines:  "import { formatDate, parseId } from '../utils';",
			RuleID: "ts-import",
		},
	}

	result := indexer.ParseMatches(matches, "src/api/handler.ts", "typescript")

	assert.Len(t, result.Edges, 2)
	assert.Equal(t, "formatDate", result.Edges[0].TargetName)
	assert.Equal(t, "imports", result.Edges[0].Kind)
	assert.Equal(t, "parseId", result.Edges[1].TargetName)
	assert.Equal(t, "imports", result.Edges[1].Kind)
}

func TestParseMatches_CallExpression(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "formatDate(new Date())",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 14, Column: 14}, End: indexer.Position{Line: 14, Column: 36}},
			File:   "/repo/src/api/handler.ts",
			Lines:  "  const now = formatDate(new Date());",
			RuleID: "ts-call-expr",
		},
	}

	result := indexer.ParseMatches(matches, "src/api/handler.ts", "typescript")

	assert.Len(t, result.Edges, 1)
	assert.Equal(t, "formatDate", result.Edges[0].TargetName)
	assert.Equal(t, "calls", result.Edges[0].Kind)
}

func TestParseMatches_ExportStatement(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "export function formatDate(date: Date): string {\n  return date.toISOString();\n}",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 0, Column: 0}, End: indexer.Position{Line: 2, Column: 1}},
			File:   "/repo/src/utils.ts",
			Lines:  "export function formatDate(date: Date): string {\n  return date.toISOString();\n}",
			RuleID: "ts-export-stmt",
		},
	}

	result := indexer.ParseMatches(matches, "src/utils.ts", "typescript")

	assert.Len(t, result.Nodes, 1)
	assert.Equal(t, "formatDate", result.Nodes[0].Name)
	assert.Equal(t, "fn", result.Nodes[0].Kind)
	assert.True(t, result.Nodes[0].Exported)
}

func TestParseMatches_BuiltinCallsFiltered(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "parseInt(raw, 10)",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 5, Column: 9}, End: indexer.Position{Line: 5, Column: 26}},
			File:   "/repo/src/utils.ts",
			Lines:  "  return parseInt(raw, 10);",
			RuleID: "ts-call-expr",
		},
	}

	result := indexer.ParseMatches(matches, "src/utils.ts", "typescript")

	assert.Len(t, result.Edges, 0, "built-in calls should be filtered out")
}

func TestParseMatches_NonExportedFunction(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "function validateHeaders(headers: Record<string, string>): boolean {\n  return 'authorization' in headers;\n}",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 18, Column: 0}, End: indexer.Position{Line: 20, Column: 1}},
			File:   "/repo/src/api/handler.ts",
			Lines:  "function validateHeaders(headers: Record<string, string>): boolean {\n  return 'authorization' in headers;\n}",
			RuleID: "ts-function-def",
		},
	}

	result := indexer.ParseMatches(matches, "src/api/handler.ts", "typescript")

	assert.Len(t, result.Nodes, 1)
	assert.Equal(t, "validateHeaders", result.Nodes[0].Name)
	assert.False(t, result.Nodes[0].Exported, "private function should not be exported")
}

func TestParseMatches_FunctionDefWithGenericConstraintObjectType(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "export function omit<T extends { [key: string]: unknown }, K extends keyof T>(obj: T, key: K): Omit<T, K> {\n  return {} as Omit<T, K>\n}",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 40, Column: 0}, End: indexer.Position{Line: 42, Column: 1}},
			File:   "/repo/src/utils.ts",
			Lines:  "export function omit<T extends { [key: string]: unknown }, K extends keyof T>(obj: T, key: K): Omit<T, K> {\n  return {} as Omit<T, K>\n}",
			RuleID: "ts-function-def",
		},
	}

	assert.NotPanics(t, func() {
		result := indexer.ParseMatches(matches, "src/utils.ts", "typescript")

		assert.Len(t, result.Nodes, 1)
		assert.Equal(t, "omit", result.Nodes[0].Name)
		assert.Equal(t, "(obj: T, key: K): Omit<T, K>", result.Nodes[0].Signature)
	})
}

func TestParseMatches_FunctionDefWithDestructuredParamType(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "export default function Layout({ children }: { children: ReactNode }) {\n  return children\n}",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 9, Column: 0}, End: indexer.Position{Line: 11, Column: 1}},
			File:   "/repo/src/layout.tsx",
			Lines:  "export default function Layout({ children }: { children: ReactNode }) {\n  return children\n}",
			RuleID: "ts-function-def",
		},
	}

	result := indexer.ParseMatches(matches, "src/layout.tsx", "typescript")

	assert.Len(t, result.Nodes, 1)
	assert.Equal(t, "Layout", result.Nodes[0].Name)
	assert.Equal(t, "({ children }: { children: ReactNode })", result.Nodes[0].Signature)
}

func TestParseMatches_FunctionDefWithObjectTypePredicateReturn(t *testing.T) {
	matches := []indexer.AstGrepMatch{
		{
			Text:   "function isErrorLike(err: unknown): err is { message: string } {\n  return typeof err === 'object'\n}",
			Range:  indexer.AstGrepRange{Start: indexer.Position{Line: 17, Column: 0}, End: indexer.Position{Line: 19, Column: 1}},
			File:   "/repo/src/utils.ts",
			Lines:  "function isErrorLike(err: unknown): err is { message: string } {\n  return typeof err === 'object'\n}",
			RuleID: "ts-function-def",
		},
	}

	result := indexer.ParseMatches(matches, "src/utils.ts", "typescript")

	assert.Len(t, result.Nodes, 1)
	assert.Equal(t, "isErrorLike", result.Nodes[0].Name)
	assert.Equal(t, "(err: unknown): err is { message: string }", result.Nodes[0].Signature)
}
