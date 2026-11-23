# Changelog

## 2.1.0 (2026-11-23)

### Added

- `async`/`await`, for scripts that pause (wait two seconds, wait for a
  texture): an async function may only be called with `await`, inside
  another async function, so nothing pauses outside a coroutine. They are
  checked, then erased: the Lua is plain coroutines, which the engine
  resumes. `async` goes before any function form, method, declaration or
  function type. An async function cannot go where a plain one is expected
  (a `table.sort` comparator), override or implement a plain method, or be a
  constructor, accessor, metamethod or `for` iterator. Errors have the code
  `async`; `not-supported` is no longer reported. Hover and signature help
  say `async`. `docs/ENGINE_INTEGRATION.md` describes what the engine does to
  resume scripts.

### Fixed

- Class fields' initial values are checked against their types; they were
  not checked at all (`x: number = "s"` and undefined names compiled).
- The language server crashed when completion was asked for inside a word
  (`pr|int`), and any request it failed on took it down; such a request is
  now answered with an error.
- `local m: { x: number } = ...` crashed the compiler; it now says that a
  shape is named with an interface.
- Functions defined into a table made with `local M = {}` (`function
  M.add(a: number)`) were not checked where they were called, and an
  `async function M.wait()` could be called without `await`. They keep
  their types now, in modules that import the table too. Scripts that
  called one with the wrong arguments get errors they did not get before.
- Plain Lua that was refused: clearing a field of a table made from a
  literal (`t.n = nil`), a field set to nil before it gets a value
  (`t.c = nil` then `t.c = 5`), and replacing a function with one that takes
  fewer parameters (`M.log = function() end`) or a vararg wrapper
  (`function(...)`). A function now fits where it is passed more arguments
  than it takes, as in Lua; it used to need exactly as many parameters.
- Type arguments after `-`, `#`, `not`, `^`, a comparison or `await` were
  read as comparisons: `await fetch<Texture>(p)` and `-id<number>(2)` did
  not compile.
- An interface's `static` methods could not be implemented: the class's
  static methods were not looked at. `static async` and `async static`
  were not accepted in interfaces, nor `static m<T>()`.
- `declare` overloads that mixed async and plain took async-ness from the
  first; they are refused now.
- Nested `match(` calls of a Lua function named `match` took twice as long
  per level: 20 levels took over a second, 40 would not finish.
- Decorators were not checked against what they decorate: `@deco` with
  `deco(f: () => void)` took an async function, or one needing arguments,
  and one returning nothing replaced the function with nil. Both ways are
  checked now, generic decorators (`same<T>(f: T): T`) included.
  Decorators on `M.f` and on methods were not checked at all, and `@deco
  function M.f()` decorated a global `f`; it replaces `M.f` now. `@deco
  local async function` did not parse.
- An override error was repeated for each ancestor and pointed at the
  class; it is reported once, on the method.
