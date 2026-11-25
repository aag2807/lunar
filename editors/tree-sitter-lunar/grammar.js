/**
 * Tree-sitter grammar for Lunar, a statically-typed superset of Lua.
 *
 * It is for editors: highlighting, folding, indentation and navigation. It
 * is deliberately more permissive than the compiler, which reports the
 * errors; what matters here is that every valid program parses without an
 * ERROR node, which CI checks against every .lunar file in the repository.
 */

const PREC = {
  PIPE: 1,
  NULLISH: 2,
  OR: 3,
  AND: 4,
  COMPARE: 5,
  BIT_OR: 6,
  BIT_XOR: 7,
  BIT_AND: 8,
  SHIFT: 9,
  CONCAT: 10,
  PLUS: 11,
  TIMES: 12,
  UNARY: 13,
  POWER: 14,
  AS: 15,
  CALL: 16,
};

const commaSep1 = (rule) => seq(rule, repeat(seq(',', rule)));
const commaSep = (rule) => optional(commaSep1(rule));
const trailingCommaSep1 = (rule) => seq(commaSep1(rule), optional(','));

// Long brackets, [[ ]] to [=====[ ]=====]: a body is anything that does
// not close its own level.
const LONG_LEVELS = [0, 1, 2, 3, 4, 5];

function longBracket(prefix, n) {
  const eq = '='.repeat(n);
  // A character that is not ']', or a ']' not followed by n '=' and ']'.
  const alts = ['[^\\]]'];
  for (let i = 0; i < n; i++) {
    alts.push('\\]' + '='.repeat(i) + '[^=]');
  }
  alts.push('\\]' + eq + '[^\\]]');
  return seq(prefix + '[' + eq + '[', new RegExp('(' + alts.join('|') + ')*'), ']' + eq + ']');
}

