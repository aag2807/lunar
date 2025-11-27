package pkgmgr

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// RunScript executes a script defined in lunar.json
func RunScript(dir, scriptName string, args []string) error {
	// Load manifest
	manifest, err := Load(dir)
	if err != nil {
		return fmt.Errorf("failed to load lunar.json: %w", err)
	}

	// Get script
	script, ok := manifest.GetScript(scriptName)
	if !ok {
		return fmt.Errorf("script '%s' not found in lunar.json", scriptName)
	}

	fmt.Printf("Running script: %s\n", scriptName)
	fmt.Printf("Command: %s\n", script)
	fmt.Println()

	// Parse and execute the script
	return executeScript(script, args, dir)
}

// executeScript executes a shell command
func executeScript(script string, args []string, dir string) error {
	// Append additional arguments to the script
	fullCommand := script
	if len(args) > 0 {
		fullCommand = script + " " + strings.Join(args, " ")
	}

	// Create command based on OS
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Use cmd.exe on Windows
		cmd = exec.Command("cmd", "/c", fullCommand)
	} else {
		// Use sh on Unix-like systems (Linux, macOS, etc.)
		cmd = exec.Command("sh", "-c", fullCommand)
	}

	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Execute
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("script failed: %w", err)
	}

	return nil
}

// ListScripts lists all available scripts in the manifest
func ListScripts(dir string) error {
	manifest, err := Load(dir)
	if err != nil {
		return fmt.Errorf("failed to load lunar.json: %w", err)
	}

	if len(manifest.Scripts) == 0 {
		fmt.Println("No scripts defined in lunar.json")
		return nil
	}

	fmt.Println("Available scripts:")
	for name, command := range manifest.Scripts {
		fmt.Printf("  %s\n    %s\n", name, command)
	}

	return nil
}
