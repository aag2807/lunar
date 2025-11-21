package pkgmgr

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// InitOptions contains options for initializing a new project
type InitOptions struct {
	Name        string
	Version     string
	Description string
	Main        string
	Target      string
	Strict      bool
	Author      string
	License     string
	SkipPrompts bool // -y flag
}

// Init creates a new lunar.json manifest interactively
func Init(dir string, opts *InitOptions) error {
	// Check if lunar.json already exists
	manifestPath := filepath.Join(dir, "lunar.json")
	if _, err := os.Stat(manifestPath); err == nil {
		return fmt.Errorf("lunar.json already exists in %s", dir)
	}

	// Get directory name for default project name
	dirName := filepath.Base(dir)
	if dirName == "." {
		cwd, _ := os.Getwd()
		dirName = filepath.Base(cwd)
	}

	manifest := DefaultManifest(dirName)

	// Skip prompts if -y flag is set
	if opts.SkipPrompts {
		applyOptions(manifest, opts, dirName)
		return manifest.Save(dir)
	}

	// Interactive prompts
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Creating lunar.json...")
	fmt.Println()

	// Project name
	name := prompt(scanner, "Project name", dirName)
	if name != "" {
		manifest.Name = name
	}

	// Version
	version := prompt(scanner, "Version", "1.0.0")
	if version != "" {
		manifest.Version = version
	}

	// Description
	description := prompt(scanner, "Description", "")
	manifest.Description = description

	// Entry point
	main := prompt(scanner, "Entry point", "main.lunar")
	if main != "" {
		manifest.Main = main
	}

	// Target Lua version
	target := prompt(scanner, "Target Lua version (lua51/lua52/lua53/lua54/luajit)", "lua53")
	if target != "" {
		validTargets := map[string]bool{
			"lua51": true, "lua52": true, "lua53": true,
			"lua54": true, "luajit": true,
		}
		if validTargets[target] {
			manifest.CompilerOptions.Target = target
		}
	}

	// Strict mode
	strictInput := prompt(scanner, "Enable strict mode? (y/N)", "N")
	manifest.CompilerOptions.Strict = strings.ToLower(strictInput) == "y"

	// Author
	author := prompt(scanner, "Author", "")
	manifest.Author = author

	// License
	license := prompt(scanner, "License", "MIT")
	if license != "" {
		manifest.License = license
	}

	fmt.Println()

	// Apply any command-line options (they override prompts)
	applyOptions(manifest, opts, dirName)

	// Save the manifest
	if err := manifest.Save(dir); err != nil {
		return err
	}

	fmt.Printf("Created lunar.json in %s ✓\n", dir)
	return nil
}

// prompt displays a prompt and returns the user input or default value
func prompt(scanner *bufio.Scanner, message, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s: (%s) ", message, defaultValue)
	} else {
		fmt.Printf("%s: ", message)
	}

	scanner.Scan()
	input := strings.TrimSpace(scanner.Text())

	if input == "" {
		return defaultValue
	}
	return input
}

// applyOptions applies command-line options to the manifest
func applyOptions(manifest *Manifest, opts *InitOptions, defaultName string) {
	if opts.Name != "" {
		manifest.Name = opts.Name
	}
	if opts.Version != "" {
		manifest.Version = opts.Version
	}
	if opts.Description != "" {
		manifest.Description = opts.Description
	}
	if opts.Main != "" {
		manifest.Main = opts.Main
	}
	if opts.Target != "" {
		manifest.CompilerOptions.Target = opts.Target
	}
	if opts.Strict {
		manifest.CompilerOptions.Strict = true
	}
	if opts.Author != "" {
		manifest.Author = opts.Author
	}
	if opts.License != "" {
		manifest.License = opts.License
	}
}
