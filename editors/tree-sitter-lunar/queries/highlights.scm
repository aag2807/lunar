; Lunar highlights. Generic captures come first; later, more specific ones
; take precedence.

(identifier) @variable

((identifier) @variable.builtin
  (#eq? @variable.builtin "self"))

; Types
(type_identifier (identifier) @type)
(type_parameters (identifier) @type)
(type_predicate name: (identifier) @variable.parameter)
((type_identifier (identifier) @type.builtin)
  (#any-of? @type.builtin "number" "string" "boolean" "nil" "any" "void" "never" "unknown" "table"))

(class_declaration name: (identifier) @type.definition)
(declare_class name: (identifier) @type.definition)
(interface_declaration name: (identifier) @type.definition)
(enum_declaration name: (identifier) @type.definition)
(type_alias name: (identifier) @type.definition)
(namespace_declaration name: (identifier) @module)
(declare_namespace name: (identifier) @module)
(import_statement alias: (identifier) @module)

; Functions
(function_declaration name: (function_name (identifier) @function))
(function_declaration name: (function_name ":" (identifier) @function.method))
(declare_statement name: (function_name (identifier) @function))
(variable_declaration name: (identifier) @function)
(class_method name: (identifier) @function.method)
(abstract_method name: (identifier) @function.method)
(method_signature name: (identifier) @function.method)
(getter name: (identifier) @property)
(setter name: (identifier) @property)
"constructor" @constructor

(call function: (identifier) @function.call)
(call function: (dot_index field: (identifier) @function.method.call))
(generic_call function: (identifier) @function.call)
(method_call method: (identifier) @function.method.call)
((call function: (identifier) @function.builtin)
  (#any-of? @function.builtin
    "assert" "error" "getmetatable" "ipairs" "load" "loadfile" "next" "pairs" "pcall" "print"
    "rawequal" "rawget" "rawlen" "rawset" "require" "select" "setmetatable" "tonumber"
    "tostring" "type" "xpcall" "warn"))

; Parameters and fields
(parameter name: (identifier) @variable.parameter)
(dot_index field: (identifier) @variable.member)
(field name: (identifier) @variable.member)
(class_property name: (identifier) @variable.member)
(property_signature name: (identifier) @variable.member)
(object_type (identifier) @variable.member)
(table_pattern (identifier) @variable.member)
(enum_member name: (identifier) @constant)
(index_signature key: (identifier) @variable.parameter)

(decorator "@" @attribute name: (identifier) @attribute)
(match_arm pattern: (pattern "_" @variable.builtin))

; Literals
(comment) @comment
(hash_bang_line) @keyword.directive
(string) @string
(template_string) @string
(template_chars) @string
(escape_sequence) @string.escape
(template_substitution ["${" "}"] @punctuation.special)
(template_literal_type) @string
(number) @number
[(true) (false)] @boolean
(nil) @constant.builtin
(vararg_expression) @constant.builtin

; Keywords
[
  "local"
  "const"
  "end"
  "do"
  "then"
  "in"
  "with"
  "when"
  "as"
  "is"
  "keyof"
  "typeof"
  "get"
  "set"
] @keyword

"function" @keyword.function
"return" @keyword.return
["if" "elseif" "else" "match"] @keyword.conditional
["for" "while" "repeat" "until"] @keyword.repeat
(break_statement) @keyword.repeat
["import" "export" "from" "declare"] @keyword.import
["and" "or" "not"] @keyword.operator
["class" "interface" "enum" "type" "namespace" "extends" "implements"] @keyword.type
["async" "await"] @keyword.coroutine
["abstract" (modifier)] @keyword.modifier

; Operators and punctuation
[
  "=" "+=" "-=" "*=" "/=" "..=" "??="
  "+" "-" "*" "/" "//" "%" "^" "#"
  "==" "~=" "!=" "<" "<=" ">" ">="
  "&" "|" "~" "<<" ">>" ".." "??" "|>" "!"
  "=>" "->" "?" "?." "?[" "?.["
] @operator

["(" ")" "[" "]" "{" "}"] @punctuation.bracket
["," "." ":" ";"] @punctuation.delimiter
