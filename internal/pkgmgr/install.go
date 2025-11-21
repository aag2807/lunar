package pkgmgr

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PackageManager handles package installation and management
type PackageManager struct {
	ProjectRoot string
	ModulesDir  string
}

// NewPackageManager creates a new package manager instance
func NewPackageManager(projectRoot string) *PackageManager {
	return &PackageManager{
		ProjectRoot: projectRoot,
		ModulesDir:  filepath.Join(projectRoot, ".lunar"),
	}
}

// Add adds a package to the project
func (pm *PackageManager) Add(packageSpec string, dev bool) error {
	// Parse package spec (name@version or git URL)
	name, source, version, err := pm.parsePackageSpec(packageSpec)
	if err != nil {
		return err
	}

	// Load manifest
	manifest, err := Load(pm.ProjectRoot)
	if err != nil {
		return fmt.Errorf("failed to load lunar.json: %w", err)
	}

	// Add dependency to manifest
	manifest.AddDependency(name, version, dev)

	// Save manifest
	if err := manifest.Save(pm.ProjectRoot); err != nil {
		return fmt.Errorf("failed to save lunar.json: %w", err)
	}

	// Install the package
	if err := pm.installPackage(name, source, version); err != nil {
		return fmt.Errorf("failed to install package: %w", err)
	}

	fmt.Printf("✓ Added %s@%s\n", name, version)
	return nil
}

// Remove removes a package from the project
func (pm *PackageManager) Remove(packageName string) error {
	// Load manifest
	manifest, err := Load(pm.ProjectRoot)
	if err != nil {
		return fmt.Errorf("failed to load lunar.json: %w", err)
	}

	// Check if package exists
	if !manifest.HasDependency(packageName) {
		return fmt.Errorf("package %s is not installed", packageName)
	}

	// Remove from manifest
	manifest.RemoveDependency(packageName)

	// Save manifest
	if err := manifest.Save(pm.ProjectRoot); err != nil {
		return fmt.Errorf("failed to save lunar.json: %w", err)
	}

	// Remove from .lunar directory
	packagePath := filepath.Join(pm.ModulesDir, packageName)
	if err := os.RemoveAll(packagePath); err != nil {
		fmt.Printf("Warning: failed to remove package files: %v\n", err)
	}

	fmt.Printf("✓ Removed %s\n", packageName)
	return nil
}

// Install installs all dependencies from lunar.json
func (pm *PackageManager) Install() error {
	// Load manifest
	manifest, err := Load(pm.ProjectRoot)
	if err != nil {
		return fmt.Errorf("failed to load lunar.json: %w", err)
	}

	// Create .lunar directory if it doesn't exist
	if err := os.MkdirAll(pm.ModulesDir, 0755); err != nil {
		return fmt.Errorf("failed to create .lunar directory: %w", err)
	}

	// Install dependencies
	totalDeps := len(manifest.Dependencies) + len(manifest.DevDependencies)
	if totalDeps == 0 {
		fmt.Println("No dependencies to install")
		return nil
	}

	fmt.Printf("Installing %d package(s)...\n", totalDeps)

	// Install regular dependencies
	for name, version := range manifest.Dependencies {
		fmt.Printf("  Installing %s@%s...\n", name, version)
		source := pm.resolveSource(name, version)
		if err := pm.installPackage(name, source, version); err != nil {
			return fmt.Errorf("failed to install %s: %w", name, err)
		}
	}

	// Install dev dependencies
	for name, version := range manifest.DevDependencies {
		fmt.Printf("  Installing %s@%s (dev)...\n", name, version)
		source := pm.resolveSource(name, version)
		if err := pm.installPackage(name, source, version); err != nil {
			return fmt.Errorf("failed to install %s: %w", name, err)
		}
	}

	fmt.Println("✓ All packages installed successfully")
	return nil
}

