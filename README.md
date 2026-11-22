# Lunar

**A statically-typed superset of Lua that compiles to plain Lua 5.3 / 5.4.**

```lunar
class Cart
    private items: number[]

    constructor()
        self.items = {}
    end

    add(price: number): void
        table.insert(self.items, price)
    end

    total(): number
        local sum: number = 0
        for _, price in ipairs(self.items) do
            sum = sum + price
        end
        return sum
    end
end

local cart: Cart = Cart()
cart.add(10)
cart.add(20)
assert(cart.total() == 30)
```

## What works today

Every line below is backed by a program in [`examples/features/`](examples/features/)
that CI compiles and **runs** under Lua 5.3 and 5.4, asserting its own results
(`tests/examples`). If a feature is not on this list, treat it as unsupported.

- Lua itself: locals, `local function`, `if`/`while`/numeric and generic `for`,
  `repeat … until`, varargs, closures, `pcall`/`error`, method calls with `:`
- Plain Lua unchanged: the files in [`tests/luacompat/`](tests/luacompat/)
  compile with no diagnostics and print what they print under Lua: Lua's
  operator precedence, `f{...}`/`f"..."` calls, `goto` and labels,
  `<const>`/`<close>`, `;` in tables, and Lunar's own words (`from`, `match`,
  `class`, ...) as names
- Type annotations on variables, parameters and returns; optional parameters
  (`b: number?` or `b?: number`); multiple returns (`(number, string)`)
- Classes: fields, `private`/`protected`, methods, `static` methods and fields,
  `extends` with `super(...)`, `abstract`, `readonly`, constructor parameter
  properties (`constructor(public x: number)`)
- Interfaces (`implements`, structural checks for tables), enums, type aliases,
  union and literal types, type guards (`value is string`), `as` assertions
- Generics on functions, classes and interfaces (erased at runtime)
- Utility types: `Pick`, `Omit`, `Partial`, `Readonly`, `Record`
- Optional chaining `?.`, `?[`, `?.()` and nullish coalescing `??`
- `match … with` pattern matching with guards and table patterns
- Array and table spread (`{...a, 4}`, `{...defaults, ...overrides}`)
- Template strings (`` `Hello, ${name}!` ``)
- Bitwise operators `& | ~ << >>`; `^` is exponentiation, as in Lua
- `import { a } from "./mod"` / `export function` (exports stay out of the
  global environment), function decorators, `namespace`
- Getters and setters (`get amount(): number`), inherited by subclasses
- Lua numerals in every form (`0xFF`, `1e-9`, `.5`); conditions test
  truthiness as in Lua, so `while true do` and `if enemy then` work
- Data files that end in `return { ... }`, loaded with no libraries opened
  (`examples/data/`, checked by `tests/host`)
- Optional fields (`speed: number?` or `speed?: number`) may be left out of a
  table literal, and a literal that does not fit names the field at fault:

  ```
  Field 'hp' of 'Enemy' must be 'number', got 'string'
  'hpp' is not a field of 'Enemy'. Did you mean 'hp'?
  Missing field(s) 'hp' required by 'Enemy'
  ```
- `Enemy.new(3)` and `Enemy(3)` both construct
- Declarations: `declare function` (the trailing `end` is optional),
  `declare class`, `declare namespace`, interfaces that refer to themselves
  or to each other

## Not supported

Function overloading is for `declare function` only, and `async`/`await` are
refused with an error: Lua's coroutines do that job. Targets other than Lua
5.3 and 5.4 (LuaJIT, 5.1) are not supported.

## Install

Requires Go 1.23 or newer.

```sh
make build                       # builds dist/lunar (and dist/lunar-lsp, an alias for `lunar lsp`)
make install PREFIX=$HOME/.local # copies them to $HOME/.local/bin
```

The binary is self-contained: the standard library declarations are built in.

## Use

A project is a folder of scripts with a `lunar.config.json` (paths are
relative to it):

```json
{
  "rootDir": "scripts",
  "outDir": "../build/scripts",
  "target": "lua54",
  "declarations": ["../engine/api"],
  "dataFiles": ["data/**"]
}
```

```sh
lunar build                 # compile every script under rootDir into outDir
lunar build --json --watch  # for an editor: JSON lines, rebuild on change
```

`lunar build` checks the whole folder first and writes only if nothing is
wrong, so the last good Lua stays in place when a script has an error. The
output mirrors `rootDir`; plain `.lua` scripts are copied through, and outputs
whose source was deleted are removed (nothing lunar did not write is touched).

