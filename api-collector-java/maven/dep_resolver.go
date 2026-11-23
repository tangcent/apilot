package maven

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

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
// through maven-indexer-cli.
//
// By default it asks `explore`, which pins each class to the version the
// project's dependency tree names. A CLI older than 1.2.0 has no explore, so
// UseGetClass switches to `get-class`, which reads the version the index
// holds. A class explore fails on is read with get-class too, and the failure
// is reported through Failed.
type MavenDependencyResolver struct {
	mu          sync.Mutex
	parser      *parser.Parser
	run         runner
	getClass    bool
	projectPath string
	cache       map[string]*parser.Class
	misses      map[string]bool
	// siblings maps a class read from the index to the classes nested in it, as
	// its source declares them. The index holds top-level classes only, so a
	// nested class is read from the source of the class that declares it.
	siblings map[string][]parser.Class
	// unpinned maps a class the indexer could not pin to the project's
	// dependency versions to the artifact it was read from.
	unpinned map[string]string
	// failed maps a class whose lookup failed to the reason.
	failed map[string]string
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
		siblings: make(map[string][]parser.Class),
		unpinned: make(map[string]string),
		failed:   make(map[string]string),
	}, nil
}

// SetRunner replaces how the resolver reaches the indexer CLI: run receives the
// arguments of one invocation and returns what the CLI printed. It lets a test
// script the indexer's answers, and count the lookups made, without the tool.
func (r *MavenDependencyResolver) SetRunner(run func(args ...string) ([]byte, error)) {
	r.run = run
}

// SetProjectPath names the project whose dependency tree pins class versions.
func (r *MavenDependencyResolver) SetProjectPath(projectPath string) {
	r.projectPath = projectPath
}

// UseGetClass makes the resolver read classes with `get-class`, for a CLI
// that has no explore. Versions then follow the index, not the project.
func (r *MavenDependencyResolver) UseGetClass() {
	r.getClass = true
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

// Failed lists the classes whose lookup failed, each as "class: reason": the
// CLI exited with an error, answered in a shape the resolver does not know,
// or returned a source no parser could read. A class missing from the index
// is not a failure. When get-class read a class explore had failed on, the
// reason says so.
func (r *MavenDependencyResolver) Failed() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.failed))
	for class, reason := range r.failed {
		out = append(out, class+": "+reason)
	}
	sort.Strings(out)
	return out
}

// ResolveClass returns the class a qualified name denotes. A class nested in
// another is named through it, `com.example.Page.Sort`, or by its binary name,
// `com.example.Page$Sort`.
func (r *MavenDependencyResolver) ResolveClass(className string) *parser.Class {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.resolve(canonicalClassName(className))
}

// resolve answers className from the cache, then from the index.
func (r *MavenDependencyResolver) resolve(className string) *parser.Class {
	if cached, ok := r.cache[className]; ok {
		return cached
	}

	if r.misses[className] {
		return nil
	}

	// The index holds top-level classes only: none of its names is that of a
	// class nested in another, so a nested class is taken from the source of the
	// class that declares it.
	if outer, nested := outermostClass(className); nested && r.resolve(outer) != nil {
		if class := r.nestedClass(outer, className); class != nil {
			r.cache[className] = class
			return class
		}
		r.misses[className] = true
		return nil
	}
	// A name whose outermost class is not in the index is asked for as written: a
	// package that starts with a capital letter reads as a class.

	class := r.resolveFromCLI(className)
	if class != nil {
		r.cache[className] = class
		return class
	}

	r.misses[className] = true
	return nil
}

// fetchedClass is the source of one class as the CLI returned it.
type fetchedClass struct {
	Source   string
	Language string
	Artifact string
	// Unpinned is set when the project's dependency tree was known but the
	// indexer still read the class from a version of its own choosing.
	Unpinned bool
}

