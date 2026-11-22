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

## Known gaps

Being fixed, in this order:

- Errors are human-readable only; there is no JSON output yet, and an error
  in a declaration file can be reported against the file being compiled
- Declarations are only found next to the input file, and the standard
  library only next to the binary (or a few folders above the current one)
- Imports resolve relative to the working directory, not the importing file,
  and a class cannot be imported from another module yet

Function overloading is for `declare function` only, and `async`/`await` are
refused with an error: Lua's coroutines do that job.

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

The type checker decides whether a member call needs Lua's `:` (an instance
method) or `.` (a function in a table). When it cannot know, because a value
is typed `any`, the call is emitted as written: write `obj:method()` to pass
`self`.

Declaration files (`*.d.lunar`) describe existing Lua or C APIs to the type
checker. Those in the same folder as the input are loaded automatically, along
with the standard library declarations in [`stdlib/`](stdlib/).

```lunar
declare function spawn(name: string, x?: number): number

declare namespace audio
    function play(name: string, volume?: number): void
end
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
