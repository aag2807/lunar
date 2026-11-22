; Scopes, definitions and references, for local-aware highlighting and
; navigation in editors that use them.

[
  (chunk)
  (block)
  (function_declaration)
  (function_definition)
  (arrow_function)
  (class_method)
  (constructor_definition)
  (getter)
  (setter)
  (for_numeric_statement)
  (for_generic_statement)
] @local.scope

(typed_name name: (identifier) @local.definition.var)
(parameter name: (identifier) @local.definition.parameter)
(variable_declaration name: (identifier) @local.definition.function)
(function_declaration name: (function_name . (identifier) @local.definition.function .))
(class_declaration name: (identifier) @local.definition.type)
(interface_declaration name: (identifier) @local.definition.type)
(enum_declaration name: (identifier) @local.definition.type)
(type_alias name: (identifier) @local.definition.type)

(identifier) @local.reference
