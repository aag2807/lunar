# The Lunar language

Lunar is Lua with types. Every Lua 5.3/5.4 program is a Lunar program, and the
compiler turns Lunar into plain Lua, erasing the types. Every example on this
page is compiled by the test suite (`tests/examples/docs_test.go`), and the
programs in [`examples/features/`](../examples/features/) are run and checked
under Lua 5.3 and 5.4.

## Lua

All of Lua's syntax works: `local`, `local function`, `if`/`elseif`/`else`,
`while`, numeric and generic `for`, `repeat … until`, `do … end`, `break`,
`goto` and `::labels::`, varargs, closures, multiple assignment and returns,
`;`, every numeral (`0xFF`, `1e-9`, `.5`), long strings and comments, method
calls with `:`, calls with a table or a string and no parentheses
(`spawn { kind = "imp" }`, `require "ai"`), Lua 5.4's `<const>` and `<close>`
(with target `lua54`), and Lua's operator precedence. A plain Lua file is a
Lunar file: the files in [`tests/luacompat/`](../tests/luacompat/) are
compiled unchanged and must print what they print under Lua.

Only Lua's own reserved words are reserved. The words Lunar adds (`class`,
`match`, `from`, `import`, `get`, `set`, `number`, ...) are names wherever a
declaration of Lunar's cannot begin: `local from = 1`, `t.class`,
`s:match("%d")`, `function f(public)`.

```lunar
local function sum(...: number): number
    local total = 0
    for _, n in ipairs({...}) do
        total = total + n
    end
    return total
end

local a, b = 1, 2
a, b = b, a
local mask = 0xFF & 0x0F
local s = ("one two"):upper()
print(sum(1, 2, 3), a, b, mask, s)

local from = { x = 1 }
for _, p in ipairs { from, { x = 2 } } do
    if p.x == 2 then goto done end
end
::done::
```

Conditions test truthiness as in Lua: everything but `nil` and `false` is true,
so `while true do` and `if enemy then` work. A file may end in `return value`,
which is how data files are written.

## Types

Annotate variables, parameters and returns with `: Type`. Unannotated values
are inferred.

| Type | |
|---|---|
| `number`, `string`, `boolean`, `nil` | Lua's values |
| `any` | anything, unchecked |
| `unknown` | anything, but it has to be narrowed before use |
| `void` | no value (a function that returns nothing) |
| `never` | no value can have it |
| `T[]` | an array |
| `table<K, V>` | a table from `K` to `V` |
| `T?` | `T` or `nil` |
| `A \| B`, `A & B` | union, intersection |
| `"north"`, `42`, `true` | literal types |
| `(number, string)` | several return values |
| `function(a: number): string` or `(a: number) => string` | functions |

```lunar
type Direction = "north" | "south" | "east" | "west"

function step(d: Direction, n: number?): number
    if n == nil then
        return 1
    end
    return n
end

function divide(a: number, b: number): (number, string)
    if b == 0 then
        return 0, "division by zero"
    end
    return a / b, "ok"
end

local moves: number[] = { step("north"), step("east", 3) }
local names: table<string, number> = { up = 1 }
local q: number, err: string = divide(1, 2)
```

A comparison with `nil`, a truthiness test or a type guard narrows a value in
each branch, and past an early return:

```lunar
function isString(value: any): value is string
    return type(value) == "string"
end

function describe(value: any, limit: number?): string
    if not limit then
        return "unlimited"
    end
    if isString(value) then
        return value .. " up to " .. tostring(limit + 1)
    end
    return "other"
end
```

## Functions

Parameters are optional when typed `T?` (or written `name?: T`); a rest
parameter collects the remaining arguments. Generic functions take type
parameters, which are erased.

```lunar
function greet(name: string, punctuation: string?): string
    return "Hello, " .. name .. (punctuation or "!")
end

function firstOf<T>(items: T[]): T?
    return items[1]
end

local double = (x: number) => x * 2
print(greet("ada"), firstOf<number>({ 7, 8 }), double(4))
```

A global function may be called before its definition in the file. One name
has one implementation; overloads are written as `declare function`
signatures, for APIs implemented elsewhere.

## Interfaces and table literals

An interface describes a table's shape. A table literal written where one is
expected is checked field by field: a field of the wrong type, a field the
interface does not have, and a missing required field are each reported by
name. Fields typed `T?` or written `name?: T` may be left out.

```lunar
interface Stats
    atk: number
    def: number
end

interface Enemy
    name: string
    hp: number
    speed?: number
    stats: Stats
    attack(target: string): number
end

local imp: Enemy = {
    name = "imp",
    hp = 4,
    stats = { atk = 2, def = 1 },
    attack = function(self, target: string): number
        return #target
    end,
}

print(imp.attack("orc"))
```

A method declared in an interface (`attack(target: string): number`) is
called with `:`, so the function a table literal gives for it takes the table
first: `function(self, target)`. A field typed as a function
(`onHit: function(n: number): void`) is a plain field, called with `.`, and so
is any function in a table literal written without a type:
`local M = { greet = function(name) ... end }` then `M.greet("bob")`.

Interfaces can `extend` others, may name themselves or interfaces declared
later, and have index signatures: with `[key: string]: number`, every name
the interface does not declare reads and writes as a number, through `.` or
`[]`. `type Name … end` declares an object shape the same way.

An interface may take type parameters, given where it is used:

