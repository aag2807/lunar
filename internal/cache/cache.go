package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// CacheEntry represents a cached compilation result
type CacheEntry struct {
	SourcePath    string            `json:"sourcePath"`
	SourceHash    string            `json:"sourceHash"`
	CompiledCode  string            `json:"compiledCode"`
	Dependencies  []string          `json:"dependencies"`
	DepHashes     map[string]string `json:"depHashes"`
	CompileTime   time.Time         `json:"compileTime"`
	SourceMapData string            `json:"sourceMapData,omitempty"`
}

// Cache manages incremental compilation cache
type Cache struct {
	cacheDir string
	entries  map[string]*CacheEntry
}

// New creates a new cache instance
func New(cacheDir string) (*Cache, error) {
	if cacheDir == "" {
		// Default to .lunar-cache in current directory
		cacheDir = ".lunar-cache"
	}

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	cache := &Cache{
		cacheDir: cacheDir,
		entries:  make(map[string]*CacheEntry),
	}

	// Load existing cache entries
	if err := cache.load(); err != nil {
		// If loading fails, start with empty cache (not an error)
		cache.entries = make(map[string]*CacheEntry)
	}

	return cache, nil
}

// Get retrieves a cached entry if it's still valid
func (c *Cache) Get(sourcePath string) (*CacheEntry, bool) {
	// Compute hash of source file
	sourceHash, err := hashFile(sourcePath)
	if err != nil {
		return nil, false
	}

	entry, exists := c.entries[sourcePath]
	if !exists {
		return nil, false
	}

	// Check if source file changed
	if entry.SourceHash != sourceHash {
		return nil, false
	}

	// Check if dependencies changed
	for _, depPath := range entry.Dependencies {
		depHash, err := hashFile(depPath)
		if err != nil {
			return nil, false
		}

		if entry.DepHashes[depPath] != depHash {
			return nil, false
		}
	}

	return entry, true
}

// Put stores a compilation result in the cache
func (c *Cache) Put(sourcePath string, compiledCode string, dependencies []string, sourceMapData string) error {
	sourceHash, err := hashFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to hash source file: %w", err)
	}

	depHashes := make(map[string]string)
	for _, depPath := range dependencies {
		depHash, err := hashFile(depPath)
		if err != nil {
			// If dependency can't be hashed, skip it
			continue
		}
		depHashes[depPath] = depHash
	}

	entry := &CacheEntry{
		SourcePath:    sourcePath,
		SourceHash:    sourceHash,
		CompiledCode:  compiledCode,
		Dependencies:  dependencies,
		DepHashes:     depHashes,
		CompileTime:   time.Now(),
		SourceMapData: sourceMapData,
	}

	c.entries[sourcePath] = entry
	return c.save()
}

// Clear removes all cache entries
func (c *Cache) Clear() error {
	c.entries = make(map[string]*CacheEntry)
	return os.RemoveAll(c.cacheDir)
}

// load reads cache entries from disk
func (c *Cache) load() error {
	cacheFile := filepath.Join(c.cacheDir, "cache.json")
	data, err := ioutil.ReadFile(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No cache file yet
		}
		return err
	}

	return json.Unmarshal(data, &c.entries)
}

// save writes cache entries to disk
func (c *Cache) save() error {
	cacheFile := filepath.Join(c.cacheDir, "cache.json")
	data, err := json.MarshalIndent(c.entries, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(cacheFile, data, 0644)
}

// hashFile computes SHA256 hash of a file
func hashFile(path string) (string, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash), nil
}

// Stats returns cache statistics
func (c *Cache) Stats() map[string]interface{} {
	totalSize := int64(0)
	for _, entry := range c.entries {
		totalSize += int64(len(entry.CompiledCode))
		totalSize += int64(len(entry.SourceMapData))
	}

	return map[string]interface{}{
		"entries":    len(c.entries),
		"totalSize":  totalSize,
		"cacheDir":   c.cacheDir,
		"cachedSize": fmt.Sprintf("%.2f KB", float64(totalSize)/1024),
	}
}
