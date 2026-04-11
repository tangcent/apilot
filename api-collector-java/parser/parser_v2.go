package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	java "github.com/tree-sitter/tree-sitter-java/bindings/go"
)

// ParserV2 provides enhanced Java source parsing with caching and logging.
type ParserV2 struct {
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

// NewParserV2 creates a new enhanced parser.
func NewParserV2(opts ParserOptions) (*ParserV2, error) {
	parser := tree_sitter.NewParser()
	language := tree_sitter.NewLanguage(java.Language())

	if err := parser.SetLanguage(language); err != nil {
		return nil, fmt.Errorf("failed to set language: %w", err)
	}

	cache, err := NewCache(opts.CacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	logger := NewLogger(opts.LogLevel)

	return &ParserV2{
		parser: parser,
		cache:  cache,
		logger: logger,
	}, nil
}

// ParseFile parses a single Java file with caching.
func (p *ParserV2) ParseFile(path string) (*ParseResult, error) {
	p.logger.Debug("Parsing file: %s", path)

	// Read file content
	source, err := os.ReadFile(path)
	if err != nil {
		return &ParseResult{
			FilePath: path,
			Error:    fmt.Errorf("failed to read file: %w", err),
		}, err
	}

	// Check cache
	if cached, found := p.cache.Get(path, source); found {
		p.logger.Debug("Cache hit for: %s", path)
		return cached, nil
	}

	p.logger.Debug("Cache miss for: %s", path)

	// Parse the file (tree-sitter parser is not thread-safe)
	p.mu.Lock()
	tree := p.parser.Parse(source, nil)
	p.mu.Unlock()

	if tree == nil {
		err := fmt.Errorf("failed to parse file")
		return &ParseResult{
			FilePath: path,
			Error:    err,
		}, err
	}
	defer tree.Close()

	// Extract classes
	classes, err := p.extractClasses(tree, source)
	if err != nil {
		return &ParseResult{
			FilePath: path,
			Error:    err,
		}, err
	}

	result := &ParseResult{
		FilePath: path,
		Classes:  classes,
		Error:    nil,
	}

	// Cache the result
	if cacheErr := p.cache.Set(path, source, result); cacheErr != nil {
		p.logger.Warn("Failed to cache result for %s: %v", path, cacheErr)
	}

	p.logger.Info("Parsed %s: %d classes found", filepath.Base(path), len(classes))
	return result, nil
}

// ParseDirectory parses all Java files in a directory.
func (p *ParserV2) ParseDirectory(dir string) ([]ParseResult, error) {
	p.logger.Info("Parsing directory: %s", dir)

	var javaFiles []string

	// Find all .java files
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".java" {
			javaFiles = append(javaFiles, path)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	p.logger.Info("Found %d Java files", len(javaFiles))

	// Parse files sequentially (parallel version in ParseDirectoryParallel)
	results := make([]ParseResult, 0, len(javaFiles))
	for _, file := range javaFiles {
		result, _ := p.ParseFile(file)
		results = append(results, *result)
	}

	return results, nil
}

// ParseDirectoryParallel parses all Java files in a directory using parallel goroutines.
func (p *ParserV2) ParseDirectoryParallel(dir string, workers int) ([]ParseResult, error) {
	p.logger.Info("Parsing directory (parallel, workers=%d): %s", workers, dir)

	var javaFiles []string

	// Find all .java files
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".java" {
			javaFiles = append(javaFiles, path)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	p.logger.Info("Found %d Java files", len(javaFiles))

	// Create worker pool
	fileChan := make(chan string, len(javaFiles))
	resultChan := make(chan ParseResult, len(javaFiles))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Each worker needs its own parser (tree-sitter is not thread-safe)
			workerParser, err := NewParserV2(ParserOptions{
				CacheDir: p.cache.cacheDir,
				LogLevel: p.logger.level,
			})
			if err != nil {
				p.logger.Error("Worker %d: failed to create parser: %v", workerID, err)
				return
			}
			defer workerParser.Close()

			for file := range fileChan {
				result, _ := workerParser.ParseFile(file)
				resultChan <- *result
			}
		}(i)
	}

	// Send files to workers
	for _, file := range javaFiles {
		fileChan <- file
	}
	close(fileChan)

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	results := make([]ParseResult, 0, len(javaFiles))
	for result := range resultChan {
		results = append(results, result)
	}

	p.logger.Info("Parsed %d files (%d with errors)", len(results), countErrors(results))
	return results, nil
}

// Close releases parser resources.
func (p *ParserV2) Close() {
	if p.parser != nil {
		p.parser.Close()
	}
}

// extractClasses extracts all classes from the AST (same as parser.go).
func (p *ParserV2) extractClasses(tree *tree_sitter.Tree, source []byte) ([]Class, error) {
	var classes []Class
	var packageName string

	rootNode := tree.RootNode()
	cursor := rootNode.Walk()
	defer cursor.Close()

	// Extract package name first
	for i := uint(0); i < rootNode.ChildCount(); i++ {
		child := rootNode.Child(i)
		if child.Kind() == "package_declaration" {
			packageName = extractPackageName(child, source)
			break
		}
	}

	// Walk the tree to find class declarations
	p.walkTree(cursor, source, func(node *tree_sitter.Node) {
		if node.Kind() == "class_declaration" {
			class := extractClass(node, source)
			class.Package = packageName
			classes = append(classes, class)
		}
	})

	return classes, nil
}

// walkTree walks the AST recursively.
func (p *ParserV2) walkTree(cursor *tree_sitter.TreeCursor, source []byte, callback func(*tree_sitter.Node)) {
	walkTreeWithDepth(cursor, source, callback, 0)
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
