package maven

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
)

// maxSourceLines is the line budget given to explore. It is far above any
// class: a source cut to fit a budget reads as a class missing its last
// members.
const maxSourceLines = "100000"

// SourceParser reads a dependency source written in a language other than
// Java into the Java declaration model. name is the class the source was
// fetched for, for messages.
type SourceParser func(name string, source []byte) ([]parser.Class, error)

var (
	sourceParsersMu sync.RWMutex
	sourceParsers   = map[string]SourceParser{}
)

// RegisterSourceParser makes dependency sources in language readable. The
// indexer reports the language of each source it returns: "kotlin" for a
// class from a Kotlin sources JAR, "java" for Java sources and for classes it
// decompiled. The collector of another JVM language registers its parser here,
// so this module never imports that collector.
func RegisterSourceParser(language string, p SourceParser) {
	sourceParsersMu.Lock()
	defer sourceParsersMu.Unlock()
	sourceParsers[language] = p
}

func registeredSourceParser(language string) SourceParser {
	sourceParsersMu.RLock()
	defer sourceParsersMu.RUnlock()
	return sourceParsers[language]
}

// MavenDependencyResolver reads classes declared in a project's dependencies
// through `maven-indexer-cli explore`, which pins each class to the version
// the project's dependency tree names.
type MavenDependencyResolver struct {
	mu          sync.Mutex
	parser      *parser.Parser
	run         runner
	projectPath string
	cache       map[string]*parser.Class
	misses      map[string]bool
	// unpinned maps a class the indexer could not pin to the project's
	// dependency versions to the artifact it was read from.
	unpinned map[string]string
}

func NewMavenDependencyResolver() (*MavenDependencyResolver, error) {
	p, err := parser.NewParser(parser.ParserOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create java parser: %w", err)
	}

	return &MavenDependencyResolver{
		parser:   p,
		run:      runCLI,
		cache:    make(map[string]*parser.Class),
		misses:   make(map[string]bool),
		unpinned: make(map[string]string),
	}, nil
}

// SetProjectPath names the project whose dependency tree pins class versions.
func (r *MavenDependencyResolver) SetProjectPath(projectPath string) {
	r.projectPath = projectPath
}

func (r *MavenDependencyResolver) Close() {
	if r.parser != nil {
		r.parser.Close()
	}
}

// ResolvedCount reports how many distinct classes have been resolved from
// dependencies so far. It is read after Collect finishes, once all parsers
// are done querying.
func (r *MavenDependencyResolver) ResolvedCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.cache)
}

// Unpinned lists the resolved classes the indexer could not pin to the
// project's dependency versions, each as "class (groupId:artifactId:version)"
// with the artifact it was read from instead. Either the project's tree does
// not name their artifact, or the version it names is not indexed.
func (r *MavenDependencyResolver) Unpinned() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.unpinned))
	for class, artifact := range r.unpinned {
		out = append(out, fmt.Sprintf("%s (%s)", class, artifact))
	}
	sort.Strings(out)
	return out
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

// exploredClass is one class of an explore answer.
type exploredClass struct {
	Source   string
	Language string
	Artifact string
	// Unpinned is set when the project's dependency tree was known but the
	// indexer still read the class from a version of its own choosing.
	Unpinned bool
}

type exploreJSON struct {
	Classes []struct {
		ClassName string `json:"className"`
		Artifact  string `json:"artifact"`
		Source    string `json:"source"`
		Language  string `json:"language"`
	} `json:"classes"`
	Meta struct {
		Resolved map[string]struct {
			Artifact string `json:"artifact"`
			Policy   string `json:"policy"`
		} `json:"resolved"`
		Project struct {
			Resolution string `json:"resolution"`
		} `json:"project"`
	} `json:"_meta"`
}