// parsePackageSpec parses a package specification
// Supports: name@version, git-url, github:user/repo
func (pm *PackageManager) parsePackageSpec(spec string) (name, source, version string, err error) {
	// Git URL (https://github.com/user/repo.git)
	if strings.HasPrefix(spec, "https://") || strings.HasPrefix(spec, "git@") {
		parts := strings.Split(spec, "/")
		if len(parts) < 2 {
			return "", "", "", fmt.Errorf("invalid git URL: %s", spec)
		}
		repoName := parts[len(parts)-1]
		repoName = strings.TrimSuffix(repoName, ".git")
		return repoName, spec, "git", nil
	}

	// GitHub shorthand (github:user/repo or user/repo)
	if strings.Contains(spec, "/") {
		spec = strings.TrimPrefix(spec, "github:")
		parts := strings.Split(spec, "/")
		if len(parts) != 2 {
			return "", "", "", fmt.Errorf("invalid GitHub spec: %s", spec)
		}
		name = parts[1]
		source = fmt.Sprintf("https://github.com/%s.git", spec)
		return name, source, "git", nil
	}

	// Name@version
	if strings.Contains(spec, "@") {
		parts := strings.SplitN(spec, "@", 2)
		name = parts[0]
		version = parts[1]
		source = pm.resolveSource(name, version)
		return name, source, version, nil
	}

	// Just name (default to latest)
	name = spec
	version = "latest"
	source = pm.resolveSource(name, version)
	return name, source, version, nil
}

// resolveSource resolves where to get a package from
func (pm *PackageManager) resolveSource(name, version string) string {
	// For now, assume packages are on GitHub in lunar-lang org
	// In the future, we could have a registry or support LuaRocks

	// If version is a git URL or hash, use it directly
	if strings.HasPrefix(version, "https://") || strings.HasPrefix(version, "git@") {
		return version
	}

	// Default: try lunar-lang GitHub organization
	return fmt.Sprintf("https://github.com/lunar-lang/%s.git", name)
}

// installPackage installs a single package
func (pm *PackageManager) installPackage(name, source, version string) error {
	packagePath := filepath.Join(pm.ModulesDir, name)

	// Remove existing package if it exists
	if _, err := os.Stat(packagePath); err == nil {
		if err := os.RemoveAll(packagePath); err != nil {
			return fmt.Errorf("failed to remove existing package: %w", err)
		}
	}

	// Install based on source type
	if version == "git" || strings.HasPrefix(source, "https://") || strings.HasPrefix(source, "git@") {
		return pm.installFromGit(name, source, packagePath)
	}

	// Try LuaRocks if available
	if pm.isLuaRocksAvailable() {
		return pm.installFromLuaRocks(name, version, packagePath)
	}

	return fmt.Errorf("package source not supported: %s", source)
}

// installFromGit clones a git repository
func (pm *PackageManager) installFromGit(name, gitURL, destPath string) error {
	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Clone repository
	cmd := exec.Command("git", "clone", "--depth", "1", gitURL, destPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	// Remove .git directory to save space
	gitDir := filepath.Join(destPath, ".git")
	os.RemoveAll(gitDir)

	return nil
}

// installFromLuaRocks installs from LuaRocks
func (pm *PackageManager) installFromLuaRocks(name, version, destPath string) error {
	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Install with luarocks
	versionSpec := name
	if version != "latest" && version != "" {
		versionSpec = fmt.Sprintf("%s %s", name, version)
	}

	cmd := exec.Command("luarocks", "install", "--tree", destPath, versionSpec)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("luarocks install failed: %w", err)
	}

	return nil
}

// isLuaRocksAvailable checks if luarocks is installed
func (pm *PackageManager) isLuaRocksAvailable() bool {
	cmd := exec.Command("luarocks", "--version")
	return cmd.Run() == nil
}

// Clean removes all installed packages
func (pm *PackageManager) Clean() error {
	if err := os.RemoveAll(pm.ModulesDir); err != nil {
		return fmt.Errorf("failed to remove .lunar directory: %w", err)
	}
	fmt.Println("✓ Cleaned .lunar directory")
	return nil
}