- An async member where an interface has a plain one (a class instance, a
  table literal's field, a table returned) was a `type` or `field-type`
  error; it is an `async` error, naming the member.
- `(self: T) => R`, `(type: T) => R`, `(n?: T) => R` and `(...: T) => R`
  did not parse as function types, and `[T]` gave a cascade of errors; it
  is one error saying an array is written `T[]`.
- Tree-sitter read `await(x)` and `await (x)`, calls of a Lua function named
  `await`, as awaits, and `local a = await` as an await of the next line.
  The grammar has a small external scanner (`src/scanner.c`) for it.

## 2.0.0 (2026-11-22)

Lunar is now one compiler that does one thing, tested by running the Lua it
produces under Lua 5.3 and 5.4 in CI. Earlier versions are in the git history;
their changelog described features that did not work.

### Projects

- `lunar build` compiles every script under `rootDir` in one run, writes the
  Lua to `outDir` only if nothing is wrong (staged and renamed into place, so
  the last good Lua stays when a script has an error), copies plain `.lua`
  scripts through, and removes outputs of deleted scripts. `--watch` rebuilds
  on change.
- `lunar.config.json`: `rootDir`, `outDir`, `target`, `declarations`,
  `copyLua`, `sourceMap`, `dataFiles`. Unknown keys are errors.
- Imports resolve from the importing file (`./`, `../`) or the root, form a
  module graph, and are checked across files; an unresolved import, a missing
  export or a cycle is an error. `require` names are relative to `rootDir`.
  `import * as m from "..."`.
- The standard library declarations are built into the binary, with `load`,
  `utf8`, and `warn` for `lua54`. `--decl` and `declarations` add shared
  declaration folders, such as an engine's API.
- Data files (`dataFiles`) are checked for anything a Lua state with no
  libraries opened lacks, string methods included.
- If an output cannot be replaced, the ones already replaced are put back:
  the folder always holds one whole build, and a build that was killed while
  writing is repaired by the next. `outDir` may not be `rootDir`, nor the
  `outDir` of another project.
- Compiling one file: flags may follow the file name, `-o` may be a device or
  pipe, and with no config only the declarations beside the file are read. A
  file outside its config's `rootDir` is reported as such.
- Two builds of one project at once wait for each other (they used to delete
  each other's outputs). A replaced file is never missing, even for a moment.
- Folders reached through symlinks are scanned (the root included), absolute
  paths in the config are used as given, and an `outDir` holding `rootDir`
  is refused. `lunar build` with no config anywhere says so at once.
- Declaration files are parsed once per build: a large engine API no longer
  costs a parse per script.
- `dataFiles` globs match as globs usually do, and one matching nothing is a
  warning. Imports of types generate nothing, so a data file may import an
  interface it shares with the game code; importing a value is an error.
- Source maps map every statement, nested ones included, and follow `-o`.
- Two modules with the same `require` name (`a.lunar`, `a/init.lunar`) are an
  error. A flag of the other mode (`lunar build -o`, `--out` for one file) is
  a usage error. `-o` may name the source no more.

### Errors

- `--json`: one JSON object per line on stdout (file, line, column, end,
  severity, stable code, message) and a closing summary; nothing on stderr.
- Exit codes: 0 compiled, 1 errors (nothing written), 2 usage or
  configuration, 3 I/O or internal.
- Every error names its own file and span, including errors in declaration
  files and the standard library.
- A table literal that does not fit its type names the field: wrong type,
  unknown field (with a suggestion), missing fields; nested and array fields
  by path.
- `lunar --version` prints `lunar 2.0.0 (<revision>)`, the same version the
  language server reports.
- Half-written code is a syntax error, never a crash or a hang: an unclosed
  string, template, long comment, table, block or `end` is reported where it
  starts, and a file that does not parse is not type checked.
- `--json=VALUE` counts from the first line; `<` and `>` are written as is.
- Messages print union types in a fixed order, and "did you mean"
  suggestions are the same from run to run.
- A recovered compiler crash exits 3. Lexical errors read as Lua's: an
  invalid escape, a malformed number (`5print`), an unexpected character
  (once, not once per byte).

### Language

- Plain Lua compiles unchanged, checked by running a corpus of Lua files
  both ways: Lua's operator precedence (`..` below `+`, `^` right
  associative and above unary minus, the bitwise levels); `f{...}` and
  `f"..."` calls anywhere; `goto` and labels, refused like Lua refuses them;
  `<const>` and `<close>`; `;` between table fields; `(f())` keeping one
  value; and every word Lunar adds (`from`, `match`, `class`, ...) usable
  as a name, since only Lua's reserved words are reserved.
- Lua's lexing exactly: every escape (`\ddd`, `\z`, `\x`, `\u{...}` to
  7FFFFFFF), strings as bytes (Latin-1 and binary strings survive), long
  strings and comments of any level, their first line break dropped and line
  endings normalised, numerals with a trailing point, a NUL byte, a UTF-8
  byte-order mark, a `#!` first line, CR-only line endings.
- More Lua compiles unchanged: bracketed keys (`{[KEY] = 1}`), parentheses
  around a callee or index (`(t.f or g)(2)`), `;` before `(`, a `(` starting
  a statement, long `elseif` chains (written flat), `(...)`, `... - 1`,
  `print(a < b, c > d)`, `local x` before a label, a label first in a
  function body, and `s.sub(s, 2, 3)`.
- More Lua is accepted: generic `for` with an expression list and any
  iterator function; `{}` taking any key and any later table; untyped
  functions called with any number of arguments; `local f` then
  `function f()`; locals and parameters shadowing library names; `local m =
  ...`; `cache = nil`; calls and operators on a table given a metatable;
  globals defined in a block; file handles' methods; a function returning
  itself (which crashed the compiler).
- A statement that is only an expression, not a call, is an error, as in
  Lua, and so is an assignment to anything but a name, a field or an index.
- `match` is typed (its value is what its arms give, bindings have the
  matched value's type, a literal pattern that can never match is an
  error) and takes alternatives (`| 1 | 2 ->`), dotted names such as enum
  members, and negative numbers.
- Default parameter values (`times: number = 1`), constraints on type
  parameters (`<T extends Named>`), and generic methods (`pass<U>(x: U)`).
  `extends Container<number>` instantiates the parent, and `Box<T>` inside
  `Box` follows the instance's arguments.
- `unknown` has to be narrowed before use; a string has only the string
  library's methods; `Pick`, `Omit` and `keyof` see inherited fields;
  `type T = typeof u` works for a local `u`.
- A subclass inherits its parent's metamethods (`__tostring`, `__eq`, ...).
- A subclass's field keeps the parent's type, a parent's `readonly` field
  is set only by the parent, `table.insert` is checked against a typed
  array, an assertion cannot turn one kind of value into another (`5 as
  string`), and decorators apply to `local function` too. `_ENV` is
  declared, and `ipairs`/`pairs` values are never nil.