func (r *MavenDependencyResolver) resolveFromCLI(className string) *parser.Class {
	explored, err := r.explore(className)
	if err != nil || explored == nil || explored.Source == "" {
		return nil
	}

	classes, err := r.parse(className, explored.Language, explored.Source)
	if err != nil {
		return nil
	}

	class := pickClass(classes, className)
	if class != nil && explored.Unpinned {
		r.unpinned[className] = explored.Artifact
	}
	return class
}

// explore fetches the source of one class, pinned to the project's dependency
// tree when a project path is set.
func (r *MavenDependencyResolver) explore(className string) (*exploredClass, error) {
	args := []string{"--json", "explore", className, "--include", "source", "--max-lines", maxSourceLines}
	if r.projectPath != "" {
		args = append(args, "--project-path", r.projectPath)
	}
	output, err := r.run(args...)
	if err != nil {
		return nil, fmt.Errorf("%s explore failed: %w", cliName, err)
	}

	var result exploreJSON
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("%s explore: unexpected output: %w", cliName, err)
	}
	if len(result.Classes) == 0 {
		return nil, nil
	}
	// One class was asked for. An inner class may come back under its binary
	// name, so the only entry is the answer whatever its spelling.
	entry := result.Classes[0]
	for _, c := range result.Classes {
		if c.ClassName == className {
			entry = c
			break
		}
	}
	explored := &exploredClass{
		Source:   entry.Source,
		Language: entry.Language,
		Artifact: entry.Artifact,
	}
	if result.Meta.Project.Resolution == "project-pinned" {
		if resolved, ok := result.Meta.Resolved[className]; ok && resolved.Policy == "cache-wide" {
			explored.Unpinned = true
		}
	}
	return explored, nil
}

// parse reads a dependency source in the language the indexer reported.
func (r *MavenDependencyResolver) parse(className, language, source string) ([]parser.Class, error) {
	switch language {
	case "", "java":
		if r.parser == nil {
			return nil, fmt.Errorf("no Java parser")
		}
		return r.parser.ParseSource([]byte(source))
	}
	if p := registeredSourceParser(language); p != nil {
		return p(className, []byte(source))
	}
	return nil, fmt.Errorf("%s: no parser for %s sources", className, language)
}

// pickClass returns the declaration a class name denotes among the
// declarations of one source: the source of Outer also declares Outer.Inner,
// and a Kotlin file may declare several top-level classes.
func pickClass(classes []parser.Class, className string) *parser.Class {
	simple := className[strings.LastIndexAny(className, ".$")+1:]
	for i := range classes {
		if classes[i].Name == simple {
			return &classes[i]
		}
	}
	return nil
}

func (r *MavenDependencyResolver) DetectDependencies(sourceDir string) ([]collector.Dependency, error) {
	deps, err := detectDependencies(sourceDir)
	if err != nil {
		return nil, err
	}

	var result []collector.Dependency
	for _, d := range deps {
		result = append(result, collector.Dependency{
			Name:    d.ArtifactID,
			Version: d.Version,
		})
	}
	return result, nil
}

func (r *MavenDependencyResolver) ResolveType(typeName string) *collector.ResolvedType {
	class := r.ResolveClass(typeName)
	if class == nil {
		return nil
	}

	rt := &collector.ResolvedType{
		Name:               class.Name,
		TypeParameters:     class.TypeParameters,
		SuperClass:         class.SuperClass,
		SuperClassTypeArgs: class.SuperClassTypeArgs,
		IsInterface:        class.IsInterface,
		Interfaces:         class.Interfaces,
	}

	for _, f := range class.Fields {
		// Statics (including static final constants) are not serialized; final
		// instance fields are, and immutable DTOs (Lombok @Value) are all-final.
		if f.IsStatic {
			continue
		}
		required := true
		for _, ann := range f.Annotations {
			if ann.Name == "Nullable" || ann.Name == "Null" {
				required = false
			}
		}
		rt.Fields = append(rt.Fields, collector.ResolvedField{
			Name:     f.Name,
			Type:     f.Type,
			Required: required,
		})
	}

	return rt
}
