package pkgmgr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Manifest represents the lunar.json package manifest
type Manifest struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description,omitempty"`
	Main        string            `json:"main,omitempty"`
	Scripts     map[string]string `json:"scripts,omitempty"`
	Dependencies    map[string]string `json:"dependencies,omitempty"`
	DevDependencies map[string]string `json:"devDependencies,omitempty"`
	CompilerOptions *CompilerOptions  `json:"compilerOptions,omitempty"`
	Keywords    []string          `json:"keywords,omitempty"`
	Author      string            `json:"author,omitempty"`
	License     string            `json:"license,omitempty"`
	Repository  *Repository       `json:"repository,omitempty"`
}

// CompilerOptions contains compiler settings embedded in lunar.json
type CompilerOptions struct {
	Target     string            `json:"target,omitempty"`
	Strict     bool              `json:"strict,omitempty"`
	SourceMap  bool              `json:"sourceMap,omitempty"`
	Bundle     bool              `json:"bundle,omitempty"`
	TreeShake  bool              `json:"treeShake,omitempty"`
	BaseURL    string            `json:"baseUrl,omitempty"`
	Paths      map[string]string `json:"paths,omitempty"`
}

// Repository contains repository information
type Repository struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// DefaultManifest returns a manifest with default values
func DefaultManifest(name string) *Manifest {
	if name == "" {
		name = "my-project"
	}

	return &Manifest{
		Name:        name,
		Version:     "1.0.0",
		Description: "",
		Main:        "main.lunar",
		Scripts: map[string]string{
			"build": "lunar --bundle main.lunar -o dist/bundle.lua",
			"dev":   "lunar --watch --run main.lunar",
			"test":  "lunar --test tests/",
		},
		Dependencies:    make(map[string]string),
		DevDependencies: make(map[string]string),
		CompilerOptions: &CompilerOptions{
			Target:    "lua53",
			Strict:    false,
			SourceMap: false,
			Bundle:    false,
			TreeShake: true,
			BaseURL:   ".",
			Paths:     make(map[string]string),
		},
		Keywords:   []string{},
		Author:     "",
		License:    "MIT",
		Repository: nil,
	}
}

// Load loads a lunar.json manifest from the given directory
func Load(dir string) (*Manifest, error) {
	manifestPath := filepath.Join(dir, "lunar.json")

	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("lunar.json not found in %s", dir)
	}

	data, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read lunar.json: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse lunar.json: %w", err)
	}

	return &manifest, nil
}

// LoadOrDefault loads a manifest or returns a default one if not found
func LoadOrDefault(dir string) (*Manifest, error) {
	manifest, err := Load(dir)
	if err != nil {
		// Return default if file doesn't exist
		if os.IsNotExist(err) {
			cwd, _ := os.Getwd()
			name := filepath.Base(cwd)
			return DefaultManifest(name), nil
		}
		return nil, err
	}
	return manifest, nil
}

// Save writes the manifest to lunar.json in the given directory
func (m *Manifest) Save(dir string) error {
	manifestPath := filepath.Join(dir, "lunar.json")

	// Marshal with indentation
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Write to file
	if err := ioutil.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write lunar.json: %w", err)
	}

	return nil
}

// FindManifest searches for lunar.json starting from dir and going up
func FindManifest(dir string) (string, *Manifest, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", nil, err
	}

	// Walk up the directory tree
	for {
		manifestPath := filepath.Join(absDir, "lunar.json")
		if _, err := os.Stat(manifestPath); err == nil {
			manifest, err := Load(absDir)
			if err != nil {
				return "", nil, err
			}
			return absDir, manifest, nil
		}

		// Move to parent directory
		parent := filepath.Dir(absDir)
		if parent == absDir {
			// Reached root, no manifest found
			return "", nil, fmt.Errorf("no lunar.json found")
		}
		absDir = parent
	}
}

// AddDependency adds a dependency to the manifest
func (m *Manifest) AddDependency(name, version string, dev bool) {
	if dev {
		if m.DevDependencies == nil {
			m.DevDependencies = make(map[string]string)
		}
		m.DevDependencies[name] = version
	} else {
		if m.Dependencies == nil {
			m.Dependencies = make(map[string]string)
		}
		m.Dependencies[name] = version
	}
}

// RemoveDependency removes a dependency from the manifest
func (m *Manifest) RemoveDependency(name string) {
	if m.Dependencies != nil {
		delete(m.Dependencies, name)
	}
	if m.DevDependencies != nil {
		delete(m.DevDependencies, name)
	}
}

// GetScript returns a script by name
func (m *Manifest) GetScript(name string) (string, bool) {
	if m.Scripts == nil {
		return "", false
	}
	script, ok := m.Scripts[name]
	return script, ok
}

// HasDependency checks if a dependency exists
func (m *Manifest) HasDependency(name string) bool {
	if m.Dependencies != nil {
		if _, ok := m.Dependencies[name]; ok {
			return true
		}
	}
	if m.DevDependencies != nil {
		if _, ok := m.DevDependencies[name]; ok {
			return true
		}
	}
	return false
}
