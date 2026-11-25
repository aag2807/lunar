# Status

What works, what does not, and how each is known. A line under **Works**
names the test that proves it; CI runs every one on Linux (Lua 5.3 and 5.4)
and Windows. Update this file in the same commit as the change it describes.

Last updated for **2.1.0**, plus what `CHANGELOG.md` lists as unreleased.

## Compiler

### Works

| What | Proved by |
|---|---|
| Plain Lua 5.3/5.4 compiles unchanged: the output prints what the original prints | `TestPlainLuaCompilesUnchanged`: 17 Lua files in `tests/luacompat`, each compiled and run both ways under 5.3 and 5.4 |
| The same for real game code | `TestGameScriptsBehaveTheSame`: 14 engine-style scripts in `tests/luacompat/game` |
| Lua's lexing: every escape, byte strings, long brackets of any level, numerals, CR/CRLF, BOM, `#!` | `tests/luacompat/lexing.lua`, `internal/lexer` tests |
| Lua's syntax: precedence, `f{}`/`f""` calls, goto/labels, `<const>`/`<close>`, generic `for` with any iterator, `(...)`, `local x` before a label | `tests/luacompat/{precedence,syntax,calls,idioms*}.lua` |
| Untyped Lua is checked as Lua (missing arguments are nil, `{}` takes any key, forward-declared locals, shadowed library names, fields cleared with nil or set to nil first, a function replaced by one with fewer parameters or a vararg wrapper) | `tests/luacompat/idioms2.lua`, `idioms3.lua`, `fields.lua` |
| Functions defined into a table made with `local M = {}` keep their types: their calls are checked, an async one must be awaited, in importing modules too | `tests/regress/run/empty_table_functions.lunar`, `tests/regress/refuse/empty_table_functions.lunar`, `TestImportedTableAsyncFunctionMustBeAwaited` |
| A function fits where it is passed more arguments than it takes, and where it is passed fewer if the rest may be left out; a vararg one takes any number | `tests/regress/run/function_params.lunar`, `tests/regress/refuse/function_params.lunar` |
| Typed Lunar: classes (inheritance, abstract, statics, accessors, metamethods), interfaces, enums, generics (constraints, methods, generic classes), `match`, namespaces, modules, utility types, default parameters, narrowing | `TestRegressions` (42 programs run under Lua, 34 that must be refused), `TestExamplesRun` (26 example programs), `TestGoldenOutput` |
| Every LANGUAGE.md example compiles (and those that run, run) | `TestDocumentationExamplesCompile` |
| Code nested more than Lua's 200 levels is refused with Lua's reason; deep code compiles in linear time, nested `async(` and `match(` calls of Lua functions too | `TestNestingLimit`, `TestNestedShapesCompileQuickly` |
| Type arguments after `-`, `#`, `not`, `^`, `==` and `await` (`await fetch<T>(p)`) | `tests/regress/run/generic_after_unary.lunar` |
| An interface's `static` methods (`static async` and `async static` too) are implemented by the class's static methods | `tests/regress/run/interface_static.lunar`, `tests/regress/refuse/interface_static.lunar` |
| `declare` overloads that mix async and plain are refused | `tests/regress/refuse/async_overloads.lunar` |
| Decorators are checked both ways: the target fits what the decorator takes, what it returns fits the target; on functions, `M.f`, methods (with self) and `local async function`; `@deco function M.f()` replaces `M.f` | `tests/regress/run/decorator_targets.lunar`, `tests/regress/refuse/decorator_targets.lunar` |
| Class fields take decorators: each is called once as the class is defined, `deco(Class, "name", info)` with the declared type and a literal default; field and function decorators are refused on each other's targets, naming both; a declared decorator's overloads are matched by its use | `tests/regress/run/field_decorators.lunar`, `tests/regress/refuse/field_decorators.lunar`, `examples/features/26_field_decorators.lunar` |
| An override is checked once, against the nearest ancestor with the method, and its error is on the method | `TestOverrideErrorsOnceAtTheMethod` |
| An async member where an interface has a plain one is an `async` error wherever the value goes (class instance, table field, argument, return) | `TestStructuralAsyncMismatchHasItsCode` |
| Arrow function types name parameters as functions do: `(self: T) =>`, `(type: T) =>`, `(n?: T) =>`, `(...: T) =>`; `[T]` is one error saying to write `T[]` | `tests/regress/run/arrow_type_params.lunar`, `TestArrowTypeParameterNames`, `TestBracketTypeIsOneError` |
| Class fields' initial values are checked against their types | `tests/regress/refuse/class_field_values.lunar` |
| `async`/`await`: an async function is only called with `await`, inside an async function; both are erased to plain coroutines | `examples/features/25_async.lunar` (a scheduler, run under 5.3 and 5.4), `tests/regress/run/async_types.lunar`, `tests/regress/refuse/async_*.lunar` (one per rule), `TestImportedAsyncFunctionMustBeAwaited`, `TestHoverShowsAsync` |
| No crash on half-written code | `test/integration` (incomplete code), `internal/parser/hang_test.go`, `internal/project/robust_test.go` |
| Fuzzing: 1,500 generated programs, against this release, printed the same compiled and not | the differential fuzzer (run by hand, not in CI) |

