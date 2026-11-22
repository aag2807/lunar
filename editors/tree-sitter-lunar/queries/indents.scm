; Indentation for nvim-treesitter.

[
  (function_declaration)
  (function_definition)
  (class_declaration)
  (declare_class)
  (interface_declaration)
  (enum_declaration)
  (namespace_declaration)
  (declare_namespace)
  (class_method)
  (constructor_definition)
  (getter)
  (setter)
  (if_statement)
  (for_numeric_statement)
  (for_generic_statement)
  (while_statement)
  (repeat_statement)
  (do_statement)
  (match_expression)
  (table_constructor)
  (arguments)
  (parameters)
] @indent.begin

[
  "end"
  "until"
  "}"
  ")"
] @indent.end

[
  "end"
  "until"
  "else"
  "elseif"
  "}"
  ")"
] @indent.branch

(comment) @indent.auto
