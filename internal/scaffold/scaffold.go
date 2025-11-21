package scaffold

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Template represents a project template
type Template struct {
	Name        string
	Description string
	Files       map[string]string
}

// GetAvailableTemplates returns all available project templates
func GetAvailableTemplates() map[string]Template {
	return map[string]Template{
		"basic": {
			Name:        "basic",
			Description: "Basic Lunar project with minimal setup",
			Files: map[string]string{
				"main.lunar": basicMainFile,
				"lunar.json": basicConfigFile,
				".gitignore": gitignoreFile,
				"README.md":  basicReadmeFile,
			},
		},
		"cli": {
			Name:        "cli",
			Description: "Command-line application with argument parsing",
			Files: map[string]string{
				"main.lunar":     cliMainFile,
				"src/args.lunar": cliArgsFile,
				"src/commands.lunar": cliCommandsFile,
				"lunar.json":     cliConfigFile,
				".gitignore":     gitignoreFile,
				"README.md":      cliReadmeFile,
			},
		},
		"web": {
			Name:        "web",
			Description: "Web server/API project",
			Files: map[string]string{
				"main.lunar":        webMainFile,
				"src/router.lunar":  webRouterFile,
				"src/handlers.lunar": webHandlersFile,
				"lunar.json":        webConfigFile,
				".gitignore":        gitignoreFile,
				"README.md":         webReadmeFile,
			},
		},
		"library": {
			Name:        "library",
			Description: "Reusable library/module",
			Files: map[string]string{
				"src/init.lunar": libraryInitFile,
				"src/types.lunar": libraryTypesFile,
				"test/init_test.lunar": libraryTestFile,
				"lunar.json":     libraryConfigFile,
				".gitignore":     gitignoreFile,
				"README.md":      libraryReadmeFile,
			},
		},
	}
}

// CreateProject scaffolds a new project from a template
func CreateProject(projectName, templateName, targetDir string) error {
	templates := GetAvailableTemplates()
	template, ok := templates[templateName]
	if !ok {
		return fmt.Errorf("unknown template '%s'. Available templates: %s",
			templateName, strings.Join(getTemplateNames(), ", "))
	}

	// Create project directory
	projectDir := filepath.Join(targetDir, projectName)
	if _, err := os.Stat(projectDir); err == nil {
		return fmt.Errorf("directory '%s' already exists", projectDir)
	}

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	fmt.Printf("Creating %s project '%s'...\n", template.Description, projectName)

	// Create files from template
	for filePath, content := range template.Files {
		fullPath := filepath.Join(projectDir, filePath)

		// Create parent directories if needed
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory '%s': %w", dir, err)
		}

		// Replace placeholders in content
		content = strings.ReplaceAll(content, "{{PROJECT_NAME}}", projectName)

		// Write file
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file '%s': %w", fullPath, err)
		}

		fmt.Printf("  Created %s\n", filePath)
	}

	fmt.Printf("\n✓ Project created successfully!\n\n")
	fmt.Printf("Next steps:\n")
	fmt.Printf("  cd %s\n", projectName)

	switch templateName {
	case "basic":
		fmt.Printf("  lunar main.lunar\n")
	case "cli":
		fmt.Printf("  lunar run main.lunar --help\n")
	case "web":
		fmt.Printf("  lunar add http\n")
		fmt.Printf("  lunar run main.lunar\n")
	case "library":
		fmt.Printf("  lunar test test/\n")
	}

	return nil
}

func getTemplateNames() []string {
	templates := GetAvailableTemplates()
	names := make([]string, 0, len(templates))
	for name := range templates {
		names = append(names, name)
	}
	return names
}

