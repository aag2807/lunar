# Lunar Compiled Output Guide

This document explains how Lunar features compile to Lua code with practical examples.

## Table of Contents
- [Classes](#classes)
- [Static Methods](#static-methods)
- [Interfaces](#interfaces)
- [Generics](#generics)
- [Type Annotations](#type-annotations)
- [Access Modifiers](#access-modifiers)
- [Inheritance](#inheritance)

---

## Classes

### Basic Class

**Lunar Code:**
```lunar
class Person
    name: string
    age: number

    constructor(name: string, age: number)
        self.name = name
        self.age = age
    end

    greet(): string
        return "Hello, I'm " .. self.name
    end
end

local p: Person = Person("Alice", 30)
print(p.greet())
```

**Compiled Lua:**
```lua
local Person = {}
Person.__index = Person

function Person.new(name, age)
    local self = setmetatable({}, Person)
    self.name = name
    self.age = age
    return self
end

function Person:greet()
    return "Hello, I'm " .. self.name
end

local p = Person.new("Alice", 30)
print(p:greet())
```

**Key Points:**
- Classes become Lua tables with `__index` metamethod
- Constructor becomes `ClassName.new()` function
- Methods use colon syntax (`:`) for implicit `self`
- Type annotations are removed (compile-time only)

---

## Static Methods

### Static vs Instance Methods

**Lunar Code:**
```lunar
interface MathLib
    static add(a: number, b: number): number
    static multiply(a: number, b: number): number
end

interface Calculator
    value: number
    add(n: number): void
    getValue(): number
end
```

**In Declaration Files (.d.lunar):**
```lunar
declare interface GraphicsModule
    static clear(r: number, g: number, b: number): void
    static print(text: string, x: number, y: number): void
    width: number
    height: number
end

declare interface Love
    graphics: GraphicsModule
    load(): void
    update(dt: number): void
    draw(): void
end

declare const love: Love
```

**Usage in Code:**
```lunar
-- Static methods use dot notation
love.graphics.clear(0, 0, 0, 1)
love.graphics.print("Hello", 100, 100)

-- Instance methods use colon notation
love:load()
love:update(0.016)
love:draw()

-- Properties use dot notation
local w: number = love.graphics.width
```

**Autocomplete Behavior:**
- Typing `love.graphics.` shows: `clear`, `print`, `width`, `height`
- Typing `love.graphics:` shows: nothing (all members are static)
- Typing `love:` shows: `load`, `update`, `draw` (instance methods only)
- Typing `love.` shows: `graphics` (properties only)

**Key Points:**
- Static methods are called with `.` (dot)
- Instance methods are called with `:` (colon)
- The LSP autocomplete filters based on the operator used
- Static keyword only affects autocomplete behavior, not compiled output

---

## Interfaces

### Interface Definition and Implementation

**Lunar Code:**
```lunar
interface Drawable
    x: number
    y: number
    draw(): void
end

class Sprite implements Drawable
    x: number
    y: number
    texture: string

    constructor(x: number, y: number, texture: string)
        self.x = x
        self.y = y
        self.texture = texture
    end

    draw(): void
        print("Drawing sprite at " .. self.x .. "," .. self.y)
    end
end
```

**Compiled Lua:**
```lua
-- Interface is not compiled (compile-time only)

local Sprite = {}
Sprite.__index = Sprite

function Sprite.new(x, y, texture)
    local self = setmetatable({}, Sprite)
    self.x = x
    self.y = y
    self.texture = texture
    return self
end

function Sprite:draw()
    print("Drawing sprite at " .. self.x .. "," .. self.y)
end
```

**Key Points:**
- Interfaces exist only at compile-time
- They don't generate any Lua code
- Used for type checking and IDE autocomplete
- Classes implementing interfaces must provide all required members

---

## Generics

### Generic Class

**Lunar Code:**
```lunar
class Box<T>
    private value: T

    constructor(value: T)
        self.value = value
    end

    getValue(): T
        return self.value
    end

    setValue(value: T): void
        self.value = value
    end
end

local numberBox: Box<number> = Box<number>(42)
local stringBox: Box<string> = Box<string>("hello")

print(numberBox.getValue())  -- 42
print(stringBox.getValue())  -- "hello"
```

**Compiled Lua:**
```lua
local Box = {}
Box.__index = Box

function Box.new(value)
    local self = setmetatable({}, Box)
    self.value = value
    return self
end

function Box:getValue()
    return self.value
end

function Box:setValue(value)
    self.value = value
end

local numberBox = Box.new(42)
local stringBox = Box.new("hello")

print(numberBox:getValue())  -- 42
print(stringBox:getValue())  -- "hello"
```

**Key Points:**
- Generic type parameters (`<T>`) are erased during compilation
- Type safety is enforced at compile-time only
- No runtime overhead for generics
- All generic instantiations use the same Lua code

---

## Type Annotations

### Function Type Annotations

**Lunar Code:**
```lunar
function add(a: number, b: number): number
    return a + b
end

function processUser(
    name: string,
    age: number,
    callback: function(user: string): void
): void
    local user: string = name .. " (" .. age .. ")"
    callback(user)
end

local result: number = add(10, 20)
```

**Compiled Lua:**
```lua
function add(a, b)
    return a + b
end

function processUser(name, age, callback)
    local user = name .. " (" .. age .. ")"
    callback(user)
end

local result = add(10, 20)
```

**Key Points:**
- All type annotations are completely removed
- Function signatures lose their type information
- Local variable types are stripped
- No runtime type checking (unless you add it manually)

---

## Access Modifiers

### Public, Private, Protected

**Lunar Code:**
```lunar
class BankAccount
    public owner: string
    private balance: number
    protected accountNumber: string

    constructor(owner: string, initial: number)
        self.owner = owner
        self.balance = initial
        self.accountNumber = "ACC" .. tostring(math.random(1000, 9999))
    end

    public deposit(amount: number): void
        self.balance = self.balance + amount
    end

    public getBalance(): number
        return self.balance
    end

    private validateAmount(amount: number): boolean
        return amount > 0
    end
end

local account: BankAccount = BankAccount("Alice", 1000)
account.deposit(500)
print(account.getBalance())  -- 1500
-- print(account.balance)  -- Compile error: cannot access private property
```

**Compiled Lua:**
```lua
local BankAccount = {}
BankAccount.__index = BankAccount

function BankAccount.new(owner, initial)
    local self = setmetatable({}, BankAccount)
    self.owner = owner
    self.balance = initial
    self.accountNumber = "ACC" .. tostring(math.random(1000, 9999))
    return self
end

function BankAccount:deposit(amount)
    self.balance = self.balance + amount
end

function BankAccount:getBalance()
    return self.balance
end

function BankAccount:validateAmount(amount)
    return amount > 0
end

local account = BankAccount.new("Alice", 1000)
account:deposit(500)
print(account:getBalance())  -- 1500
-- print(account.balance) works in Lua! Access control is compile-time only
```

**Key Points:**
- Access modifiers (`public`, `private`, `protected`) are compile-time only
- In compiled Lua, all members are accessible
- The Lunar compiler prevents invalid access during type checking
- Use naming conventions (e.g., `_private`) if you need runtime privacy hints

---

## Inheritance

### Class Inheritance and Super Calls

**Lunar Code:**
```lunar
class Animal
    protected name: string
    protected age: number

    constructor(name: string, age: number)
        self.name = name
        self.age = age
    end

    speak(): void
        print(self.name .. " makes a sound")
    end

    getInfo(): string
        return self.name .. " is " .. tostring(self.age) .. " years old"
    end
end

class Dog extends Animal
    private breed: string

    constructor(name: string, age: number, breed: string)
        super(name, age)
        self.breed = breed
    end

    speak(): void
        print(self.name .. " barks: Woof!")
    end

    getBreed(): string
        return self.breed
    end
end

local dog: Dog = Dog("Buddy", 3, "Golden Retriever")
dog.speak()        -- "Buddy barks: Woof!"
print(dog.getInfo())  -- "Buddy is 3 years old"
print(dog.getBreed()) -- "Golden Retriever"
```

**Compiled Lua:**
```lua
local Animal = {}
Animal.__index = Animal

function Animal.new(name, age)
    local self = setmetatable({}, Animal)
    self.name = name
    self.age = age
    return self
end

function Animal:speak()
    print(self.name .. " makes a sound")
end

function Animal:getInfo()
    return self.name .. " is " .. tostring(self.age) .. " years old"
end

local Dog = {}
Dog.__index = Dog
setmetatable(Dog, { __index = Animal })

function Dog.new(name, age, breed)
    local self = setmetatable({}, Dog)
    -- Super call
    Animal.new(self, name, age)
    self.breed = breed
    return self
end

function Dog:speak()
    print(self.name .. " barks: Woof!")
end

function Dog:getBreed()
    return self.breed
end

local dog = Dog.new("Buddy", 3, "Golden Retriever")
dog:speak()        -- "Buddy barks: Woof!"
print(dog:getInfo())  -- "Buddy is 3 years old"
print(dog:getBreed()) -- "Golden Retriever"
```

**Key Points:**
- Inheritance uses metatable chaining
- `Dog.__index` is set to `Dog`, and `Dog`'s metatable `__index` is set to `Animal`
- `super()` calls become `ParentClass.new(self, ...)`
- Child class can override parent methods
- Parent methods are accessible through the metatable chain

---

## Additional Examples

### Readonly Properties

**Lunar Code:**
```lunar
class Config
    readonly appName: string
    readonly version: number

    constructor(appName: string, version: number)
        self.appName = appName
        self.version = version
    end

    getInfo(): string
        return self.appName .. " v" .. tostring(self.version)
    end
end

local config: Config = Config("MyApp", 1.0)
print(config.appName)  -- "MyApp"
-- config.appName = "NewName"  -- Compile error: cannot assign to readonly property
```

**Compiled Lua:**
```lua
local Config = {}
Config.__index = Config

function Config.new(appName, version)
    local self = setmetatable({}, Config)
    self.appName = appName
    self.version = version
    return self
end

function Config:getInfo()
    return self.appName .. " v" .. tostring(self.version)
end

local config = Config.new("MyApp", 1.0)
print(config.appName)  -- "MyApp"
config.appName = "NewName"  -- Works in Lua! readonly is compile-time only
```

**Key Points:**
- `readonly` properties can only be assigned in the constructor
- Compile-time enforcement only
- In Lua, readonly properties can still be modified (no runtime protection)

---

## Summary

| Lunar Feature | Compile-Time | Run-Time | Lua Output |
|---------------|--------------|----------|------------|
| **Classes** | Structure enforced | Metatables | Table with `__index` |
| **Static methods** | Autocomplete filtering | No difference | Regular functions |
| **Interfaces** | Type checking | Not present | No code generated |
| **Generics** | Type safety | Type erasure | Same code for all types |
| **Type annotations** | Type checking | Removed | No types in output |
| **Access modifiers** | Enforced | Not enforced | All members accessible |
| **Inheritance** | Type hierarchy | Metatable chain | Chained `__index` |
| **Readonly** | Assignment check | Not enforced | Regular property |

**General Principles:**
1. **Type information is compile-time only** - All types are erased before generating Lua
2. **Clean Lua output** - Generated code is readable and idiomatic Lua
3. **No runtime overhead** - Classes compile to efficient metatable-based code
4. **100% Lua compatible** - Output can be used with any Lua runtime
5. **Safety through static analysis** - Errors caught before runtime
