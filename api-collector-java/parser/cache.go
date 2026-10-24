package parser

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// cacheSchemaVersion is folded into the cache key. Bump it whenever a Class,
// Method or Field gains information the parser did not previously extract:
// cached entries are keyed on file content alone, so an entry written by an
// older build would otherwise be replayed with the new field left empty and
// the missing information would look like source that does not carry it.
const cacheSchemaVersion = "2"

// Cache provides file-level caching based on content hash.
type Cache struct {
	cacheDir string
	mu       sync.RWMutex
	enabled  bool
}

// NewCache creates a new cache instance.
func NewCache(cacheDir string) (*Cache, error) {
	if cacheDir == "" {
		return &Cache{enabled: false}, nil
	}

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &Cache{
		cacheDir: cacheDir,
		enabled:  true,
	}, nil
}

// Get retrieves cached parse result for a file.
func (c *Cache) Get(filePath string, content []byte) (*ParseResult, bool) {
	if !c.enabled {
		return nil, false
	}

	hash := c.computeHash(content)
	cacheFile := c.getCacheFilePath(filePath, hash)

	// Hold lock while reading file to prevent TOCTOU race
	c.mu.RLock()
	data, err := os.ReadFile(cacheFile)
	c.mu.RUnlock()

	if err != nil {
		return nil, false
	}

	var result ParseResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, false
	}

	return &result, true
}

// Set stores parse result in cache.
func (c *Cache) Set(filePath string, content []byte, result *ParseResult) error {
	if !c.enabled {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	hash := c.computeHash(content)
	cacheFile := c.getCacheFilePath(filePath, hash)

	// Create cache subdirectory
	cacheSubDir := filepath.Dir(cacheFile)
	if err := os.MkdirAll(cacheSubDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache subdirectory: %w", err)
	}

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// Clear removes all cached entries.
func (c *Cache) Clear() error {
	if !c.enabled {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	return os.RemoveAll(c.cacheDir)
}

// computeHash computes SHA256 hash of file content and the cache schema.
func (c *Cache) computeHash(content []byte) string {
	hash := sha256.New()
	hash.Write([]byte(cacheSchemaVersion))
	hash.Write([]byte{0})
	hash.Write(content)
	return hex.EncodeToString(hash.Sum(nil))
}

// getCacheFilePath returns the cache file path for a given file and hash.
func (c *Cache) getCacheFilePath(filePath, hash string) string {
	// Use first 2 chars of hash as subdirectory for better distribution
	subDir := hash[:2]
	fileName := fmt.Sprintf("%s_%s.json", filepath.Base(filePath), hash)
	return filepath.Join(c.cacheDir, subDir, fileName)
}
