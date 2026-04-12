// Package parser provides Tree-sitter based Java/Kotlin source parsing.
package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	java "github.com/tree-sitter/tree-sitter-java/bindings/go"
)

const (
	// maxWalkDepth prevents stack overflow in deeply nested AST structures.
	maxWalkDepth = 1000
)

// Parser provides Java source parsing with caching, logging, and parallel support.
type Parser struct {
	parser *tree_sitter.Parser
	cache  *Cache
	logger *Logger
	mu     sync.Mutex // Protects parser access (tree-sitter is not thread-safe)
}

// ParserOptions configures parser behavior.
type ParserOptions struct {
	CacheDir string
	LogLevel LogLevel
}

// NewParser creates a new Java parser.
func NewParser(opts ParserOptions) (*Parser, error) {
	p := tree_sitter.NewParser()
	language := tree_sitter.NewLanguage(java.Language())

	if err := p.SetLanguage(language); err != nil {
		return nil, fmt.Errorf("failed to set language: %w", err)
	}

	cache, err := NewCache(opts.CacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	return &Parser{
		parser: p,
		cache:  cache,
		logger: NewLogger(opts.LogLevel),
	}, nil
}

// ParseFile parses a single Java file with caching.
func (p *Parser) ParseFile(path string) (*ParseResult, error) {
	p.logger.Debug("Parsing file: %s", path)

	source, err := os.ReadFile(path)
	if err != nil {
		return &ParseResult{FilePath: path, Error: fmt.Errorf("failed to read file: %w", err)}, err
	}

	if cached, found := p.cache.Get(path, source); found {
		p.logger.Debug("Cache hit for: %s", path)
		return cached, nil
	}

	p.logger.Debug("Cache miss for: %s", path)

	p.mu.Lock()
	tree := p.parser.Parse(source, nil)
	p.mu.Unlock()

	if tree == nil {
		err := fmt.Errorf("failed to parse file")
		return &ParseResult{FilePath: path, Error: err}, err
	}
	defer tree.Close()

	classes, err := extractAllClasses(tree, source)
	if err != nil {
		return &ParseResult{FilePath: path, Error: err}, err
	}

	result := &ParseResult{FilePath: path, Classes: classes}

	if cacheErr := p.cache.Set(path, source, result); cacheErr != nil {
		p.logger.Warn("Failed to cache result for %s: %v", path, cacheErr)
	}

	p.logger.Info("Parsed %s: %d classes found", filepath.Base(path), len(classes))
	return result, nil
}

// ParseDirectory parses all Java files in a directory sequentially.
func (p *Parser) ParseDirectory(dir string) ([]ParseResult, error) {
	p.logger.Info("Parsing directory: %s", dir)

	javaFiles, err := findJavaFiles(dir)
	if err != nil {
		return nil, err
	}

	p.logger.Info("Found %d Java files", len(javaFiles))

	results := make([]ParseResult, 0, len(javaFiles))
	for _, file := range javaFiles {
		result, _ := p.ParseFile(file)
		results = append(results, *result)
	}

	return results, nil
}

// ParseDirectoryParallel parses all Java files in a directory using a worker pool.
func (p *Parser) ParseDirectoryParallel(dir string, workers int) ([]ParseResult, error) {
	p.logger.Info("Parsing directory (parallel, workers=%d): %s", workers, dir)

	javaFiles, err := findJavaFiles(dir)
	if err != nil {
		return nil, err
	}

	p.logger.Info("Found %d Java files", len(javaFiles))

	fileChan := make(chan string, len(javaFiles))
	resultChan := make(chan ParseResult, len(javaFiles))
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Each worker needs its own parser (tree-sitter is not thread-safe)
			wp, err := NewParser(ParserOptions{
				CacheDir: p.cache.cacheDir,
				LogLevel: p.logger.level,
			})
			if err != nil {
				p.logger.Error("Worker %d: failed to create parser: %v", workerID, err)
				return
			}
			defer wp.Close()

			for file := range fileChan {
				result, _ := wp.ParseFile(file)
				resultChan <- *result
			}
		}(i)
	}

	for _, file := range javaFiles {
		fileChan <- file
	}
	close(fileChan)

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	results := make([]ParseResult, 0, len(javaFiles))
	for result := range resultChan {
		results = append(results, result)
	}

	p.logger.Info("Parsed %d files (%d with errors)", len(results), countErrors(results))
	return results, nil
}

// Close releases parser resources.
func (p *Parser) Close() {
	if p.parser != nil {
		p.parser.Close()
	}
}

// findJavaFiles returns all .java files under dir.
func findJavaFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".java" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}
	return files, nil
}

// extractAllClasses extracts classes and interfaces from a parsed tree.
func extractAllClasses(tree *tree_sitter.Tree, source []byte) ([]Class, error) {
	var classes []Class
	var packageName string

	rootNode := tree.RootNode()
	cursor := rootNode.Walk()
	defer cursor.Close()

	for i := uint(0); i < rootNode.ChildCount(); i++ {
		child := rootNode.Child(i)
		if child.Kind() == "package_declaration" {
			packageName = extractPackageName(child, source)
			break
		}
	}

	walkTreeWithDepth(cursor, source, func(node *tree_sitter.Node) {
		switch node.Kind() {
		case "class_declaration":
			class := extractClass(node, source)
			class.Package = packageName
			classes = append(classes, class)
		case "interface_declaration":
			iface := extractInterface(node, source)
			iface.Package = packageName
			classes = append(classes, iface)
		}
	}, 0)

	return classes, nil
}

// countErrors counts parse results with errors.
func countErrors(results []ParseResult) int {
	count := 0
	for _, r := range results {
		if r.Error != nil {
			count++
		}
	}
	return count
}
