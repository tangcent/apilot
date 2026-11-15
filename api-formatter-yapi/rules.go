package yapi

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// RulesFileName is the rules file of the EasyYapi IntelliJ plugin. A project
// that already documents its endpoints with EasyYapi keeps its type conversions
// and mock rules there, so a YApi export from APilot reads the same file.
const RulesFileName = ".easy.api.config"

// Rules are the parts of an EasyYapi rules file that change what a YApi export
// contains:
//
//   - json.rule.convert[java.lang.Long]=java.lang.String exports a type as
//     another one; it only applies between the primitive types the model knows
//     (string, int, long, float, double, boolean);
//   - mock.[path|type]=value attaches a mock to every schema node whose JSON
//     path and JSON type match, the first matching rule winning;
//   - properties.additional=<file or URL> includes another rules file at that
//     point, and ${name} in a value is replaced by the value of name.
//
// Every other key is ignored: it belongs to EasyYapi's own rule engine.
type Rules struct {
	convert map[string]string
	mocks   []mockRule
	// Sources lists the files and URLs that were read, in reading order.
	Sources []string
	// Warnings names rules that were read but cannot be applied.
	Warnings []string
}

type mockRule struct {
	path  func(string) bool
	typ   func(string) bool
	value string
}

// convertType returns the model type a field of type t is exported as.
func (r *Rules) convertType(t string) string {
	if r != nil {
		if to, ok := r.convert[t]; ok {
			return to
		}
	}
	return t
}

// mockFor returns the mock of the first rule matching a schema node.
func (r *Rules) mockFor(path, jsonType string) (string, bool) {
	if r == nil {
		return "", false
	}
	for _, rule := range r.mocks {
		if rule.path(path) && rule.typ(jsonType) {
			return rule.value, true
		}
	}
	return "", false
}

// Fetcher reads a rules file that is given by URL.
type Fetcher func(url string) ([]byte, error)

// HTTPFetcher fetches a URL with the given client.
func HTTPFetcher(client *http.Client) Fetcher {
	return func(url string) ([]byte, error) {
		resp, err := client.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		return io.ReadAll(resp.Body)
	}
}

// LoadRules reads the rules that apply to a project: the .easy.api.config in
// sourceDir and in each of its direct subdirectories (a multi-module project
// keeps it in a module), then the files named in extra. With nothing to read,
// it returns empty rules.
func LoadRules(sourceDir string, extra []string, fetch Fetcher) (*Rules, error) {
	l := &loader{fetch: fetch, seen: map[string]bool{}}
	for _, path := range append(discoverRuleFiles(sourceDir), extra...) {
		if err := l.load(path, 0); err != nil {
			return nil, err
		}
	}
	return l.rules(), nil
}

func discoverRuleFiles(sourceDir string) []string {
	if sourceDir == "" {
		return nil
	}
	var out []string
	add := func(dir string) {
		path := filepath.Join(dir, RulesFileName)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			out = append(out, path)
		}
	}
	add(sourceDir)
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		switch name := e.Name(); {
		case strings.HasPrefix(name, "."), name == "build", name == "target", name == "out", name == "node_modules":
			continue
		}
		add(filepath.Join(sourceDir, e.Name()))
	}
	return out
}

type entry struct {
	key, value string
}

type loader struct {
	fetch   Fetcher
	seen    map[string]bool
	entries []entry
	sources []string
}

const maxIncludeDepth = 8

func isURL(location string) bool {
	return strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://")
}

func (l *loader) load(location string, depth int) error {
	if depth > maxIncludeDepth {
		return fmt.Errorf("rules file %s: includes nest more than %d levels", location, maxIncludeDepth)
	}
	if l.seen[location] {
		return nil
	}
	l.seen[location] = true

	var content []byte
	var err error
	if isURL(location) {
		if l.fetch == nil {
			return fmt.Errorf("rules file %s: no fetcher for a URL", location)
		}
		content, err = l.fetch(location)
	} else {
		content, err = os.ReadFile(location)
	}
	if err != nil {
		return fmt.Errorf("reading rules from %s: %w", location, err)
	}
	l.sources = append(l.sources, location)

	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), string(rune(0xFEFF))))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.Index(line, "=")
		if eq <= 0 {
			continue
		}
		key, value := strings.TrimSpace(line[:eq]), strings.TrimSpace(line[eq+1:])
		if key == "properties.additional" {
			include := value
			if !isURL(include) && !filepath.IsAbs(include) && !isURL(location) {
				include = filepath.Join(filepath.Dir(location), include)
			}
			if err := l.load(include, depth+1); err != nil {
				return err
			}
			continue
		}
		l.entries = append(l.entries, entry{key: key, value: value})
	}
	return scanner.Err()
}

