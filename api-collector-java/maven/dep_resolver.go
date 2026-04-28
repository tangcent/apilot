package maven

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

type MavenDependencyResolver struct {
	mu      sync.Mutex
	parser  *parser.Parser
	cache   map[string]*parser.Class
	misses  map[string]bool
}

func NewMavenDependencyResolver() (*MavenDependencyResolver, error) {
	if !isCLIAvailable() {
		return nil, fmt.Errorf("maven-indexer-cli not available")
	}

	p, err := parser.NewParser(parser.ParserOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create java parser: %w", err)
	}

	return &MavenDependencyResolver{
		parser: p,
		cache:  make(map[string]*parser.Class),
		misses: make(map[string]bool),
	}, nil
}

func (r *MavenDependencyResolver) Close() {
	if r.parser != nil {
		r.parser.Close()
	}
}

func (r *MavenDependencyResolver) ResolveClass(className string) *parser.Class {
	r.mu.Lock()
	defer r.mu.Unlock()

	if cached, ok := r.cache[className]; ok {
		return cached
	}

	if r.misses[className] {
		return nil
	}

	class := r.resolveFromCLI(className)
	if class != nil {
		r.cache[className] = class
		return class
	}

	r.misses[className] = true
	return nil
}

func (r *MavenDependencyResolver) resolveFromCLI(className string) *parser.Class {
	source, err := r.fetchSource(className)
	if err != nil || source == "" {
		return nil
	}

	classes, err := r.parser.ParseSource([]byte(source))
	if err != nil || len(classes) == 0 {
		return nil
	}

	for i := range classes {
		if classes[i].Name == className {
			return &classes[i]
		}
	}

	if len(classes) > 0 {
		return &classes[0]
	}

	return nil
}

func (r *MavenDependencyResolver) fetchSource(className string) (string, error) {
	cmd := exec.Command(cliName, "get-class", className, "--json", "--type", "source")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("maven-indexer-cli get-class failed: %w", err)
	}

	var raw string
	if err := json.Unmarshal(output, &raw); err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" || strings.HasPrefix(trimmed, "Class") {
			return "", nil
		}
		return trimmed, nil
	}

	if raw == "" {
		return "", nil
	}

	source, err := extractSourceFromMarkdown(raw)
	if err != nil {
		log.Printf("[maven-dep] failed to extract source for %s: %v", className, err)
		return raw, nil
	}

	return source, nil
}

func extractSourceFromMarkdown(raw string) (string, error) {
	codeBlockStart := "```java\n"
	codeBlockEnd := "```"

	startIdx := strings.Index(raw, codeBlockStart)
	if startIdx == -1 {
		return raw, nil
	}

	afterStart := startIdx + len(codeBlockStart)
	remaining := raw[afterStart:]

	endIdx := strings.Index(remaining, codeBlockEnd)
	if endIdx == -1 {
		return remaining, nil
	}

	source := remaining[:endIdx]
	return strings.TrimSuffix(source, "\n"), nil
}
