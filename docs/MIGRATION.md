# Lua to Lunar Migration Tool

The Lunar migration tool automatically converts existing Lua code to Lunar with type annotations.

## Quick Start

```bash
# Migrate a single file
lunar --migrate myfile.lua

# Migrate with custom output
lunar --migrate myfile.lua --migrate-output converted.lunar

# Migrate without type annotations
lunar --migrate myfile.lua --migrate-no-types

# Migrate directory (batch processing)
for file in *.lua; do
    lunar --migrate "$file"
done
```

## Features

### Automatic Type Inference

The migration tool analyzes your Lua code and infers types based on usage patterns:

#### Variable Types

```lua
-- Input (Lua)
local name = "Alice"
local age = 30
local isActive = true
local items = {1, 2, 3}

-- Output (Lunar)
local name: string = "Alice"
local age: number = 30
local isActive: boolean = true
local items: array<number> = {1, 2, 3}
```

#### Function Return Types

```lua
-- Input (Lua)
function greet(name)
    return "Hello, " .. name
end

function add(a, b)
    return a + b
end

-- Output (Lunar)
function greet(name): string
	return "Hello, " .. name
end

function add(a, b): number
	return a + b
end
```

#### Table Types

```lua
-- Input (Lua)
function createPerson(name, age)
    return {
        name = name,
        age = age
    }
end

-- Output (Lunar)
function createPerson(name, age): table<string, any>
	return {
		name = name,
		age = age
	}
end
```

### Type Inference Rules

The migration tool uses these heuristics to infer types:

| Pattern | Inferred Type |
|---------|---------------|
| `"string literal"` | `string` |
| `42`, `3.14` | `number` |
| `true`, `false` | `boolean` |
| `nil` | `nil` |
| `{1, 2, 3}` | `array<number>` |
| `{"a", "b"}` | `array<string>` |
| `{key = value}` | `table<string, any>` |
| `a + b` | `number` |
| `a .. b` | `string` |
| `a == b` | `boolean` |
| `function() end` | `function` |

### Const Detection

With `--migrate-const` (enabled by default), the tool attempts to detect immutable variables:

```lua
-- Input (Lua)
local PI = 3.14159
local config = {host = "localhost", port = 8080}

-- Output (Lunar with --migrate-const)
const PI: number = 3.14159
const config: table<string, any> = {host = "localhost", port = 8080}
```

## Command-Line Options

### `--migrate <file>`
**Required.** Specifies the Lua file to migrate.

```bash
lunar --migrate script.lua
```

### `--migrate-output <file>`
**Optional.** Specifies the output file path. Default: `<input>.lunar`

```bash
lunar --migrate script.lua --migrate-output output.lunar
```

### `--migrate-no-types`
**Optional.** Disables automatic type annotation addition.

```bash
lunar --migrate script.lua --migrate-no-types
```

Use this when you want to manually add types later or if the inferred types are incorrect.

### `--migrate-const`
**Optional.** Use `const` for variables that appear immutable. Default: `true`

```bash
lunar --migrate script.lua --migrate-const=false
```

## Migration Workflow

### Step 1: Prepare Your Code

Before migration:
1. **Backup your code** - Always keep a copy of the original
2. **Run your Lua tests** - Ensure the code works correctly
3. **Clean up** - Remove unused code and fix any linter warnings

### Step 2: Run Migration

```bash
lunar --migrate myapp.lua
```

This creates `myapp.lunar` with type annotations.

### Step 3: Review and Refine

The migrator makes best-effort guesses. Review the output:

```lunar
-- Generated code (may need refinement)
function processUser(user)
	local name = user.name
	local age = user.age
	return {name = name, age = age}
end

-- Better: Add explicit interface
interface User
	name: string
	age: number
end

function processUser(user: User): User
	local name: string = user.name
	local age: number = user.age
	return {name = name, age = age}
end
```

### Step 4: Compile and Test

```bash
# Compile the migrated code
lunar myapp.lunar

# Run tests
lua myapp.lua
```

Fix any type errors that arise.

### Step 5: Iterate

Refine types based on compiler feedback:

```lunar
-- If you see: "Cannot assign type 'nil' to variable 'x' of type 'string'"
-- Change from:
local x: string = someFunction()

-- To:
local x: string | nil = someFunction()
```

## Advanced Usage

### Migrating Large Codebases

For projects with many files:

```bash
#!/bin/bash
# migrate-all.sh

find . -name "*.lua" -type f | while read file; do
    echo "Migrating $file..."
    lunar --migrate "$file"
done

echo "Migration complete!"
```

### Custom Migration Script

```bash
#!/bin/bash
# custom-migrate.sh

INPUT_DIR="./lua-src"
OUTPUT_DIR="./lunar-src"

mkdir -p "$OUTPUT_DIR"

for file in "$INPUT_DIR"/*.lua; do
    filename=$(basename "$file" .lua)
    lunar --migrate "$file" --migrate-output "$OUTPUT_DIR/$filename.lunar"
done
```

### Selective Migration

Migrate specific modules first:

```bash
# Start with utility functions (least dependencies)
lunar --migrate utils.lua

# Then core modules
lunar --migrate core.lua

# Finally, main application
lunar --migrate main.lua
```

## Common Patterns

### Pattern 1: Module Migration

```lua
-- Input: mymodule.lua
local M = {}

function M.add(a, b)
    return a + b
end

function M.greet(name)
    return "Hello, " .. name
end

return M

-- Output: mymodule.lunar
local M = {}
function M.add(a, b): number
	return (a + b)
end
function M.greet(name): string
	return ("Hello, " .. name)
end
return M
```

After migration, you can enhance with interfaces:

