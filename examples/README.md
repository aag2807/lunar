# Lunar Examples

This directory contains example code demonstrating various Lunar language features.

## Examples

### class.lunar
**Basic OOP demonstration**

Shows how to create classes with:
- Private properties
- Constructors
- Public methods

A simple `Point` class with x/y coordinates.

```bash
lunar class.lunar
lua class.lua
```

### love_game_example.lunar
**LÖVE 2D Game Framework Integration**

Demonstrates using Lunar with the LÖVE game framework:
- Player movement with keyboard input
- Game state management
- Drawing graphics
- Type-safe game development

Requires the `love.d.lunar` declaration file for type checking.

```bash
lunar love_game_example.lunar
love love_game_example.lua  # Requires LÖVE 2D installed
```

### love.d.lunar
**Declaration file for LÖVE 2D Framework**

Type declarations for the LÖVE game framework including:
- Graphics API (`love.graphics.*`)
- Keyboard input (`love.keyboard.*`)
- Game callbacks (`load`, `update`, `draw`, etc.)

This is an example of how to create declaration files for external Lua libraries.

## Declaration Files

Declaration files (`.d.lunar`) provide type information for existing Lua libraries:

### lua.d.lunar
Copy of the standard library core globals declarations. Provides types for:
- `print()`, `tostring()`, `tonumber()`
- `assert()`, `error()`
- `pairs()`, `ipairs()`
- And more...

Place this in your project directory for automatic stdlib type checking.

## Running Examples

1. **Build Lunar compiler** (if not already built):
   ```bash
   cd .. && make build
   ```

2. **Compile an example**:
   ```bash
   ../lunar class.lunar
   ```

3. **Run the generated Lua**:
   ```bash
   lua class.lua
   ```

## Creating Your Own Examples

Use these examples as templates for your own Lunar projects. Key patterns demonstrated:

- **Type Safety**: All variables and function parameters are typed
- **OOP**: Classes with encapsulation and methods
- **External Libraries**: Using declaration files for Lua libraries
- **Clean Output**: Generated Lua code is readable and efficient

## Tips

- Start with `class.lunar` to understand basic syntax
- Copy `lua.d.lunar` to your project for stdlib support
- Use `love.d.lunar` as a template for creating your own declaration files
- Run `lunar --help` to see all compiler options

## Need Help?

- Read the [Language Specification](../LANGUAGE_SPEC.md)
- Check the [Main README](../README.md) for installation and usage
- See [Standard Library Declarations](../stdlib/README.md) for stdlib types
