# Lunar Examples

This directory contains example code demonstrating Lunar language features.

## Declaration Files (`.d.lunar`)

Declaration files provide type information for external libraries written in Lua. They use the `declare` keyword to define ambient declarations without implementations.

### love.d.lunar

Type declarations for the LÖVE 2D game framework. This file demonstrates:

- `declare interface` for module types
- `declare const` for global objects
- `declare function` for global callbacks
- Nested interfaces for module structure

**Usage:**
Place `love.d.lunar` in your project directory. The Lunar compiler will use it to provide type checking for LÖVE APIs.

### love_game_example.lunar

A simple LÖVE game that uses the type declarations from `love.d.lunar`. Shows how typed external APIs enable:

- IntelliSense-like completion
- Type-safe API usage
- Compile-time error detection
- Better code documentation

## Declaration File Syntax

```lunar
-- Declare a constant
declare const PI: number

-- Declare a function
declare function print(message: string): void

-- Declare an interface
declare interface Graphics {
    clear: function(): void
    setColor: function(r: number, g: number, b: number): void
}
end

-- Declare a type alias
declare type Vector2 {
    x: number
    y: number
}
end

-- Declare with generics
declare type Optional<T> = T | nil
```

## Creating Your Own Declaration Files

1. Create a `.d.lunar` file named after your library (e.g., `mylibrary.d.lunar`)
2. Use `declare` statements to define the library's API
3. Place the file in your project directory
4. The Lunar compiler will automatically recognize and use it

Declaration files are perfect for:
- External Lua libraries
- C libraries with Lua bindings
- Game engine APIs (LÖVE, Corona, etc.)
- Any runtime-provided APIs

## Benefits

- **Type Safety**: Catch errors at compile time instead of runtime
- **Documentation**: Self-documenting API types
- **IDE Support**: Enable better code completion and navigation
- **Gradual Typing**: Add types to existing Lua projects incrementally
