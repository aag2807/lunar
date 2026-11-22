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
- Type annotations on variables, parameters and returns; optional parameters
  (`b: number?` or `b?: number`); multiple returns (`(number, string)`)
- Classes: fields, `private`/`protected`, methods, `static` methods and fields,
  `extends` with `super(...)`, `abstract`, `readonly`, constructor parameter
  properties (`constructor(public x: number)`)
- Interfaces (`implements`, structural checks for tables), enums, type aliases,
  union and literal types, type guards (`value is string`), `as` assertions
- Generics on functions and classes (erased at runtime)
- Utility types: `Pick`, `Omit`, `Partial`, `Readonly`, `Record`
- Optional chaining `?.`, `?.[]`, `?.()` and nullish coalescing `??`
- `match … with` pattern matching with guards and table patterns
- Array and table spread (`{...a, 4}`, `{...defaults, ...overrides}`)
- Template strings (`` `Hello, ${name}!` ``)
- Bitwise operators `& | ~ << >>`; `^` is exponentiation, as in Lua
- `import { a } from "./mod"` / `export function`, function decorators,
  `namespace`

## Known gaps

Being fixed, in this order (see the change log as each lands):

- A file cannot end in a top-level `return { ... }`; `field: T?` in an
  interface is not treated as optional; table-literal errors do not name the
  offending field; `Enemy.new(3)` miscompiles to `Enemy:new(3)`
- `declare function` without a trailing `end` swallows the next declarations
- Declarations are only found next to the input file, and the standard
  library only next to the binary (or a few folders above the current one)
- Errors are human-readable only; there is no JSON output yet
- `while true do` is rejected; enum members after an explicit value are
  numbered wrongly; getters/setters do not work at runtime
- Function overloading and `async`/`await` compile but do not work; they will
  be refused with an error
- Imports resolve relative to the working directory, not the importing file
- Number literals with exponents or in hex (`1e-9`, `0xFF`) do not lex

## Install

Requires Go 1.23 or newer.

```sh
make build        # builds dist/lunar and dist/lunar-lsp
```

## Use

```sh
lunar game.lunar                    # writes game.lua
lunar -o out/game.lua game.lunar    # choose the output file
lunar --target lua54 game.lunar     # lua53 (default) or lua54
lunar --source-map game.lunar       # also writes game.lua.map
lunar --watch game.lunar            # recompile on change
lunar --version
```

Declaration files (`*.d.lunar`) describe existing Lua or C APIs to the type
checker. Those in the same folder as the input are loaded automatically, along
with the standard library declarations in [`stdlib/`](stdlib/).

```lunar
declare function spawn(name: string, x?: number): number end
```

## Editor support

`lunar-lsp` is a language server (diagnostics, hover, completion). Editor
plugins live in [`editors/`](editors/); they are being reworked, along with the
server, for Neovim, VS Code and Zed.

## Development

```sh
go test ./...                                   # unit, golden and integration tests
LUNAR_REQUIRE_LUA=1 go test ./tests/...         # fail (not skip) when Lua is missing
go test ./tests/examples -run TestGoldenOutput -update   # accept new codegen output
```

The runtime tests need `lua5.3` and `lua5.4`; the no-libraries host test needs
their development files (`liblua5.3-dev`, `liblua5.4-dev`) and a C compiler.

## License

MIT