### Does not work, or not yet

| What | Notes |
|---|---|
| Targets other than Lua 5.3 and 5.4 (LuaJIT, 5.1, 5.2) | refused |
| Two implementations of one function (overloads) | refused; overloads belong in `declare` |
| Plain Lua `f(a < b, c > (d))` reads as a generic call, as in TypeScript | refused with "'a' is not a generic type or function"; write `c > d`. Kept on purpose |
| A local declared with a value keeps its type: `local x = 1` then `x = "s"` is refused, in plain Lua too | as in TypeScript; declare it `local x` or `local x: number \| string`. Kept on purpose |
| Destructuring a table's fields | only a call's results: `local (a, b) = f()` |
| Nested template strings, a backtick inside `${...}` | refused |
| `s.f(...)` on a string where the arguments do not fill `string.f`'s parameters is taken as `s:f(...)` | `s.rep("ab", 2)` means `s:rep("ab", 2)`; write `string.rep("ab", 2)` for the plain call |
| Runtime error lines are those of the generated Lua | source maps (`--source-map`) map them back |

## Standard library

### Works

| What | Proved by |
|---|---|
| Every member of every library, every global and every file method of Lua 5.3 and 5.4 is declared for its target, and nothing else is (`bit32` for 5.3, `warn` and the rest for 5.4) | `TestStdlibMatchesLua`, against the interpreters themselves |
| Every function is accepted in the forms the manual gives, and gives Lua's results | `tests/luacompat/stdlib.lua`, `stdlib54.lua`, `stdlib53.lua` |
| Results are typed for typed code (`math.floor` is a number, `string.rep` a string, ...) | `tests/regress/run/stdlib_typed.lunar` |
| Functions that return nil only on failure (`io.open`, `os.getenv`, `math.type`, ...) are declared by their success type, as in Teal | `tests/luacompat/fail_results.lua`, `tests/regress/refuse/library_results.lunar` |

### Does not work, or not yet

| What | Notes |
|---|---|
| Results whose type depends on the arguments (`string.find`, `string.byte`, `select`, `os.date`) are `any` | correct values, no checking |
| Lua builds without the 5.3 compatibility functions (`math.pow` and friends) | declared, as the standard builds have them; such a build fails at run time |

## Projects and the command line

### Works

| What | Proved by |
|---|---|
| `lunar build`: the whole folder, all or nothing, the last good Lua kept on failure, stale outputs removed | `internal/project` build and write tests |
| Concurrent builds, symlinked folders, absolute config paths, an `outDir` that would hold `rootDir` refused | `internal/project/paths_test.go` |
| `--json` (one object per line), exit codes 0/1/2/3, usage mistakes | `TestUsageMistakes`, `TestEngine*`, `internal/diag` |
| Data files: no globals or library calls, imports of types only | `internal/project/datafiles_test.go`, `tests/host` (loaded in a Lua state with no libraries) |
| Declarations from anywhere; large engine APIs parsed once per build | `internal/project` tests |
| Source maps for every statement, named after `-o` | `TestSourceMapCoversNestedStatements`, `TestSourceMapFollowsTheOutputName` |

### Does not work, or not yet

| What | Notes |
|---|---|
| JSON columns count bytes | documented; editors' own diagnostics come from the language server |

## Editors

### Works

| What | Proved by |
|---|---|
| Language server: diagnostics, definition, references, rename (refusing conflicts), hover, completion while typing, signature help, document and workspace symbols, file watching, untitled buffers, a workspace without a config | `internal/lsp` (tests run with the race detector) |
| A request the server fails on is answered with an error, and the server goes on | `TestPanickingRequestIsAnError` |
| Neovim 0.12: filetype, tree-sitter highlighting, the server attaching, definition, diagnostics | `editors/nvim/test/run.sh` (CI) |
| Tree-sitter grammar: every `.lunar` file and the Lua corpus parse cleanly | `editors/tree-sitter-lunar/check.sh` (CI) |
| Tree-sitter reads `await` as the compiler does: Lunar's keyword before a name on the same line, Lua's name in `await(x)`, `await (x)` and `local a = await` | corpus test "await as a name" (`src/scanner.c`) |
| VS Code extension and Zed extension build and package | CI |
| VS Code TextMate grammar | checked by hand with vscode-textmate; no test in CI |

### Does not work, or not yet

| What | Notes |
|---|---|
| Zed itself not tried | the extension builds; it has not been run inside Zed |
| VS Code extension not published to the marketplace | install the `.vsix` CI builds |
| The tree-sitter grammar reads `as` at the start of a line as an assertion on the line before | highlighting only; the compiler reads it right |
| The VS Code problem matcher keeps JSON escapes in messages | the language server's diagnostics have none |
