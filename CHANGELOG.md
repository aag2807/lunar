# Changelog

All notable changes to Lunar will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2024-11-10

### Added - Core Language Features
- **Type System** - Complete static type checking system
  - Primitive types: `string`, `number`, `boolean`, `nil`, `any`, `void`
  - Complex types: arrays (`T[]`), tables (`table<K,V>`), tuples
  - Union types: `T1 | T2`
  - Literal types: string and number literals
  - Optional types: `T?` (shorthand for `T | nil`)
  - Type aliases with `type` keyword

- **Object-Oriented Programming**
  - Classes with constructors and methods
  - Inheritance with `extends` keyword
  - Access modifiers: `public`, `private`, `protected`
  - Interfaces with structural typing
  - Interface inheritance with `extends`
  - Abstract type checking

- **Generics**
  - Generic classes: `class Box<T>`
  - Generic functions: `function map<T, U>(...)`
  - Generic interfaces: `interface Container<T>`
  - Type parameter constraints

- **Enums**
  - Numeric enums: `enum Direction { North, South }`
  - String enums with explicit values
  - Enum member access and type checking

- **Advanced Type Features**
  - Object types: `type Point { x: number, y: number }`
  - Function types: `(a: number) => string`
  - Const enforcement: `const` keyword for immutable variables
  - Type guards: `value is Type` syntax

### Added - Tooling & Developer Experience

- **Declaration Files (`.d.lunar`)**
  - `declare` keyword for ambient declarations
  - Automatic discovery and loading of declaration files
  - Support for declaring functions, classes, interfaces, types, and enums
  - Zero-configuration - just place `.d.lunar` files in your project

- **Standard Library Declarations**
  - `lua.d.lunar` - Core global functions (print, tostring, tonumber, etc.)
  - `math.d.lunar` - Mathematical functions (sin, cos, random, floor, etc.)
  - `io.d.lunar` - File I/O operations (open, read, write, File interface)
  - `os.d.lunar` - OS facilities (time, execute, date, DateTable type)
  - Comprehensive type coverage for Lua 5.1 stdlib

- **lunar2decl Tool**
  - Command-line tool to generate `.d.lunar` files from Lua code
  - Extracts function signatures automatically
  - Handles parameters and varargs
  - Starting point for typing existing Lua libraries

- **Enhanced Error Messages**
  - Clear file:line:column location format
  - Source code context showing surrounding lines
  - Visual caret (^) pointing to exact error location
  - Numbered errors with clean formatting
  - Helpful, descriptive error messages

- **Compiler Features**
  - Clean, readable Lua output
  - Preserves code structure and comments
  - Optional type checking with `--no-type-check` flag
  - Custom output file with `-o` flag
  - Version and help information

### Fixed

- **Parser Bug** - Fixed dot expression precedence issue that prevented method calls like `math.sin(x)` from working correctly. Previously parsed the entire call as the right side of the dot expression; now correctly parses only the identifier.

### Known Limitations

- **Keyword Conflicts** - `string`, `table`, and `type` are type keywords and cannot be used as identifiers. This prevents declaring the Lua `string.*` and `table.*` standard library functions. Workaround: These functions still work in generated Lua code, just without type checking. Full support planned for v1.1 with context-aware keywords.

- **Module-Style Functions** - The `lunar2decl` tool currently skips functions like `module.func()` that would require interface declarations. These must be manually declared using interfaces.

### Technical Details

- **Compiler Architecture**
  - Lexer: Token-based scanning with keyword recognition
  - Parser: Recursive descent parser generating typed AST
  - Type Checker: Environment-based scope tracking with type inference
  - Code Generator: Direct Lua code emission

- **Type System Implementation**
  - Structural typing for interfaces and object types
  - Nominal typing for classes and enums
  - Type compatibility checking with `IsAssignableTo()` method
  - Generic type instantiation and constraint checking

## [Unreleased] - Planned for v1.1

### Planned
- Context-aware keywords to enable full `string.*` and `table.*` stdlib support
- Enhanced error messages with "Did you mean...?" suggestions
- Additional standard library declarations
- Performance optimizations for large codebases

## [Unreleased] - Future (v2.0+)

### Planned
- Language Server Protocol (LSP) implementation for IDE support
- Source maps for debugging compiled code
- Package manager integration
- Code formatter tool
- Watch mode for continuous compilation
- Incremental compilation

---

## Version History

- **1.0.0** (2024-11-10) - Initial release with complete type system, OOP, generics, stdlib declarations, and tooling