```lunar
interface MyModule
	add: function(a: number, b: number): number
	greet: function(name: string): string
end

local M: MyModule = {}

function M.add(a: number, b: number): number
	return a + b
end

function M.greet(name: string): string
	return "Hello, " .. name
end

return M
```

### Pattern 2: Class-like Tables

```lua
-- Input: Person.lua
function Person:new(name, age)
    local obj = {
        name = name,
        age = age
    }
    setmetatable(obj, self)
    self.__index = self
    return obj
end

function Person:greet()
    print("Hello, I'm " .. self.name)
end

-- After migration, convert to Lunar class:
class Person
	name: string
	age: number

	function constructor(name: string, age: number)
		self.name = name
		self.age = age
	end

	function greet(): void
		print("Hello, I'm " .. self.name)
	end
end
```

### Pattern 3: Configuration Files

```lua
-- Input: config.lua
local config = {
    database = {
        host = "localhost",
        port = 5432,
        name = "mydb"
    },
    api = {
        timeout = 30,
        retries = 3
    }
}

return config

-- Enhanced with interfaces:
interface DatabaseConfig
	host: string
	port: number
	name: string
end

interface APIConfig
	timeout: number
	retries: number
end

interface Config
	database: DatabaseConfig
	api: APIConfig
end

const config: Config = {
	database = {
		host = "localhost",
		port = 5432,
		name = "mydb"
	},
	api = {
		timeout = 30,
		retries = 3
	}
}

return config
```

## Type Refinement Tips

### 1. Union Types for Optional Values

```lunar
-- Generated:
function findUser(id): table<string, any>
	-- ...
end

-- Better:
interface User
	id: number
	name: string
end

function findUser(id: number): User | nil
	-- Return nil if not found
end
```

### 2. Specific Array Types

```lunar
-- Generated:
local items = {1, 2, 3}  -- array<number>

-- Better (if items are IDs):
type UserID = number
local userIds: array<UserID> = {1, 2, 3}
```

### 3. Function Types

```lunar
-- Generated:
local callback = function(x) return x * 2 end

-- Better:
type NumberTransform = function(x: number): number
local callback: NumberTransform = function(x) return x * 2 end
```

### 4. Literal Types for Constants

```lunar
-- Generated:
local status = "active"

-- Better (using literal types):
type Status = "active" | "inactive" | "pending"
local status: Status = "active"
```

## Limitations

The migration tool has some limitations:

1. **Complex Type Inference**: Cannot infer complex generic types
   ```lunar
   -- May need manual refinement:
   function map(arr, fn)  -- Needs manual generic types
   	-- ...
   end

   -- Refine to:
   function map<T, U>(arr: array<T>, fn: function(T): U): array<U>
   	-- ...
   end
   ```

2. **Dynamic Table Access**: Cannot infer types for dynamic keys
   ```lunar
   local key = "name"
   local value = user[key]  -- Inferred as 'any'
   ```

3. **Metatables**: Metatable behavior is not analyzed
   ```lunar
   setmetatable(obj, mt)  -- Type of obj remains basic
   ```

4. **Conditional Types**: Cannot infer different types in branches
   ```lunar
   local result
   if condition then
   	result = "string"
   else
   	result = 42
   end
   -- Needs union type: string | number
   ```

## Best Practices

1. **Migrate incrementally** - Start with leaf modules (no dependencies)
2. **Test frequently** - Compile and run tests after each migration
3. **Add interfaces** - Define interfaces for table structures
4. **Review generated code** - Don't blindly accept all inferred types
5. **Use strict mode** - Enable `--migrate-const` to catch mutations
6. **Document assumptions** - Add comments for non-obvious type choices
7. **Create type aliases** - Use `type` for commonly used patterns

## Example: Complete Migration

Here's a complete before/after example:

**Before (user.lua):**
```lua
local M = {}

function M.createUser(name, email, age)
    return {
        name = name,
        email = email,
        age = age,
        createdAt = os.time()
    }
end

function M.validateEmail(email)
    return string.match(email, ".+@.+%.%w+") ~= nil
end

function M.isAdult(user)
    return user.age >= 18
end

return M
```

**After migration (user.lunar):**
```lunar
interface User
	name: string
	email: string
	age: number
	createdAt: number
end

interface UserModule
	createUser: function(name: string, email: string, age: number): User
	validateEmail: function(email: string): boolean
	isAdult: function(user: User): boolean
end

const M: UserModule = {}

function M.createUser(name: string, email: string, age: number): User
	return {
		name = name,
		email = email,
		age = age,
		createdAt = os.time()
	}
end

function M.validateEmail(email: string): boolean
	return string.match(email, ".+@.+%.%w+") ~= nil
end

function M.isAdult(user: User): boolean
	return user.age >= 18
end

return M
```

## Troubleshooting

### Issue: Too many 'any' types

**Solution**: The migrator is conservative. Manually refine types based on actual usage.

### Issue: Type errors after migration

**Solution**: This is expected! The migrator reveals type inconsistencies in your original Lua code. Fix them systematically.

### Issue: Lost comments

**Solution**: Currently, comments are not preserved during migration. Copy them manually or use diff tools.

### Issue: Formatting issues

**Solution**: Run the Lunar formatter:
```bash
lunar --format-write migrated.lunar
```

## Getting Help

- Check the [Lunar Type System Guide](./TYPE_SYSTEM.md)
- See [examples/](../examples/) for more migration examples
- Report issues at [github.com/your-repo/lunar/issues](https://github.com/your-repo/lunar/issues)

## Summary

The Lunar migration tool provides a solid starting point for converting Lua to Lunar. While it won't produce perfect code, it significantly reduces manual work and helps you adopt strong typing incrementally.

Happy migrating! 🚀
