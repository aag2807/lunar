# Lunar Language Specification

## Overview
Lunar is a statically-typed superset of Lua 5.1 that compiles to clean, efficient Lua code. It adds type safety and modern programming features while maintaining full compatibility with existing Lua code.

## Table of Contents
1. [Basic Types](#basic-types)
2. [Variables and Constants](#variables-and-constants)
3. [Functions](#functions)
4. [Interfaces](#interfaces)
5. [Classes](#classes)
6. [Generics](#generics)
7. [Enums](#enums)
8. [Type System](#type-system)
9. [Modules](#modules)
10. [Type Declarations](#type-declarations)

## Basic Types

### Primitive Types
- `string`: Text values
- `number`: Both integer and floating-point numbers
- `boolean`: `true` or `false`
- `nil`: Represents absence of a value
- `any`: Any type (escape hatch from type checking)
- `void`: Represents no return value in functions

### Complex Types
- Arrays: `T[]` where T is any valid type
- Tables: `table<K, V>` where K and V are valid types
- Tuples: `(T1, T2, ...)` for multiple return values
- Union Types: `T1 | T2`
- Optional Types: `T?` (shorthand for `T | nil`)

## Variables and Constants

### Variable Declaration
```lua
-- Type inference
local name = "lunar"  -- inferred as string

-- Explicit type annotation
local age: number = 25
local isValid: boolean = true

-- Optional types
local data: string? = nil

-- Constants (immutable variables)
const MAX_SIZE: number = 100
const DEBUG: boolean = false
```

## Functions

### Function Declaration
```lua
-- Basic function with type annotations
function greet(name: string): string
    return "Hello, " .. name
end

-- Optional parameters
function createUser(name: string, age: number?): User
    -- Implementation
end

-- Multiple return values using tuple type
function getCoordinates(): (number, number)
    return 10, 20
end

-- Generic function
function map<T, U>(array: T[], fn: (item: T) => U): U[]
    local result: U[] = {}
    for _, item in ipairs(array) do
        table.insert(result, fn(item))
    end
    return result
end
```

## Interfaces

### Interface Declaration
```lua
interface Vehicle
    brand: string
    year: number
    start(): void
    stop(): void
end

interface ElectricVehicle extends Vehicle
    batteryLevel: number
    charge(duration: number): void
end
```

## Classes

### Class Declaration
```lua
class Car implements Vehicle
    private brand: string
    private year: number
    private running: boolean

    constructor(brand: string, year: number)
        self.brand = brand
        self.year = year
        self.running = false
    end

    public start(): void
        self.running = true
    end

    public stop(): void
        self.running = false
    end
end
```

### Access Modifiers
- `public`: Accessible from anywhere (default)
- `private`: Accessible only within the class
- `protected`: Accessible within the class and its descendants

## Generics

### Generic Types
```lua
class Stack<T>
    private items: T[]

    constructor()
        self.items = {}
    end

    public push(item: T): void
        table.insert(self.items, item)
    end

    public pop(): T?
        return table.remove(self.items)
    end
end

-- Usage
local numberStack: Stack<number> = Stack<number>.new()
```

## Enums

### Enum Declaration
```lua
enum Direction
    North
    South
    East
    West
end

enum HttpStatus
    OK = 200
    NotFound = 404
    ServerError = 500
end
```

## Type System

### Type Aliases
```lua
type UserId = number
type Email = string
type UserCallback = (user: User) => void

type Point
    x: number
    y: number
end
```

### Union Types
```lua
type Status = "loading" | "success" | "error"
type NumberOrString = number | string
```

### Type Guards
```lua
function isString(value: any): value is string
    return type(value) == "string"
end
```

## Modules

### Module System
```lua
-- Exporting (in user.lunar)
export interface User
    id: number
    name: string
end

export class UserService
    -- Implementation
end

-- Importing (in main.lunar)
import { User, UserService } from "./user"
```

## Type Declarations

### Declaration Files
```lua
-- types.lunar
declare interface Window
    width: number
    height: number
end

declare function setTimeout(callback: () => void, ms: number): number
```

## Conventions

### Naming Conventions
- Interface names: PascalCase
- Class names: PascalCase
- Function names: camelCase
- Variable names: camelCase
- Constants: UPPER_SNAKE_CASE
- File names: lowercase with hyphens (e.g., user-service.lunar)

### File Extension
- `.lunar` for Lunar source files
- `.d.lunar` for Lunar type declaration files

### Comments
```lua
-- Single line comment

--[[
    Multi-line
    comment
]]
```

### Type Annotations Style
- Space after colon in type annotations: `name: string`
- No space before colon: `name: string` (not `name : string`)
- Space after comma in generic types: `Map<string, number>`