var placeholder = regexp.MustCompile(`\$\{([^}]+)\}`)

func (l *loader) rules() *Rules {
	props := map[string]string{}
	for _, e := range l.entries {
		props[e.key] = e.value
	}
	resolve := func(value string) string {
		for i := 0; i < 10 && strings.Contains(value, "${"); i++ {
			value = placeholder.ReplaceAllStringFunc(value, func(m string) string {
				if v, ok := props[m[2:len(m)-1]]; ok {
					return v
				}
				return m
			})
		}
		return value
	}

	r := &Rules{convert: map[string]string{}, Sources: l.sources}
	for _, e := range l.entries {
		switch {
		case strings.HasPrefix(e.key, "json.rule.convert[") && strings.HasSuffix(e.key, "]"):
			from := strings.TrimSuffix(strings.TrimPrefix(e.key, "json.rule.convert["), "]")
			to := resolve(e.value)
			mf, okFrom := javaPrimitive[strings.TrimSpace(from)]
			mt, okTo := javaPrimitive[strings.TrimSpace(to)]
			if !okFrom || !okTo {
				r.Warnings = append(r.Warnings, fmt.Sprintf(
					"%s=%s: only conversions between primitive types and String are applied", e.key, e.value))
				continue
			}
			r.convert[mf] = mt
		case e.key == "mock.rule":
			// EasyYapi 3.x's mock.rule=<pattern>=<mock> fills parameter
			// examples, not body schemas; it is not the rule read here.
			continue
		case strings.HasPrefix(e.key, "mock."):
			pattern := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(e.key, "mock."), "["), "]")
			pathPattern, typePattern := pattern, "*"
			if bar := strings.Index(pattern, "|"); bar >= 0 {
				pathPattern, typePattern = pattern[:bar], pattern[bar+1:]
			}
			r.mocks = append(r.mocks, mockRule{
				path:  globMatcher(pathPattern),
				typ:   globMatcher(typePattern),
				value: resolve(e.value),
			})
		}
	}
	return r
}

// javaPrimitive maps the Java types a conversion rule may name to the model's
// types. The model does not tell a boxed type from its primitive, so a rule on
// java.lang.Long applies to long fields as well.
var javaPrimitive = map[string]string{
	"java.lang.String": "string", "String": "string", "java.lang.CharSequence": "string",
	"java.lang.Long": "long", "Long": "long", "long": "long", "java.math.BigInteger": "long",
	"java.lang.Integer": "int", "Integer": "int", "int": "int",
	"java.lang.Short": "int", "short": "int", "java.lang.Byte": "int", "byte": "int",
	"java.lang.Double": "double", "Double": "double", "double": "double", "java.math.BigDecimal": "double",
	"java.lang.Float": "float", "Float": "float", "float": "float",
	"java.lang.Boolean": "boolean", "Boolean": "boolean", "boolean": "boolean",
}

// globMatcher compiles a mock-rule pattern the way EasyYapi reads it: blank or
// "*" matches anything, a pattern without "*" matches itself only, "*."
// matches any leading path ending at a dot (or nothing), and "*" matches any
// run of characters.
func globMatcher(pattern string) func(string) bool {
	p := strings.TrimSpace(pattern)
	if p == "" || p == "*" {
		return func(string) bool { return true }
	}
	if !strings.Contains(p, "*") {
		return func(v string) bool { return v == p }
	}
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(p); {
		switch {
		case strings.HasPrefix(p[i:], "*."):
			b.WriteString(`(?:.*\.)?`)
			i += 2
		case p[i] == '*':
			b.WriteString(".*")
			i++
		default:
			j := i
			for j < len(p) && p[j] != '*' {
				j++
			}
			b.WriteString(regexp.QuoteMeta(p[i:j]))
			i = j
		}
	}
	b.WriteString("$")
	re := regexp.MustCompile(b.String())
	return re.MatchString
}