module.exports = grammar({
  name: 'lunar',

  extras: ($) => [/\s/, $.comment],

  word: ($) => $.identifier,

  // `await` followed by a name on the same line; see src/scanner.c.
  externals: ($) => [$._await],

  supertypes: ($) => [$._statement, $._expression, $._type],

  conflicts: ($) => [
    [$._type, $.conditional_type],
    [$.interface_declaration, $._identifier_like],
    [$.class_declaration, $._identifier_like],
    [$.modifier, $._identifier_like],
    [$._identifier_like, $.function_definition],
    [$._identifier_like, $.match_expression],
    [$._identifier_like, $.arrow_function],
    [$.export_statement, $._identifier_like],
    [$.import_statement, $._identifier_like],
    [$.variable_declaration, $._identifier_like],
    [$._expression, $.generic_call],
    [$.union_type],
    [$.variable_declaration, $.export_statement],
    [$.type_predicate, $._identifier_like],
    [$.parameter, $.type_identifier],
    [$.declare_class, $._identifier_like],
    [$.constructor_definition, $._identifier_like],
    [$.spread_element, $.vararg_expression],
    [$.parameter, $.vararg_expression],
    [$.parameter, $._prefix_expression],
    [$._identifier_like, $.function_type],
  ],

  rules: {
    chunk: ($) => seq(optional($.hash_bang_line), repeat($._statement)),

    hash_bang_line: (_) => /#![^\n]*/,

    // ---------------------------------------------------------------- statements

    _statement: ($) =>
      choice(
        ';',
        $.variable_declaration,
        $.assignment_statement,
        $.call_statement,
        $.do_statement,
        $.while_statement,
        $.repeat_statement,
        $.if_statement,
        $.for_numeric_statement,
        $.for_generic_statement,
        $.function_declaration,
        $.return_statement,
        $.break_statement,
        $.goto_statement,
        $.label_statement,
        $.class_declaration,
        $.interface_declaration,
        $.enum_declaration,
        $.type_alias,
        $.namespace_declaration,
        $.declare_statement,
        $.import_statement,
        $.export_statement,
        $.decorated_declaration,
        // await wait(1) on its own: it pauses, then carries on.
        $.await_expression,
      ),

    block: ($) => repeat1($._statement),

    variable_declaration: ($) =>
      prec.right(
        seq(
          choice('local', 'const', seq('local', 'const')),
          choice(
            seq(optional('async'), 'function', field('name', $.identifier), $._function_rest),
            seq(
              commaSep1($.typed_name),
              optional(seq('=', commaSep1($._expression))),
            ),
            seq(
              '(',
              commaSep1($.typed_name),
              ')',
              optional(seq('=', commaSep1($._expression))),
            ),
          ),
        ),
      ),

    typed_name: ($) =>
      seq(
        field('name', $._identifier_like),
        optional(field('attribute', $.attribute)),
        optional(seq(':', field('type', $._type))),
      ),

    // Lua 5.4: local x <const> = 5, local h <close> = v
    attribute: ($) => seq('<', field('name', $.identifier), '>'),

    assignment_statement: ($) =>
      seq(commaSep1($._assignable), choice('=', '+=', '-=', '*=', '/=', '..=', '??='), commaSep1($._expression)),

    _assignable: ($) => choice($._identifier_like, $.dot_index, $.bracket_index),

    call_statement: ($) => prec(-1, choice($.call, $.method_call, $.generic_call)),

    do_statement: ($) => seq('do', optional($.block), 'end'),

    while_statement: ($) =>
      seq('while', field('condition', $._expression), 'do', optional($.block), 'end'),

    repeat_statement: ($) =>
      seq('repeat', optional($.block), 'until', field('condition', $._expression)),

    if_statement: ($) =>
      seq(
        'if',
        field('condition', $._expression),
        'then',
        optional($.block),
        repeat($.elseif_clause),
        optional($.else_clause),
        'end',
      ),

    elseif_clause: ($) =>
      seq('elseif', field('condition', $._expression), 'then', optional($.block)),

    else_clause: ($) => seq('else', optional($.block)),

    for_numeric_statement: ($) =>
      seq(
        'for',
        $.typed_name,
        '=',
        $._expression,
        ',',
        $._expression,
        optional(seq(',', $._expression)),
        'do',
        optional($.block),
        'end',
      ),

    for_generic_statement: ($) =>
      seq('for', commaSep1($.typed_name), 'in', commaSep1($._expression), 'do', optional($.block), 'end'),

    function_declaration: ($) =>
      seq(optional('async'), 'function', field('name', $.function_name), $._function_rest),

    function_name: ($) =>
      seq(
        $._identifier_like,
        repeat(seq('.', $._identifier_like)),
        optional(seq(':', $._identifier_like)),
      ),

    _function_rest: ($) =>
      seq(
        optional($.type_parameters),
        field('parameters', $.parameters),
        optional(seq(':', field('return_type', $._return_type))),
        optional(field('body', $.block)),
        'end',
      ),

    _function_signature: ($) =>
      seq(
        optional($.type_parameters),
        field('parameters', $.parameters),
        optional(seq(':', field('return_type', $._return_type))),
      ),

    parameters: ($) => seq('(', optional(seq(commaSep1($.parameter), optional(','))), ')'),

    parameter: ($) =>
      choice(
        seq(
          repeat($.modifier),
          field('name', $._identifier_like),
          optional('?'),
          optional(seq(':', field('type', $._type))),
          optional(seq('=', field('default', $._expression))),
        ),
        seq('...', optional(field('name', $.identifier)), optional(seq(':', field('type', $._type)))),
      ),

    _return_type: ($) => choice($._type, $.type_predicate),

    type_predicate: ($) => seq(field('name', $.identifier), 'is', field('type', $._type)),

    return_statement: ($) =>
      prec.right(seq('return', optional(commaSep1($._expression)), optional(';'))),

    break_statement: (_) => 'break',

    goto_statement: ($) => seq('goto', field('label', $.identifier)),

    label_statement: ($) => seq('::', field('name', $.identifier), '::'),

    // ------------------------------------------------------------ declarations

    decorator: ($) =>
      seq('@', field('name', $._identifier_like), optional(field('arguments', $.arguments))),

    decorated_declaration: ($) =>
      // (a decorated local is a local function; the compiler refuses others)
      seq(repeat1($.decorator), choice($.function_declaration, $.class_declaration, $.variable_declaration)),

    class_declaration: ($) =>
      seq(
        optional('abstract'),
        'class',
        field('name', $.identifier),
        optional($.type_parameters),
        optional(seq('extends', field('superclass', $._type))),
        optional(seq('implements', commaSep1($._type))),
        repeat($._class_member),
        'end',
      ),

    _class_member: ($) =>
      choice(
        $.class_property,
        $.class_method,
        $.constructor_definition,
        $.getter,
        $.setter,
        $.index_signature,
        $.abstract_method,
      ),

    modifier: (_) => choice('public', 'private', 'protected', 'static', 'readonly'),

    // A property has a type or a value (as the compiler requires), so a
    // bare word after the class name is part of its header: extends Actor.
    class_property: ($) =>
      prec.right(
        seq(
          // @export before a field: called once, with (Class, name, info).
          repeat($.decorator),
          repeat($.modifier),
          field('name', $._identifier_like),
          optional('?'),
          choice(
            seq(':', field('type', $._type), optional(seq('=', field('value', $._expression)))),
            seq('=', field('value', $._expression)),
          ),
        ),
      ),

    class_method: ($) =>
      seq(
        repeat($.decorator),
        // static async and async static, as the compiler takes them.
        repeat($.modifier),
        optional(seq('async', repeat($.modifier))),
        optional('function'),
        field('name', $._identifier_like),
        $._function_rest,
      ),

    abstract_method: ($) =>
      prec.right(
        seq(
          repeat($.modifier),
          'abstract',
          repeat($.modifier),
          field('name', $._identifier_like),
          $._function_signature,
        ),
      ),

    constructor_definition: ($) =>
      seq(optional('async'), optional('function'), 'constructor', field('parameters', $.parameters), optional($.block), 'end'),

    // async on a constructor or accessor is the compiler's to refuse, with
    // its own message; parsing it keeps the rest of the class whole.
    getter: ($) =>
      seq(
        repeat($.modifier),
        optional('async'),
        'get',
        field('name', $._identifier_like),
        '(',
        ')',
        optional(seq(':', field('type', $._type))),
        optional($.block),
        'end',
      ),

    setter: ($) =>
      seq(
        repeat($.modifier),
        optional('async'),
        'set',
        field('name', $._identifier_like),
        field('parameters', $.parameters),
        optional($.block),
        'end',
      ),

    index_signature: ($) =>
      seq('[', field('key', $.identifier), ':', $._type, ']', ':', field('type', $._type)),

    interface_declaration: ($) =>
      seq(
        'interface',
        field('name', $.identifier),
        optional($.type_parameters),
        optional(seq('extends', commaSep1($._type))),
        repeat(choice($.property_signature, $.method_signature, $.index_signature)),
        'end',
      ),

    property_signature: ($) =>
      prec.right(
        seq(
          repeat($.modifier),
          field('name', $._identifier_like),
          optional('?'),
          ':',
          field('type', $._type),
          optional(','),
        ),
      ),

    method_signature: ($) =>
      prec.right(
        seq(
          repeat($.modifier),
          optional(seq('async', repeat($.modifier))),
          field('name', $._identifier_like),
          $._function_signature,
          optional(','),
        ),
      ),

    enum_declaration: ($) =>
      seq('enum', field('name', $.identifier), repeat($.enum_member), 'end'),

    enum_member: ($) =>
      prec.right(seq(field('name', $.identifier), optional(seq('=', field('value', $._expression))), optional(','))),

    type_alias: ($) =>
      prec.right(
        seq(
          'type',
          field('name', $.identifier),
          optional($.type_parameters),
          choice(seq('=', field('value', $._type)), seq(repeat($.property_signature), 'end')),
        ),
      ),

    namespace_declaration: ($) =>
      seq('namespace', field('name', $.identifier), repeat($._statement), 'end'),

    // `declare` describes Lua that Lunar did not compile: signatures only.
    declare_statement: ($) =>
      seq(
        'declare',
        choice(
          prec.right(seq(optional('async'), 'function', field('name', $.function_name), $._function_signature, optional('end'))),
          $.declare_class,
          $.interface_declaration,
          $.enum_declaration,
          $.type_alias,
          $.declare_namespace,
          seq(choice('local', 'const'), commaSep1($.typed_name)),
        ),
      ),

    declare_class: ($) =>
      seq(
        'class',
        field('name', $.identifier),
        optional($.type_parameters),
        optional(seq('extends', field('superclass', $._type))),
        optional(seq('implements', commaSep1($._type))),
        repeat(
          choice(
            $.property_signature,
            $.method_signature,
            seq('constructor', $.parameters),
            seq(repeat($.modifier), 'get', $._identifier_like, '(', ')', optional(seq(':', $._type))),
            seq(repeat($.modifier), 'set', $._identifier_like, $.parameters),
          ),
        ),
        'end',
      ),

    declare_namespace: ($) =>
      seq(
        'namespace',
        field('name', $.identifier),
        repeat(
          choice(
            prec.right(seq(optional('async'), 'function', field('name', $.identifier), $._function_signature)),
            seq(choice('local', 'const'), commaSep1($.typed_name)),
            $.interface_declaration,
            $.enum_declaration,
            $.type_alias,
            $.declare_class,
            $.declare_namespace,
          ),
        ),
        'end',
      ),

    import_statement: ($) =>
      seq(
        'import',
        choice(
          seq('{', optional(trailingCommaSep1($.identifier)), '}'),
          seq('*', 'as', field('alias', $.identifier)),
        ),
        'from',
        field('source', $.string),
      ),

    export_statement: ($) =>
      seq(
        'export',
        choice(
          $.variable_declaration,
          $.function_declaration,
          $.class_declaration,
          $.interface_declaration,
          $.enum_declaration,
          $.type_alias,
          $.namespace_declaration,
          $.decorated_declaration,
          seq(choice('const', 'local'), commaSep1($.typed_name), '=', commaSep1($._expression)),
          seq('{', optional(trailingCommaSep1($.identifier)), '}'),
        ),
      ),

    // --------------------------------------------------------------- expressions

    _expression: ($) =>
      choice(
        $.nil,
        $.true,
        $.false,
        $.number,
        $.string,
        $.template_string,
        $.vararg_expression,
        $.function_definition,
        $.arrow_function,
        $.table_constructor,
        $._prefix_expression,
        $.binary_expression,
        $.unary_expression,
        $.as_expression,
        $.match_expression,
        $.await_expression,
      ),

    _prefix_expression: ($) =>
      choice(
        $._identifier_like,
        $.dot_index,
        $.bracket_index,
        $.call,
        $.method_call,
        $.generic_call,
        $.parenthesized_expression,
      ),

    // Lunar's own keywords are ordinary names in Lua, and so wherever a
    // name may stand: local from = 1, t.class, s:match(p), f(public).
    _identifier_like: ($) =>
      choice(
        $.identifier,
        alias(
          choice('type', 'get', 'set', 'from', 'as', 'is', 'constructor',
            'match', 'with', 'when', 'class', 'interface', 'enum', 'namespace', 'declare', 'export', 'import',
            'static', 'public', 'private', 'protected', 'readonly', 'abstract', 'async', 'await', 'extends',
            'implements', 'keyof', 'typeof', 'const'),
          $.identifier,
        ),
      ),

    _field_name: ($) => $._identifier_like,

    parenthesized_expression: ($) => seq('(', $._expression, ')'),

    dot_index: ($) =>
      prec(PREC.CALL, seq(field('object', $._prefix_expression), choice('.', '?.'), field('field', $._field_name))),

    bracket_index: ($) =>
      prec(PREC.CALL, seq(field('object', $._prefix_expression), choice('[', '?[', '?.['), field('index', $._expression), ']')),

    call: ($) =>
      prec(PREC.CALL, seq(field('function', $._prefix_expression), optional('?.'), field('arguments', $.arguments))),

    // f<T>(x) against a < b > (x): both are tried; the call wins when both parse.
    generic_call: ($) =>
      prec.dynamic(1, seq(field('function', $._prefix_expression), $.type_arguments, field('arguments', $.arguments))),

    method_call: ($) =>
      prec(
        PREC.CALL,
        seq(
          field('object', $._prefix_expression),
          ':',
          field('method', $._field_name),
          field('arguments', $.arguments),
        ),
      ),

    arguments: ($) =>
      choice(
        seq('(', optional(seq(commaSep1(choice($._expression, $.spread_element)), optional(','))), ')'),
        $.table_constructor,
        $.string,
      ),

    spread_element: ($) => seq('...', $._expression),

    vararg_expression: (_) => '...',

    function_definition: ($) =>
      seq(optional('async'), 'function', $._function_rest),

    arrow_function: ($) =>
      prec.right(
        seq(
          optional('async'),
          field('parameters', $.parameters),
          optional(seq(':', field('return_type', $._type))),
          '=>',
          field('body', $._expression),
        ),
      ),

    table_constructor: ($) =>
      seq('{', optional(seq($.field, repeat(seq(choice(',', ';'), $.field)), optional(choice(',', ';')))), '}'),

    field: ($) =>
      choice(
        seq('[', field('key', $._expression), ']', '=', field('value', $._expression)),
        seq(field('name', $._field_name), '=', field('value', $._expression)),
        $.spread_element,
        field('value', $._expression),
      ),

    binary_expression: ($) =>
      choice(
        ...[
          ['|>', PREC.PIPE],
          ['??', PREC.NULLISH],
          ['or', PREC.OR],
          ['and', PREC.AND],
          ['<', PREC.COMPARE],
          ['<=', PREC.COMPARE],
          ['>', PREC.COMPARE],
          ['>=', PREC.COMPARE],
          ['==', PREC.COMPARE],
          ['~=', PREC.COMPARE],
          ['!=', PREC.COMPARE],
          ['|', PREC.BIT_OR],
          ['~', PREC.BIT_XOR],
          ['&', PREC.BIT_AND],
          ['<<', PREC.SHIFT],
          ['>>', PREC.SHIFT],
          ['+', PREC.PLUS],
          ['-', PREC.PLUS],
          ['*', PREC.TIMES],
          ['/', PREC.TIMES],
          ['//', PREC.TIMES],
          ['%', PREC.TIMES],
        ].map(([op, p]) =>
          prec.left(p, seq(field('left', $._expression), field('operator', op), field('right', $._expression))),
        ),
        prec.right(PREC.CONCAT, seq(field('left', $._expression), field('operator', '..'), field('right', $._expression))),
        prec.right(PREC.POWER, seq(field('left', $._expression), field('operator', '^'), field('right', $._expression))),
      ),

    unary_expression: ($) =>
      prec.left(PREC.UNARY, seq(field('operator', choice('not', '#', '-', '~', '!')), field('operand', $._expression))),

    as_expression: ($) =>
      prec.left(PREC.AS, seq($._expression, 'as', choice('const', $._type))),

    await_expression: ($) => prec.right(PREC.UNARY, seq(alias($._await, 'await'), $._expression)),

    match_expression: ($) =>
      seq('match', field('subject', $._expression), 'with', repeat1($.match_arm), 'end'),

    // An arm's value ends at the next arm's '|', which is therefore not
    // bitwise or; parenthesize to use one there.
    match_arm: ($) =>
      prec.left(
        PREC.BIT_OR + 1,
        seq(
          '|',
          field('pattern', $.pattern),
          // | 1 | 2 -> ...: alternatives, before the '->'
          repeat(seq('|', field('pattern', $.pattern))),
          optional(seq('when', field('guard', $._expression))),
          '->',
          field('value', $._expression),
        ),
      ),

    pattern: ($) =>
      choice(
        '_',
        $.number,
        seq('-', $.number),
        $.string,
        $.true,
        $.false,
        $.nil,
        $.identifier,
        // A type pattern names one type (the compiler takes a name), so a
        // following '|' is the next alternative, not a union.
        prec(1, seq(field('binding', $.identifier), ':', field('type', $._type_primary))),
        $.dot_index,
        $.table_pattern,
      ),

    table_pattern: ($) =>
      seq('{', optional(trailingCommaSep1(seq($._field_name, ':', $.pattern))), '}'),

    // --------------------------------------------------------------------- types

    _type: ($) =>
      choice(
        $._type_primary,
        $.union_type,
        $.intersection_type,
        $.function_type,
        $.conditional_type,
      ),

    _type_primary: ($) =>
      choice(
        $.type_identifier,
        $.generic_type,
        $.array_type,
        $.optional_type,
        $.parenthesized_type,
        $.tuple_type,
        $.literal_type,
        $.object_type,
        $.mapped_type,
        $.keyof_type,
        $.typeof_type,
        $.template_literal_type,
      ),

    type_identifier: ($) =>
      prec.right(seq($._identifier_like, repeat(seq('.', $._identifier_like)))),

    generic_type: ($) => prec(1, seq($.type_identifier, $.type_arguments)),

    type_arguments: ($) => seq('<', commaSep1($._type), '>'),

    type_parameters: ($) =>
      seq('<', commaSep1(seq($.identifier, optional(seq('extends', $._type)))), '>'),

    // '[]' is one token: a '[' followed by anything else after a type is
    // the next member, such as an index signature on the following line.
    array_type: ($) => prec(2, seq($._type_primary, token(seq('[', /\s*/, ']')))),

    optional_type: ($) => prec(2, seq($._type_primary, '?')),

    parenthesized_type: ($) => seq('(', $._type, ')'),

    tuple_type: ($) => seq('(', $._type, ',', commaSep1($._type), ')'),

    literal_type: ($) => choice($.string, $.number, seq('-', $.number), $.true, $.false, $.nil),

    object_type: ($) =>
      seq('{', optional(trailingCommaSep1(seq($._field_name, optional('?'), ':', $._type))), '}'),

    mapped_type: ($) => seq('{', '[', $.identifier, 'in', $._type, ']', ':', $._type, '}'),

    keyof_type: ($) => prec(3, seq('keyof', $._type_primary)),

    typeof_type: ($) => prec(3, seq('typeof', $._prefix_expression)),

    template_literal_type: ($) =>
      seq('`', repeat(choice($.template_chars, seq('${', $._type, '}'))), '`'),

    union_type: ($) => prec.left(1, seq(optional('|'), $._type, '|', $._type)),

    intersection_type: ($) => prec.left(2, seq($._type, '&', $._type)),

    function_type: ($) =>
      prec.right(
        choice(
          seq(
            optional('async'),
            'function',
            optional(seq(optional($.type_parameters), $.parameters, optional(seq(':', $._return_type)))),
          ),
          seq(optional('async'), optional($.type_parameters), $.parameters, '=>', $._return_type),
        ),
      ),

    conditional_type: ($) =>
      prec.right(seq($._type_primary, 'extends', $._type_primary, '?', $._type, ':', $._type)),

    // ------------------------------------------------------------------- tokens

    identifier: (_) => /[a-zA-Z_][a-zA-Z0-9_]*/,

    nil: (_) => 'nil',
    true: (_) => 'true',
    false: (_) => 'false',

    number: (_) =>
      token(
        choice(
          /0[xX][0-9a-fA-F]*(\.[0-9a-fA-F]*)?([pP][+-]?[0-9]+)?/,
          /[0-9]+(\.[0-9]*)?([eE][+-]?[0-9]+)?/,
          /\.[0-9]+([eE][+-]?[0-9]+)?/,
        ),
      ),

    string: (_) =>
      token(
        choice(
          // \z skips the whitespace after it, line breaks included; a
          // backslash keeps a line break, CRLF included.
          seq('"', repeat(choice(/[^"\\\n]+/, /\\(z[ \t\r\n\f\v]*|\r\n|.|\n)/)), '"'),
          seq("'", repeat(choice(/[^'\\\n]+/, /\\(z[ \t\r\n\f\v]*|\r\n|.|\n)/)), "'"),
          ...LONG_LEVELS.map((n) => longBracket('', n)),
        ),
      ),

    template_string: ($) =>
      seq('`', repeat(choice($.template_chars, $.escape_sequence, $.template_substitution)), '`'),

    template_chars: (_) => token.immediate(prec(1, /([^`$\\]|\$[^{`])+/)),

    escape_sequence: (_) => token.immediate(/\\./),

    template_substitution: ($) => seq('${', $._expression, '}'),

    comment: (_) =>
      token(
        choice(
          ...LONG_LEVELS.map((n) => longBracket('--', n)),
          seq('--', /[^\n]*/),
        ),
      ),
  },
});