- **Imports** starting with `./` or `../` are relative to the importing file,
  others to `rootDir`. `require` names are relative to `rootDir`:
  `scripts/ai/brain.lunar` is `require("ai.brain")`, so the engine sets
  `package.path` to the output folder.
- **Declarations** are the bundled Lua standard library (built into lunar),
  every `*.d.lunar` under `rootDir`, and the `declarations` paths and `--decl`
  flags, which may be anywhere, such as the engine's API.
- **Data files** (`dataFiles` globs) are loaded by the engine with no libraries
  opened, so using a global (`tostring`, the engine API) or a construct whose
  Lua needs one (classes, spreads, template strings, `match`) is an error in
  them.

Single files compile the same way, checking what they import:

```sh
lunar game.lunar                    # writes game.lua
lunar -o out/game.lua game.lunar    # choose the output file
lunar --target lua54 game.lunar     # lua53 (default) or lua54
lunar --source-map game.lunar       # also writes game.lua.map
lunar --watch game.lunar            # recompile on change
lunar --json game.lunar             # errors as JSON lines, for tools
lunar --version
```

### Errors for tools

With `--json`, stdout carries one JSON object per line and stderr stays
empty, so a program that runs the compiler can list errors and jump to them:

```
{"kind":"diagnostic","file":"/game/scripts/enemy.lunar","line":5,"column":37,"endLine":5,"endColumn":42,"severity":"error","code":"field-type","message":"Field 'hp' of 'Enemy' must be 'number', got 'string'"}
{"kind":"summary","errors":1,"warnings":0,"files":1,"written":false}
```

Lines and columns are 1-based, columns count bytes, and the end is just past
the span. `file` is absolute; errors in the bundled declarations read
`<stdlib>/lua.d.lunar`. `code` names the kind of error and does not change
between releases: `syntax`, `type`, `field-type`, `unknown-field`,
`missing-fields`, `unknown-member`, `duplicate-function`, `not-supported`,
`usage`, `io`, `internal`. In watch mode every rebuild ends with a summary.

Exit codes: `0` compiled, `1` the source has errors and nothing was written
(the previous output is left as it was), `2` usage or configuration error,
`3` I/O failure or internal error.

The type checker decides whether a member call needs Lua's `:` (an instance
method) or `.` (a function in a table). When it cannot know, because a value
is typed `any`, the call is emitted as written: write `obj:method()` to pass
`self`.

Declaration files (`*.d.lunar`) describe existing Lua or C APIs to the type
checker:

```lunar
declare function spawn(name: string, x?: number): number

declare namespace audio
    function play(name: string, volume?: number): void
end
```

## Editor support

`lunar lsp` is the language server. It reads the same configuration and
declarations as `lunar build`, so the editor and the build agree:
diagnostics for every file as you type, go to definition (into the standard
library too), references and rename across files, hover with types,
completion of members and names in scope, signature help, and document and
workspace symbols.

| Editor | | |
|---|---|---|
| Neovim 0.11+ | [`editors/nvim`](editors/nvim) | `vim.lsp`, tree-sitter highlighting, folding and indentation |
| Zed | [`editors/zed`](editors/zed) | the language server and tree-sitter highlighting |
| VS Code | [`editors/vscode`](editors/vscode) | the language server and TextMate highlighting; runs in WSL |

The tree-sitter grammar is [`editors/tree-sitter-lunar`](editors/tree-sitter-lunar).

## Documentation

- [`docs/LANGUAGE.md`](docs/LANGUAGE.md): the language, with examples the tests compile
- [`docs/ENGINE_INTEGRATION.md`](docs/ENGINE_INTEGRATION.md): running Lunar from a
  game engine or build tool: configuration, `lunar build`, JSON output, exit
  codes, `require` names, data files, declaring the engine's API
- [`CHANGELOG.md`](CHANGELOG.md)

## Development

```sh
go test ./...                                   # unit, golden, runtime and integration tests
LUNAR_REQUIRE_LUA=1 go test ./...               # fail (not skip) when Lua is missing
go test ./tests/examples -run TestGoldenOutput -update   # accept new codegen output
editors/tree-sitter-lunar/check.sh              # the grammar: needs the tree-sitter CLI
editors/nvim/test/run.sh                        # headless Neovim smoke test
```

The runtime tests need `lua5.3` and `lua5.4`; the no-libraries host test needs
their development files (`liblua5.3-dev`, `liblua5.4-dev`) and a C compiler.

## License

MIT
