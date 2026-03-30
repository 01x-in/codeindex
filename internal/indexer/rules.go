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
// We match type_spec (not type_declaration) so that grouped type blocks
// like `type ( A struct{}; B interface{} )` emit one match per type spec
// instead of one match for the whole block.
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
  kind: type_spec
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

// PythonRules is the inline rules YAML for Python symbol extraction.
const PythonRules = `id: python-func-def
language: Python
rule:
  kind: function_definition
---
id: python-class-def
language: Python
rule:
  kind: class_definition
---
id: python-import
language: Python
rule:
  kind: import_statement
---
id: python-from-import
language: Python
rule:
  kind: import_from_statement
---
id: python-call-expr
language: Python
rule:
  kind: call`

// LanguageRules maps language names to their inline rule strings.
var LanguageRules = map[string]string{
	"typescript": TypeScriptRules,
	"go":         GoRules,
	"python":     PythonRules,
}