// GenerateConfig generates a lunar.json config file
func GenerateConfig(projectName, entryPoint string, strict bool) (string, error) {
	config := map[string]interface{}{
		"name":    projectName,
		"version": "0.1.0",
		"entry":   entryPoint,
		"target":  "lua54",
		"strict":  strict,
		"dependencies": map[string]string{},
		"devDependencies": map[string]string{},
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// Template file contents

const basicMainFile = `-- {{PROJECT_NAME}}
-- A simple Lunar application

local function main()
    print("Hello from Lunar!")
    print("Edit main.lunar to get started.")
end

main()
`

const basicConfigFile = `{
  "name": "{{PROJECT_NAME}}",
  "version": "0.1.0",
  "entry": "main.lunar",
  "target": "lua54",
  "strict": false,
  "dependencies": {},
  "devDependencies": {}
}
`

const basicReadmeFile = `# {{PROJECT_NAME}}

A Lunar project.

## Getting Started

Run the application:
` + "```bash\nlunar main.lunar\n```" + `

Build to Lua:
` + "```bash\nlunar -o build/main.lua main.lunar\n```" + `

## Project Structure

- ` + "`main.lunar`" + ` - Entry point
- ` + "`lunar.json`" + ` - Project configuration

## Learn More

- [Lunar Documentation](https://github.com/yourusername/lunar)
`

const cliMainFile = `-- {{PROJECT_NAME}}
-- A command-line application built with Lunar

import "src/args"
import "src/commands"

local function main()
    local args = Args.parse()

    if args.help then
        Commands.showHelp()
        return
    end

    if args.version then
        print("{{PROJECT_NAME}} v0.1.0")
        return
    end

    -- Execute the command
    local command = args[1] or "help"
    Commands.execute(command, args)
end

main()
`

const cliArgsFile = `-- Argument parsing utilities

export class Args {
    static function parse(): table
        local args = {}
        local i = 1

        while i <= #arg do
            local current = arg[i]

            if current:sub(1, 2) == "--" then
                local key = current:sub(3)
                args[key] = true

                -- Check if next arg is a value
                if i < #arg and arg[i + 1]:sub(1, 1) ~= "-" then
                    args[key] = arg[i + 1]
                    i = i + 1
                end
            elseif current:sub(1, 1) == "-" then
                local key = current:sub(2)
                args[key] = true
            else
                table.insert(args, current)
            end

            i = i + 1
        end

        return args
    end
}
`

const cliCommandsFile = `-- Command implementations

export class Commands {
    static function showHelp()
        print("{{PROJECT_NAME}} - A Lunar CLI application")
        print("")
        print("Usage: lunar run main.lunar [command] [options]")
        print("")
        print("Commands:")
        print("  help        Show this help message")
        print("  version     Show version information")
        print("")
        print("Options:")
        print("  --help      Show help")
        print("  --version   Show version")
    end

    static function execute(command: string, args: table)
        if command == "help" then
            Commands.showHelp()
        else
            print("Unknown command: " .. command)
            print("Run with --help for usage information")
        end
    end
}
`

const cliConfigFile = `{
  "name": "{{PROJECT_NAME}}",
  "version": "0.1.0",
  "entry": "main.lunar",
  "target": "lua54",
  "strict": true,
  "dependencies": {},
  "devDependencies": {}
}
`

const cliReadmeFile = `# {{PROJECT_NAME}}

A command-line application built with Lunar.

## Getting Started

Run the CLI:
` + "```bash\nlunar run main.lunar --help\n```" + `

Build to executable:
` + "```bash\nlunar -o build/cli.lua main.lunar\n```" + `

## Project Structure

- ` + "`main.lunar`" + ` - Entry point
- ` + "`src/args.lunar`" + ` - Argument parsing
- ` + "`src/commands.lunar`" + ` - Command implementations
- ` + "`lunar.json`" + ` - Project configuration

## Adding Commands

Edit ` + "`src/commands.lunar`" + ` to add new commands:

` + "```lunar\nstatic function execute(command: string, args: table)\n    if command == \"mycommand\" then\n        -- Your command logic\n    end\nend\n```" + `
`

const webMainFile = `-- {{PROJECT_NAME}}
-- A web server built with Lunar

import "src/router"
import "src/handlers"

local function main()
    print("Starting {{PROJECT_NAME}} server...")
    print("Server will listen on http://localhost:8080")
    print("Add your HTTP library and implement the server logic!")

    -- TODO: Integrate with HTTP library (e.g., lua-http, lapis, etc.)
    -- local server = Router.create()
    -- server:start(8080)
end

main()
`

const webRouterFile = `-- HTTP router

export class Router {
    routes: table

    function new()
        self.routes = {}
    end

    function get(self, path: string, handler: function)
        self.routes[path] = { method = "GET", handler = handler }
    end

    function post(self, path: string, handler: function)
        self.routes[path] = { method = "POST", handler = handler }
    end

    static function create(): Router
        local router = Router.new()

        -- Register routes
        router:get("/", Handlers.index)
        router:get("/api/health", Handlers.health)

        return router
    end

    function start(self, port: number)
        print("Server listening on port " .. port)
        -- TODO: Start actual HTTP server
    end
}
`

const webHandlersFile = `-- HTTP request handlers

export class Handlers {
    static function index(req: table): table
        return {
            status = 200,
            body = "Welcome to {{PROJECT_NAME}}!"
        }
    end

    static function health(req: table): table
        return {
            status = 200,
            body = '{"status":"ok"}'
        }
    end
}
`

const webConfigFile = `{
  "name": "{{PROJECT_NAME}}",
  "version": "0.1.0",
  "entry": "main.lunar",
  "target": "lua54",
  "strict": true,
  "dependencies": {},
  "devDependencies": {}
}
`

const webReadmeFile = `# {{PROJECT_NAME}}

A web server built with Lunar.

## Getting Started

Install an HTTP library (choose one):
` + "```bash\nlunar add lua-http\n# or\nlunar add lapis\n```" + `

Run the server:
` + "```bash\nlunar run main.lunar\n```" + `

## Project Structure

- ` + "`main.lunar`" + ` - Entry point
- ` + "`src/router.lunar`" + ` - HTTP routing
- ` + "`src/handlers.lunar`" + ` - Request handlers
- ` + "`lunar.json`" + ` - Project configuration

## Adding Routes

Edit ` + "`src/router.lunar`" + ` to add new routes:

` + "```lunar\nrouter:get(\"/api/users\", Handlers.getUsers)\nrouter:post(\"/api/users\", Handlers.createUser)\n```" + `

Implement handlers in ` + "`src/handlers.lunar`" + `.
`

const libraryInitFile = `-- {{PROJECT_NAME}}
-- A reusable Lunar library

import "src/types"

export class {{PROJECT_NAME}} {
    version: string

    function new()
        self.version = "0.1.0"
    end

    -- Add your library functions here
    static function hello(name: string): string
        return "Hello, " .. name .. " from {{PROJECT_NAME}}!"
    end
}
`

const libraryTypesFile = `-- Type definitions for {{PROJECT_NAME}}

export interface Config {
    debug: boolean
    timeout: number
}

export type Result<T> = {
    success: boolean,
    value: T?,
    error: string?
}
`

const libraryTestFile = `-- Tests for {{PROJECT_NAME}}

import "src/init"

local function test_hello()
    local result = {{PROJECT_NAME}}.hello("World")
    assert(result == "Hello, World from {{PROJECT_NAME}}!", "hello() should return greeting")
    print("✓ test_hello passed")
end

local function run_tests()
    print("Running tests for {{PROJECT_NAME}}...")
    test_hello()
    print("\nAll tests passed!")
end

run_tests()
`

const libraryConfigFile = `{
  "name": "{{PROJECT_NAME}}",
  "version": "0.1.0",
  "entry": "src/init.lunar",
  "target": "lua54",
  "strict": true,
  "dependencies": {},
  "devDependencies": {}
}
`

const libraryReadmeFile = `# {{PROJECT_NAME}}

A reusable Lunar library.

## Installation

` + "```bash\nlunar add {{PROJECT_NAME}}\n```" + `

## Usage

` + "```lunar\nimport \"{{PROJECT_NAME}}\"\n\nlocal result = {{PROJECT_NAME}}.hello(\"World\")\nprint(result)\n```" + `

## Development

Run tests:
` + "```bash\nlunar test test/\n```" + `

Build:
` + "```bash\nlunar -o build/init.lua src/init.lunar\n```" + `

## Project Structure

- ` + "`src/init.lunar`" + ` - Main library file
- ` + "`src/types.lunar`" + ` - Type definitions
- ` + "`test/`" + ` - Test files
- ` + "`lunar.json`" + ` - Project configuration

## API Documentation

### ` + "`{{PROJECT_NAME}}.hello(name: string): string`" + `

Returns a greeting message.

` + "```lunar\nlocal greeting = {{PROJECT_NAME}}.hello(\"Alice\")\n-- Returns: \"Hello, Alice from {{PROJECT_NAME}}!\"\n```" + `
`

const gitignoreFile = `# Compiled Lua files
*.lua
*.lua.map

# Build directory
build/
dist/

# Dependencies
.lunar/
node_modules/

# Cache
.lunar-cache/

# Editor files
.vscode/
.idea/
*.swp
*.swo
*~

# OS files
.DS_Store
Thumbs.db

# Logs
*.log
`
