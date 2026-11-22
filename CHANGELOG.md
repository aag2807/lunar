# Changelog

## 2.0.0

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
  libraries opened lacks.

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

### Language

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
  emitted as written.
- Refused with an error instead of miscompiled: implementation overloads,
  `async`/`await`, `import * from` without a name, statements after `return`.
- Targets are `lua53` and `lua54` only.

### Tools

- `lunar lsp`, the language server, rebuilt on the same pipeline as `lunar
  build`: diagnostics for every file, definition (into the standard library
  too), references and rename across files, hover, completion, signature help,
  document and workspace symbols.
- A tree-sitter grammar (`editors/tree-sitter-lunar`), a Neovim plugin for
  `vim.lsp` and nvim-treesitter's main branch, a Zed extension, and an updated
  VS Code extension.

### Removed

WASM output, JIT hints, compiler plugins, the package manager and LuaRocks
commands, the bundler and `vendor/` libraries, the test runner, the REPL, the
formatter, linter, minifier and optimizers, the compile cache, `migrate`,
`docs`, project scaffolding, `lunar2decl`, `--no-typecheck`, and the
documentation for all of them.