type exploreJSON struct {
	// Classes is a pointer so that an answer without the key is told apart
	// from an answer that found nothing.
	Classes *[]struct {
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
	var fetched *fetchedClass
	var err error
	if r.getClass {
		fetched, err = r.fetchWithGetClass(className)
		if err != nil {
			r.failed[className] = err.Error()
			return nil
		}
	} else {
		fetched, err = r.explore(className)
		if err != nil {
			// The explore contract is missing or broken: read the class the
			// way a CLI without explore does, and say so.
			reason := err.Error()
			fetched, err = r.fetchWithGetClass(className)
			switch {
			case err != nil:
				reason += "; get-class: " + err.Error()
			case fetched != nil:
				reason += "; read with get-class instead, not pinned to the project's version"
			}
			r.failed[className] = reason
		}
	}
	if fetched == nil || fetched.Source == "" {
		return nil
	}

	classes, err := r.parse(className, fetched.Language, fetched.Source)
	if err != nil {
		r.failed[className] = fmt.Sprintf("reading the %s source: %v", languageName(fetched.Language), err)
		return nil
	}

	class := pickClass(classes, className)
	if class != nil {
		r.keepSiblings(className, classes, class)
	}
	if class != nil && fetched.Unpinned {
		r.unpinned[className] = fetched.Artifact
	}
	return class
}

// keepSiblings remembers the classes nested in the class that was asked for, so
// that a class nested in it is answered without another call. The other
// top-level classes of the source are not kept: each is asked for on its own, as
// it always was, and may be read from another version of the library.
func (r *MavenDependencyResolver) keepSiblings(className string, classes []parser.Class, picked *parser.Class) {
	var siblings []parser.Class
	for i := range classes {
		c := classes[i]
		if &classes[i] == picked || c.Package != picked.Package {
			continue
		}
		if outermost, _, _ := strings.Cut(c.Outer, "."); outermost == picked.Name && picked.Outer == "" {
			siblings = append(siblings, c)
		}
	}
	if len(siblings) == 0 {
		return
	}
	if r.siblings == nil {
		r.siblings = make(map[string][]parser.Class)
	}
	r.siblings[className] = siblings
}

// nestedClass returns the class named name that the source of the class outer
// declares.
func (r *MavenDependencyResolver) nestedClass(outer, name string) *parser.Class {
	for _, sibling := range r.siblings[outer] {
		if sibling.QualifiedName() == name {
			class := sibling
			return &class
		}
	}
	return nil
}

// Siblings returns the classes nested in the class className was read from, or
// in the class that declares it when className is a nested class, and that
// declaring class too. The type resolver registers them with the class, so that
// a name that denotes one of them is not asked of the index.
func (r *MavenDependencyResolver) Siblings(className string) []parser.Class {
	r.mu.Lock()
	defer r.mu.Unlock()

	className = canonicalClassName(className)
	if siblings, ok := r.siblings[className]; ok {
		return append([]parser.Class(nil), siblings...)
	}
	outer, nested := outermostClass(className)
	if !nested {
		return nil
	}
	var siblings []parser.Class
	if declaring := r.cache[outer]; declaring != nil {
		siblings = append(siblings, *declaring)
	}
	return append(siblings, r.siblings[outer]...)
}

func languageName(language string) string {
	if language == "" {
		return "java"
	}
	return language
}

// explore fetches the source of one class, pinned to the project's dependency
// tree when a project path is set. A class the index does not hold is no
// error: the answer is nil.
func (r *MavenDependencyResolver) explore(className string) (*fetchedClass, error) {
	args := []string{"--json", "explore", className, "--include", "source", "--max-lines", maxSourceLines}
	if r.projectPath != "" {
		args = append(args, "--project-path", r.projectPath)
	}
	output, err := r.run(args...)
	if err != nil {
		return nil, fmt.Errorf("explore failed: %w", err)
	}

	var result exploreJSON
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("explore answered in an unexpected format: %w", err)
	}
	if result.Classes == nil {
		return nil, fmt.Errorf("explore answered without a classes list")
	}
	classes := *result.Classes
	if len(classes) == 0 {
		return nil, nil
	}
	// One class was asked for. An inner class may come back under its binary
	// name, so the only entry is the answer whatever its spelling.
	entry := classes[0]
	for _, c := range classes {
		if c.ClassName == className {
			entry = c
			break
		}
	}
	fetched := &fetchedClass{
		Source:   entry.Source,
		Language: entry.Language,
		Artifact: entry.Artifact,
	}
	if result.Meta.Project.Resolution == "project-pinned" {
		if resolved, ok := result.Meta.Resolved[className]; ok && resolved.Policy == "cache-wide" {
			fetched.Unpinned = true
		}
	}
	return fetched, nil
}

