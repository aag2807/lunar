package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Config represents the lunar.config.json configuration
type Config struct {
	// Compiler options
	CompilerOptions CompilerOptions `json:"compilerOptions"`

	// Entry points for bundling
	Entry string `json:"entry,omitempty"`

	// Output configuration
	OutDir  string `json:"outDir,omitempty"`
	OutFile string `json:"outFile,omitempty"`

	// Files to include/exclude
	Include []string `json:"include,omitempty"`
	Exclude []string `json:"exclude,omitempty"`
}

// CompilerOptions contains compiler-specific settings
type CompilerOptions struct {
	// Type checking
	Strict      bool `json:"strict,omitempty"`
	NoTypeCheck bool `json:"noTypeCheck,omitempty"`

	// Output options
	SourceMap   bool `json:"sourceMap,omitempty"`
	Declaration bool `json:"declaration,omitempty"`

	// Target Lua version: "lua51", "lua52", "lua53", "lua54", "luajit"
	Target string `json:"target,omitempty"`

	// Module resolution
	BaseURL string            `json:"baseUrl,omitempty"`
	Paths   map[string]string `json:"paths,omitempty"`

	// Bundling
	Bundle    bool `json:"bundle,omitempty"`
	TreeShake bool `json:"treeShake,omitempty"`
}

// Target constants
const (
	TargetLua51  = "lua51"
	TargetLua52  = "lua52"
	TargetLua53  = "lua53"
	TargetLua54  = "lua54"
	TargetLuaJIT = "luajit"
)

// IsLuaJIT returns true if target is LuaJIT or Lua 5.1
func (c *CompilerOptions) IsLuaJIT() bool {
	return c.Target == TargetLuaJIT || c.Target == TargetLua51
}

// DefaultConfig returns a config with default values
func DefaultConfig() *Config {
	return &Config{
		CompilerOptions: CompilerOptions{
			Strict:      false,
			NoTypeCheck: false,
			SourceMap:   false,
			Declaration: false,
			Target:      TargetLua53,
			BaseURL:     ".",
			Paths:       make(map[string]string),
			Bundle:      false,
			TreeShake:   false,
		},
		Include: []string{"**/*.lunar"},
		Exclude: []string{"node_modules", "*.d.lunar"},
	}
}

// Load loads configuration from lunar.config.json in the given directory
func Load(dir string) (*Config, error) {
	configPath := filepath.Join(dir, "lunar.config.json")

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	// Read config file
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Parse JSON
	config := DefaultConfig()
	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}

// LoadFromFile loads configuration from a specific file path
func LoadFromFile(configPath string) (*Config, error) {
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	// Read config file
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// Parse JSON
	config := DefaultConfig()
	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}

// FindConfig searches for lunar.config.json starting from dir and going up
func FindConfig(dir string) (string, *Config, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", nil, err
	}

	// Walk up the directory tree
	for {
		configPath := filepath.Join(absDir, "lunar.config.json")
		if _, err := os.Stat(configPath); err == nil {
			config, err := LoadFromFile(configPath)
			if err != nil {
				return "", nil, err
			}
			return absDir, config, nil
		}

		// Move to parent directory
		parent := filepath.Dir(absDir)
		if parent == absDir {
			// Reached root, no config found
			return "", DefaultConfig(), nil
		}
		absDir = parent
	}
}

// ResolvePath resolves a path using the config's path aliases
func (c *Config) ResolvePath(importPath string, fromDir string) string {
	// Check path aliases
	for alias, target := range c.CompilerOptions.Paths {
		// Handle wildcard aliases like "@/*" -> "src/*"
		if len(alias) > 1 && alias[len(alias)-1] == '*' {
			prefix := alias[:len(alias)-1]
			if len(importPath) > len(prefix) && importPath[:len(prefix)] == prefix {
				suffix := importPath[len(prefix):]
				targetPrefix := target
				if len(target) > 1 && target[len(target)-1] == '*' {
					targetPrefix = target[:len(target)-1]
				}
				resolved := filepath.Join(c.CompilerOptions.BaseURL, targetPrefix+suffix)
				return resolved
			}
		} else if importPath == alias {
			return filepath.Join(c.CompilerOptions.BaseURL, target)
		}
	}

	return importPath
}

// Merge merges command-line options into the config
func (c *Config) Merge(noTypeCheck, sourceMap, bundle bool, target string) {
	if noTypeCheck {
		c.CompilerOptions.NoTypeCheck = true
	}
	if sourceMap {
		c.CompilerOptions.SourceMap = true
	}
	if bundle {
		c.CompilerOptions.Bundle = true
	}
	if target != "" {
		c.CompilerOptions.Target = target
	}
}