```lunar
interface Slot<T>
    item: T
    count: number
end

local arrows: Slot<string> = { item = "arrow", count = 20 }
```

## Classes

```lunar
abstract class Actor
    protected name: string
    static count: number = 0

    constructor(name: string)
        self.name = name
    end

    abstract speak(): string

    describe(): string
        return self.name .. " says " .. self.speak()
    end
end

class Goblin extends Actor
    private hp: number

    constructor(name: string, hp: number)
        super(name)
        self.hp = hp
    end

    speak(): string
        return "grr"
    end

    get health(): number
        return self.hp
    end

    set health(value: number)
        self.hp = value
    end
end

class Shaman extends Goblin
end

local g = Goblin("snik", 5)
local s = Shaman.new("mog", 7)
g.health = 3
print(g.describe(), g.health, s.describe())
```

- `Goblin(...)` and `Goblin.new(...)` both construct. A subclass without a
  constructor inherits its parent's. Declaring a member called `new` is an
  error.
- Members are `public` unless marked `private` or `protected`; `readonly`
  fields can only be set in the constructor; `static` members belong to the
  class.
- `constructor(public x: number)` declares and sets a field in one step.
- Getters and setters read and write like fields, and are inherited.
- A member the class (and its parents) does not have is an error.
- Methods are called with `.` in Lunar source (`g.describe()`); the compiler
  emits `:` where Lua needs it, including on a union of classes that all have
  the method.
- A method has to be called where it is named. `local f = g.describe` is an
  error: Lua has no bound methods, so `f()` would run without `self`. Wrap it:
  `local f = function() return g.describe() end`.
- Classes exported from a `namespace` or reached through `import * as m`
  construct the same way: `Game.Orc(5)`, `m.Enemy.new(3)`.

## Enums

Members without a value count up from the previous number, starting at 0.

```lunar
enum Rank
    Grunt
    Elite = 10
    Boss
end

local r: Rank = Rank.Boss
assert(Rank.Grunt == 0 and Rank.Boss == 11)
```

## Pattern matching

`match` tests a value against patterns in order: literals, `_`, a name that
binds the value, `name: Type`, table patterns, and `when` guards. An arm ends
at the next `|`; parenthesize a bitwise or inside one.

```lunar
function area(shape: any): number
    return match shape with
        | { kind: "circle", radius: r } -> 3.14159 * r * r
        | { kind: "rect", w: w, h: h } -> w * h
        | _ -> 0
    end
end

function grade(score: number): string
    return match score with
        | 100 -> "perfect"
        | s when s >= 90 -> "A"
        | _ -> "F"
    end
end
```

## Optional chaining and strings

`?.`, `?[` and `?.()` stop at `nil` instead of erroring; `??` replaces `nil`.
Template strings interpolate with `${}`.

```lunar
local config: any = { server = { port = 3000 } }
local port = config?.server?.port ?? 8080
local first = config?.tags?[1] ?? "none"
local label = `listening on ${port}`
```

## Spread and destructuring

```lunar
local base = { 1, 2, 3 }
local more = { 0, ...base, 4 }
local defaults = { retries = 3, timeout = 30 }
local settings = { ...defaults, timeout = 60 }
```

## Modules

A file exports with `export` and imports with `import`. Paths starting with
`./` or `../` are relative to the importing file; others are relative to the
project's `rootDir`.

```lunar fragment
-- mod/math.lunar
export function double(n: number): number
    return n * 2
end

-- main.lunar
import { double } from "./mod/math"
import * as math2 from "./mod/math"
print(double(2), math2.double(3))
```

An imported name that the module does not export, an import that finds no
module, and an import cycle are all errors. Exports stay out of the global
environment; a file that exports cannot also `return` a value.

## Declarations

Declaration files (`*.d.lunar`) describe Lua the compiler did not produce:
the standard library (built in), the engine's API, a Lua library. They hold
only signatures:

```lunar
declare function spawn(kind: string, count?: number): number
declare const VERSION: string

declare namespace audio
    function play(name: string, volume?: number): void
end

declare class Sound
    static new(path: string): Sound
    play(): void
end

declare interface Clock
    now: function(): number
    after: function(seconds: number): Clock
end
```

## Decorators and namespaces

A decorator is a function that receives a function or class and returns its
replacement. A namespace groups functions under a table.

```lunar
function logged(fn: any): any
    return function(...: any): any
        print("calling")
        return fn(...)
    end
end

@logged
function work(n: number): number
    return n + 1
end

namespace Geometry
    export function area(w: number, h: number): number
        return w * h
    end
end

print(work(1), Geometry.area(3, 4))
```

## Utility types

`Pick<T, K>`, `Omit<T, K>`, `Partial<T>`, `Required<T>`, `Readonly<T>`,
`Record<K, V>`, `keyof T`, `typeof value`, conditional types
(`T extends U ? X : Y`), mapped types (`{ [K in T]: U }`) and `as` assertions,
including `as const`.

```lunar
interface User
    id: number
    name: string
    email: string
end

type Preview = Pick<User, "id" | "name">
local p: Preview = { id = 1, name = "ada" }
local raw: any = "text"
local s: string = raw as string
```

## Not supported

- **Implementation overloads**: two bodies for one function name are an
  error; Lua keeps only the last.
- **`async`/`await`**: refused; use `coroutine.create` and `coroutine.resume`.
- **Targets other than Lua 5.3 and 5.4.**