// fetchWithGetClass fetches the source of one class with `get-class`, from
// the version the index holds. A class the index does not hold is no error:
// the answer is nil.
func (r *MavenDependencyResolver) fetchWithGetClass(className string) (*fetchedClass, error) {
	output, err := r.run("get-class", className, "--json", "--type", "source")
	if err != nil {
		return nil, fmt.Errorf("get-class failed: %w", err)
	}

	var raw string
	if err := json.Unmarshal(output, &raw); err != nil {
		raw = string(output)
	}
	raw = strings.TrimSpace(raw)
	// A miss is a sentence, such as "Class '…' not found in the index." or
	// "Artifact … not found in index.".
	if raw == "" || (strings.HasPrefix(raw, "Class ") || strings.HasPrefix(raw, "Artifact ")) && strings.Contains(raw, "not found") {
		return nil, nil
	}
	source, language := extractSourceFromMarkdown(raw)
	return &fetchedClass{Source: source, Language: language}, nil
}

// extractSourceFromMarkdown returns the first fenced code block of a
// get-class answer, and the language its fence names: "kotlin" for a class of
// a Kotlin sources JAR, "java" for Java sources and decompiled classes. An
// answer without a fence is taken as Java source.
func extractSourceFromMarkdown(raw string) (string, string) {
	for _, language := range []string{"java", "kotlin"} {
		fence := "```" + language + "\n"
		start := strings.Index(raw, fence)
		if start == -1 {
			continue
		}
		remaining := raw[start+len(fence):]
		if end := strings.Index(remaining, "```"); end != -1 {
			remaining = remaining[:end]
		}
		return strings.TrimSuffix(remaining, "\n"), language
	}
	return raw, "java"
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
	return nil, fmt.Errorf("no parser for %s sources", language)
}

// pickClass returns the declaration a class name denotes among the
// declarations of one source: the source of Outer also declares Outer.Inner,
// and a Kotlin file may declare several top-level classes. The declaration
// whose qualified name is the one asked for answers; failing that, one that
// carries the simple name, a top-level one first; and when no declaration
// carries the simple name, the first one answers, as it always has.
func pickClass(classes []parser.Class, className string) *parser.Class {
	className = canonicalClassName(className)
	for i := range classes {
		if classes[i].QualifiedName() == className {
			return &classes[i]
		}
	}
	simple := className[strings.LastIndexAny(className, ".$")+1:]
	for i := range classes {
		if classes[i].Outer == "" && classes[i].Name == simple {
			return &classes[i]
		}
	}
	for i := range classes {
		if classes[i].Name == simple {
			return &classes[i]
		}
	}
	if len(classes) > 0 {
		return &classes[0]
	}
	return nil
}

// canonicalClassName spells a binary class name, `com.example.Page$Sort`, with
// the dot of source, `com.example.Page.Sort`: a decompiler or a build tool may
// name a nested class either way. A `$` that is not a separator between two
// names, as in `Outer$1`, is left alone.
func canonicalClassName(name string) string {
	if !strings.Contains(name, "$") {
		return name
	}
	parts := strings.Split(name, "$")
	for _, part := range parts {
		if part == "" || strings.HasSuffix(part, ".") {
			return name
		}
		if first, _ := utf8.DecodeRuneInString(part); !unicode.IsLetter(first) && first != '_' {
			return name
		}
	}
	return strings.Join(parts, ".")
}

// outermostClass returns the top-level class a class name is nested in, and
// whether the name is that of a nested class at all: the outermost class of
// com.example.Page.Sort is com.example.Page. A package name starts with a
// lower-case letter and a class name does not, which is as far as the spelling
// goes.
func outermostClass(name string) (string, bool) {
	segments := strings.Split(name, ".")
	for i, segment := range segments {
		if segment == "" {
			return name, false
		}
		if first, _ := utf8.DecodeRuneInString(segment); unicode.IsLower(first) {
			continue
		}
		if i == len(segments)-1 {
			return name, false
		}
		return strings.Join(segments[:i+1], "."), true
	}
	return name, false
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
