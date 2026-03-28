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

// LanguageRules maps language names to their inline rule strings.
var LanguageRules = map[string]string{
	"typescript": TypeScriptRules,
}
