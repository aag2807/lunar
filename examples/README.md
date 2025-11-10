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
Simply place `love.d.lunar` in your project directory alongside your `.lunar` files. The Lunar compiler **automatically discovers and loads** all `.d.lunar` files in the same directory - no imports or special configuration needed!

```bash
# Just compile your game - declarations load automatically!
lunar my_game.lunar
```

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

-- Declare a function (note: needs 'end')
declare function print(message: string): void end

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

## Auto-Loading Declaration Files

The Lunar compiler **automatically discovers and loads** all `.d.lunar` files in the same directory as your source file. This means:

- ✅ **Zero configuration** - just drop declaration files in your project
- ✅ **No imports needed** - types are available immediately
- ✅ **Multiple libraries** - all `.d.lunar` files are loaded
- ✅ **Type safety** - declarations are checked before your code

**How it works:**
1. When compiling `my_game.lunar`, the compiler scans the same directory
2. Finds all `*.d.lunar` files (e.g., `love.d.lunar`, `mylib.d.lunar`)
3. Parses and registers all declarations
4. Type-checks your code with those declarations available

## Creating Your Own Declaration Files

1. Create a `.d.lunar` file named after your library (e.g., `mylibrary.d.lunar`)
2. Use `declare` statements to define the library's API
3. Place the file in your project directory
4. **That's it!** The compiler will automatically find and use it

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
