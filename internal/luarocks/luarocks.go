package luarocks

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Package represents a LuaRocks package
type Package struct {
	Name        string
	Version     string
	Description string
	Homepage    string
	License     string
	Modules     map[string]string // module name -> file path
}

// Manager handles LuaRocks package installation and type generation
type Manager struct {
	LuaRocksPath string
	TypesDir     string
}

// New creates a new LuaRocks manager
func New() (*Manager, error) {
	// Check if luarocks is installed
	luarocksPath, err := exec.LookPath("luarocks")
	if err != nil {
		return nil, fmt.Errorf("luarocks not found in PATH. Please install LuaRocks first")
	}

	// Default types directory
	typesDir := filepath.Join(".", "types")

	return &Manager{
		LuaRocksPath: luarocksPath,
		TypesDir:     typesDir,
	}, nil
}

// Install installs a LuaRocks package
func (m *Manager) Install(packageName string, version string) error {
	args := []string{"install", packageName}
	if version != "" {
		args = append(args, version)
	}

	cmd := exec.Command(m.LuaRocksPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install %s: %w", packageName, err)
	}

	fmt.Printf("✓ Successfully installed %s\n", packageName)
	return nil
}

// Remove removes a LuaRocks package
func (m *Manager) Remove(packageName string) error {
	cmd := exec.Command(m.LuaRocksPath, "remove", packageName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove %s: %w", packageName, err)
	}

	fmt.Printf("✓ Successfully removed %s\n", packageName)
	return nil
}

// List lists installed LuaRocks packages
func (m *Manager) List() ([]Package, error) {
	cmd := exec.Command(m.LuaRocksPath, "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list packages: %w", err)
	}

	var packages []Package
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {
			packages = append(packages, Package{
				Name:    parts[0],
				Version: parts[1],
			})
		}
	}

	return packages, nil
}

// Search searches for LuaRocks packages
func (m *Manager) Search(query string) ([]Package, error) {
	cmd := exec.Command(m.LuaRocksPath, "search", query, "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to search packages: %w", err)
	}

	var packages []Package
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {
			packages = append(packages, Package{
				Name:    parts[0],
				Version: parts[1],
			})
		}
	}

	return packages, nil
}

// Show gets detailed information about a package
func (m *Manager) Show(packageName string) (*Package, error) {
	cmd := exec.Command(m.LuaRocksPath, "show", packageName)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to show package info: %w", err)
	}

	pkg := &Package{
		Name:    packageName,
		Modules: make(map[string]string),
	}

	lines := strings.Split(string(output), "\n")
	inModulesSection := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "Version:") {
			pkg.Version = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
		} else if strings.HasPrefix(line, "Description:") {
			pkg.Description = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
		} else if strings.HasPrefix(line, "Homepage:") {
			pkg.Homepage = strings.TrimSpace(strings.TrimPrefix(line, "Homepage:"))
		} else if strings.HasPrefix(line, "License:") {
			pkg.License = strings.TrimSpace(strings.TrimPrefix(line, "License:"))
		} else if strings.HasPrefix(line, "Modules:") {
			inModulesSection = true
		} else if inModulesSection && strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				moduleName := strings.TrimSpace(parts[0])
				modulePath := strings.TrimSpace(parts[1])
				pkg.Modules[moduleName] = modulePath
			}
		}
	}

	return pkg, nil
}

// GenerateTypeDefinitions generates .d.lunar files for an installed package
func (m *Manager) GenerateTypeDefinitions(packageName string) error {
	pkg, err := m.Show(packageName)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(m.TypesDir, 0755); err != nil {
		return fmt.Errorf("failed to create types directory: %w", err)
	}

	// Generate type definition file
	typeDefPath := filepath.Join(m.TypesDir, packageName+".d.lunar")
	content := m.generateTypeDefinitionContent(pkg)

	if err := ioutil.WriteFile(typeDefPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write type definition: %w", err)
	}

	fmt.Printf("✓ Generated type definitions: %s\n", typeDefPath)
	return nil
}

// generateTypeDefinitionContent generates the content for a .d.lunar file
func (m *Manager) generateTypeDefinitionContent(pkg *Package) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("-- Type definitions for %s\n", pkg.Name))
	if pkg.Version != "" {
		sb.WriteString(fmt.Sprintf("-- Version: %s\n", pkg.Version))
	}
	if pkg.Description != "" {
		sb.WriteString(fmt.Sprintf("-- %s\n", pkg.Description))
	}
	if pkg.Homepage != "" {
		sb.WriteString(fmt.Sprintf("-- %s\n", pkg.Homepage))
	}
	sb.WriteString("\n")

	// Generate module declarations
	for moduleName := range pkg.Modules {
		sb.WriteString(fmt.Sprintf("-- Module: %s\n", moduleName))
		sb.WriteString(fmt.Sprintf("declare interface %sLib\n", toInterfaceName(moduleName)))
		sb.WriteString("    -- TODO: Add type definitions for this module\n")
		sb.WriteString("    -- Run 'lunar types:generate " + pkg.Name + "' to auto-generate types\n")
		sb.WriteString("end\n\n")
		sb.WriteString(fmt.Sprintf("declare const %s: %sLib\n\n", moduleName, toInterfaceName(moduleName)))
	}

	return sb.String()
}

// toInterfaceName converts a module name to a valid interface name
func toInterfaceName(moduleName string) string {
	// Replace dots and hyphens with underscores
	name := strings.ReplaceAll(moduleName, ".", "_")
	name = strings.ReplaceAll(name, "-", "_")

	// Capitalize first letter
	if len(name) > 0 {
		name = strings.ToUpper(name[:1]) + name[1:]
	}

	return name
}

// Config represents the lunarocks.json configuration file
type Config struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// LoadConfig loads the lunarocks.json configuration
func (m *Manager) LoadConfig(configPath string) (*Config, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// SaveConfig saves the lunarocks.json configuration
func (m *Manager) SaveConfig(configPath string, config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := ioutil.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// InstallDependencies installs all dependencies from lunarocks.json
func (m *Manager) InstallDependencies(configPath string) error {
	config, err := m.LoadConfig(configPath)
	if err != nil {
		return err
	}

	fmt.Println("Installing dependencies...")

	// Install regular dependencies
	for name, version := range config.Dependencies {
		if err := m.Install(name, version); err != nil {
			return err
		}
		if err := m.GenerateTypeDefinitions(name); err != nil {
			fmt.Printf("Warning: failed to generate types for %s: %v\n", name, err)
		}
	}

	// Install dev dependencies
	for name, version := range config.DevDependencies {
		if err := m.Install(name, version); err != nil {
			return err
		}
		if err := m.GenerateTypeDefinitions(name); err != nil {
			fmt.Printf("Warning: failed to generate types for %s: %v\n", name, err)
		}
	}

	fmt.Println("✓ All dependencies installed")
	return nil
}
