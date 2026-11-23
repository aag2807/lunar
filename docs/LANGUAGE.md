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

Code without annotations is checked as Lua: a local declared empty (`local
x`) takes whatever it is given later; a table made from a literal gains the
fields assigned to it, and a field it lacks reads as `any`; a parameter
without a type may be left out; `a or b` and `a and b` have the type of the
operand they return (`(c.attempt or 0) + 1` is a number), and `a and a.x`
knows `a` is not nil on the right; values are adjusted to names as in Lua
(`local ok = pcall(f)`, `local a, b = 1`); `function M.add(a, b)` and
`function Point:len()` define functions into tables. Once you annotate
something, it is checked strictly: an interface you write down has exactly
its fields, and a typed parameter must be passed.

Only Lua's own reserved words are reserved. The words Lunar adds (`class`,
`match`, `from`, `import`, `get`, `set`, `number`, ...) are names wherever a
declaration of Lunar's cannot begin: `local from = 1`, `t.class`,
`s:match("%d")`, `function f(public)`. `async(x)`, `await(x)` and `local
async` are Lua's too. `await` followed by a name on the same line (`await
f()`) and `local async function f()` on one line are Lunar's: in plain Lua
they would be two statements side by side.

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

A string's methods may be called with `.` as with `:` (`s.upper()` is
`s:upper()`), the way Lunar calls methods. A call that is already complete
as plain Lua stays as written: `s.sub(s, 2, 3)`, which passes the string
itself, and `s.upper("xyz")`, whose arguments fill every parameter, are
`string.sub(s, 2, 3)` and `string.upper("xyz")`.

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

Parameters are optional when typed `T?` (or written `name?: T`), or when
they have a default (`times: number = 1`), the value they take when the
argument is left out or nil; a rest parameter collects the remaining
arguments. Generic functions, and methods, take type parameters, which are
erased; `<T extends Named>` constrains one, so a `T` has what `Named` has.

```lunar
function greet(name: string, punctuation: string?): string
    return "Hello, " .. name .. (punctuation or "!")
end

function repeated(s: string, times: number = 2): string
    return string.rep(s, times)
end

function firstOf<T>(items: T[]): T?
    return items[1]
end

interface Named
    name: string
end

function nameOf<T extends Named>(x: T): string
    return x.name
end

local double = (x: number) => x * 2
print(greet("ada"), repeated("ab"), firstOf<number>({ 7, 8 }), nameOf({ name = "n" }), double(4))
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
binds the value, `name: Type`, table patterns, a dotted name such as an enum
member (`State.Idle`), alternatives (`| 1 | 2 ->`), and `when` guards. Its
value has the type of what its arms give, and a binding has the matched
value's type. An arm ends at the next `|`; parenthesize a bitwise or inside
one.

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

A spread copies a list's values, or a record's fields, into a table literal.
Destructuring takes a call's several results by name, `local (a, b) = f()`
(the same as Lua's `local a, b = f()`, which also works); there is no
destructuring of a table's fields.

```lunar
local base = { 1, 2, 3 }
local more = { 0, ...base, 4 }
local defaults = { retries = 3, timeout = 30 }
local settings = { ...defaults, timeout = 60 }

local function minmax(xs: number[]): (number, number)
    return math.min(table.unpack(xs)), math.max(table.unpack(xs))
end
local (lo, hi) = minmax(base)
print(#more, settings.timeout, lo, hi)
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

## Async functions

An async function may pause the coroutine it runs in and carry on later:
wait two seconds, or wait for a texture to load. Lunar checks that it is
only ever called with `await`, inside another async function, so nothing
pauses outside a coroutine, where Lua would stop with "attempt to yield from
outside a coroutine". `async` and `await` are erased, as types are: `await
wait(2)` is `wait(2)` in the Lua. Lua's coroutines pause from any depth of
calls, so the Lua needs nothing else.

Something has to resume a paused coroutine: in a game, the engine, once a
frame. The functions that really pause are the engine's, declared `async`;
they yield the running coroutine, and the engine resumes it with their
results (see [Scripts that pause](ENGINE_INTEGRATION.md#scripts-that-pause)).

```lunar
interface Texture
    width: number
end

declare async function wait(seconds: number): void
declare async function loadTexture(path: string): Texture
declare function spawn(f: async function(): void): void

class Door
    open: boolean = false

    async slide(): void
        await wait(0.5)
        self.open = true
    end
end

async function cutscene(door: Door): void
    local banner = await loadTexture("banner.png")
    print(banner.width)
    await door.slide()
    await wait(2)
end

spawn(async function()
    await cutscene(Door())
end)
```

- `async` goes before any function: `async function f`, `local async
  function f`, `async function M.f`, a method (`async slide()`), `declare
  async function`, `async function() ... end` and `async (x) => ...`.
  Function types say it too: `async function(): void`, `async () => void`.
- An async function is called with `await`, and `await` is written only in an
  async function. A plain function nested in an async one is not async.
- `await f()` has the type of `f`'s results, all of them: `local a, b = await
  f()`.
- An async function cannot go where a plain function is expected, such as a
  `table.sort` comparator or a parameter typed `function(): void`: whatever
  calls it there would not await it. A plain function may go where an async
  one is expected, as to `spawn` above.
- An async method cannot override or implement a plain one. Constructors,
  getters, setters, metamethods and the iterator of a generic `for` cannot be
  async: nothing awaits them.
- A value typed `any`, or a bare `function`, may hold either, unchecked. So
  may a table made with `local M = {}`, whose fields are not typed: write
  `export async function f()` in a module, or give `M` a type, for
  `async function M.f()` to be checked.

[`examples/features/25_async.lunar`](../examples/features/25_async.lunar)
runs a small scheduler of its own.

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
- **Targets other than Lua 5.3 and 5.4.**
- **Code nested more than 200 levels deep** (blocks, or expressions inside
  expressions): Lua refuses to load it ("C stack overflow"), so Lunar
  refuses it too, saying so.