- Code nested more than 200 levels deep, which Lua refuses to load ("C
  stack overflow"), is refused with that reason instead of being compiled
  slowly into Lua that does not load (2,000 nested blocks took 5 s,
  10,000 nested functions over two minutes). Functions inside tables and
  generic calls inside generic calls were exponential in depth (65 levels
  of `{ function() ... end }` never finished); they are linear.
- Standard library functions that return nil only to report failure
  (`io.open`, `os.getenv`, `math.type`, ...) are declared by their success
  type, as in Teal, so plain Lua using their results compiles.
- Untyped Lua is checked as Lua: `and`/`or` have the type of the operand they
  return; empty locals, tables grown field by field, left-out untyped
  parameters, values adjusted to names, and `function M.f()` /
  `function t:m()` are accepted. Annotated code stays strict.
- A script may define a function a declaration file declares; untyped
  parameters and return take the declared types.
- Repeated interface declarations merge. The standard library declarations
  gain `table.pack/unpack/move`, `string.pack/unpack/packsize/dump`,
  `math.tointeger/type/ult/maxinteger/mininteger`, `coroutine.isyieldable`
  (and `close` for lua54), `rawlen`, `collectgarbage`, and Lua's full
  signatures for `error`, `tonumber`, `next`, `loadfile` and `dofile`;
  `table.maxn` and the global `unpack`, absent from 5.3 and 5.4, are gone.
- Tables are written in source order; fields with no value write no line.

- A file may end in a top-level `return`.
- `field: T?` and `field?: T` may be left out of table literals; `field?: T`
  was being dropped by the parser.
- `Enemy.new(3)` constructs (it compiled to `Enemy:new(3)`); classes from
  other modules construct correctly.
- `declare function` needs no trailing `end`, which used to swallow the next
  declarations. `declare class` and `declare namespace` added. Interfaces may
  refer to themselves and to ones declared later.
- Conditions test truthiness, so `while true do` and `if x then` work, and
  narrow optionals.
- Every Lua numeral lexes: `0xFF`, `1e-9`, `.5`.
- Enum members count up from the previous value.
- Getters and setters work, including inherited ones.
- A subclass without a constructor inherits its parent's; abstract methods
  implemented by an intermediate class count.
- Exported functions are no longer globals.
- The checker alone decides `:` versus `.`; a call on an `any` value is
  emitted as written. Functions in a table literal are plain fields
  (`M.greet()`); a literal implementing an interface method takes `self`; a
  union of classes dispatches when its types agree; a method read without a
  call is an error.
- `m.Enemy(3)` through `import * as m`, and classes exported from a
  namespace, construct correctly.
- `local xs = {1, 2, 3}` is `number[]`; arithmetic works on unions of
  numbers; a class value fits a union that lists its class; a string index
  signature covers every undeclared name; `["a"] = 1` in a literal is the
  field `a`.
- Generic interfaces: `interface Box<T>`, used as `Box<number>`, also across
  modules.
- Refused with an error instead of miscompiled: implementation overloads,
  `async`/`await`, `import * from` without a name, statements after `return`.
- Targets are `lua53` and `lua54` only.

### Tools

- `lunar lsp`, the language server, rebuilt on the same pipeline as `lunar
  build`: diagnostics for every file, definition (into the standard library
  too), references and rename across files, hover, completion, signature help,
  document and workspace symbols. Each file uses the config found from its own
  folder; config errors are shown; rename is refused while a file does not
  parse, and includes table-literal keys. An edit re-checks only the edited
  file and what imports it (about 70 ms instead of 1 s for 200 scripts).
  Windows drive letters and `\\wsl$` paths are handled, and replies use the
  editor's own URIs.
- Completion while typing (from the text with the line being typed set
  aside), of imported names, string methods, plain tables' fields and a
  class's statics; not in comments or strings, and only types where a type
  goes. Definition, hover, references and rename on every kind of name.
  Rename refuses to change what a name refers to. A workspace folder with
  no config checks each project in it, and loose files on their own; every
  workspace folder counts; files are watched; the shutdown and exit
  lifecycle follows the protocol, and a malformed message is skipped
  rather than ending the server. Signature help ignores strings and
  comments and knows `Class.new(`, generic calls and `s:sub(`; hover shows
  `x?: T`, an instance's member types and setters' types; outline entries
  span their declarations and list constructors; untitled buffers are
  checked. The VS Code grammar treats Lunar's words as names where Lua
  would, and highlights `goto`, labels and `<close>`.
- A tree-sitter grammar (`editors/tree-sitter-lunar`), a Neovim plugin for
  `vim.lsp` and nvim-treesitter's main branch, a Zed extension, and an updated
  VS Code extension.

### Removed

WASM output, JIT hints, compiler plugins, the package manager and LuaRocks
commands, the bundler and `vendor/` libraries, the test runner, the REPL, the
formatter, linter, minifier and optimizers, the compile cache, `migrate`,
`docs`, project scaffolding, `lunar2decl`, `--no-typecheck`, and the
documentation for all of them.
