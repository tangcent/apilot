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

	// Parse the file
	tree := p.parser.Parse(source, nil)
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

	rootNode := tree.RootNode()
	cursor := rootNode.Walk()
	defer cursor.Close()

	// Walk the tree to find class declarations
	p.walkTree(cursor, source, func(node *tree_sitter.Node) {
		if node.Kind() == "class_declaration" {
			class := p.extractClass(node, source)
			classes = append(classes, class)
		}
	})

	return classes, nil
}

// extractClass extracts class information (delegates to original parser.go logic).
func (p *ParserV2) extractClass(node *tree_sitter.Node, source []byte) Class {
	class := Class{}

	// Extract class name
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "identifier" {
			class.Name = child.Utf8Text(source)
			break
		}
	}

	// Extract class annotations
	class.Annotations = p.extractAnnotations(node, source)

	// Extract methods
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "class_body" {
			class.Methods = p.extractMethods(child, source)
			break
		}
	}

	return class
}

// extractAnnotations extracts annotations from a node.
func (p *ParserV2) extractAnnotations(node *tree_sitter.Node, source []byte) []Annotation {
	var annotations []Annotation

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "modifiers" {
			for j := uint(0); j < child.ChildCount(); j++ {
				modChild := child.Child(j)
				if modChild.Kind() == "marker_annotation" || modChild.Kind() == "annotation" {
					ann := p.extractAnnotation(modChild, source)
					annotations = append(annotations, ann)
				}
			}
		}
	}

	return annotations
}

// extractAnnotation extracts a single annotation.
func (p *ParserV2) extractAnnotation(node *tree_sitter.Node, source []byte) Annotation {
	ann := Annotation{
		Params: make(map[string]string),
	}

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		switch child.Kind() {
		case "identifier", "scoped_identifier":
			ann.Name = child.Utf8Text(source)
		case "annotation_argument_list":
			p.extractAnnotationParams(child, source, ann.Params)
		}
	}

	return ann
}

// extractAnnotationParams extracts annotation parameters.
func (p *ParserV2) extractAnnotationParams(node *tree_sitter.Node, source []byte, params map[string]string) {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		if child.Kind() == "element_value_pair" {
			var key, value string
			for j := uint(0); j < child.ChildCount(); j++ {
				subChild := child.Child(j)
				if subChild.Kind() == "identifier" {
					key = subChild.Utf8Text(source)
				} else if subChild.Kind() == "string_literal" {
					value = p.extractStringLiteral(subChild, source)
				}
			}
			if key != "" {
				params[key] = value
			}
		} else if child.Kind() == "string_literal" {
			params["value"] = p.extractStringLiteral(child, source)
		}
	}
}

// extractStringLiteral extracts string content without quotes.
func (p *ParserV2) extractStringLiteral(node *tree_sitter.Node, source []byte) string {
	text := node.Utf8Text(source)
	if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		return text[1 : len(text)-1]
	}
	return text
}

// extractMethods extracts all methods from a class body.
func (p *ParserV2) extractMethods(classBody *tree_sitter.Node, source []byte) []Method {
	var methods []Method

	for i := uint(0); i < classBody.ChildCount(); i++ {
		child := classBody.Child(i)
		if child.Kind() == "method_declaration" {
			method := p.extractMethod(child, source)
			methods = append(methods, method)
		}
	}

	return methods
}

// extractMethod extracts method information.
func (p *ParserV2) extractMethod(node *tree_sitter.Node, source []byte) Method {
	method := Method{}

	method.Annotations = p.extractAnnotations(node, source)

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		switch child.Kind() {
		case "identifier":
			method.Name = child.Utf8Text(source)
		case "type_identifier", "generic_type":
			method.ReturnType = child.Utf8Text(source)
		case "formal_parameters":
			method.Parameters = p.extractParameters(child, source)
		}
	}

	return method
}

// extractParameters extracts method parameters.
func (p *ParserV2) extractParameters(node *tree_sitter.Node, source []byte) []Parameter {
	var params []Parameter

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "formal_parameter" {
			param := p.extractParameter(child, source)
			params = append(params, param)
		}
	}

	return params
}

// extractParameter extracts a single parameter.
func (p *ParserV2) extractParameter(node *tree_sitter.Node, source []byte) Parameter {
	param := Parameter{}

	param.Annotations = p.extractAnnotations(node, source)

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		switch child.Kind() {
		case "type_identifier", "generic_type", "integral_type":
			param.Type = child.Utf8Text(source)
		case "identifier":
			param.Name = child.Utf8Text(source)
		}
	}

	return param
}

// walkTree walks the AST recursively.
func (p *ParserV2) walkTree(cursor *tree_sitter.TreeCursor, source []byte, callback func(*tree_sitter.Node)) {
	node := cursor.Node()
	callback(node)

	if cursor.GotoFirstChild() {
		p.walkTree(cursor, source, callback)
		cursor.GotoParent()
	}

	if cursor.GotoNextSibling() {
		p.walkTree(cursor, source, callback)
	}
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
