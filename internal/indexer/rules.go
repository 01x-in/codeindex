package indexer

// TypeScriptRules is the inline rules YAML for TypeScript symbol extraction.
// Multiple rules are separated by "---" for use with --inline-rules.
const TypeScriptRules = `id: ts-function-def
language: TypeScript
rule:
  kind: function_declaration
---
id: ts-class-def
language: TypeScript
rule:
  kind: class_declaration
---
id: ts-interface-def
language: TypeScript
rule:
  kind: interface_declaration
---
id: ts-type-def
language: TypeScript
rule:
  kind: type_alias_declaration
---
id: ts-export-stmt
language: TypeScript
rule:
  kind: export_statement
---
id: ts-import
language: TypeScript
rule:
  kind: import_statement
---
id: ts-call-expr
language: TypeScript
rule:
  kind: call_expression`

// GoRules is the inline rules YAML for Go symbol extraction.
// Note: Go's tree-sitter grammar nests struct_type/interface_type inside
// type_spec inside type_declaration. We match type_declaration and
// differentiate struct vs interface in the parser via text inspection.
const GoRules = `id: go-function-def
language: Go
rule:
  kind: function_declaration
---
id: go-method-def
language: Go
rule:
  kind: method_declaration
---
id: go-type-decl
language: Go
rule:
  kind: type_declaration
---
id: go-import
language: Go
rule:
  kind: import_declaration
---
id: go-call-expr
language: Go
rule:
  kind: call_expression`

// LanguageRules maps language names to their inline rule strings.
var LanguageRules = map[string]string{
	"typescript": TypeScriptRules,
	"go":         GoRules,
}
