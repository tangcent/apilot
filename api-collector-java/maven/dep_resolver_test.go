package maven

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

func TestMavenDependencyResolver_New(t *testing.T) {
	// Construction does not require the CLI: availability gating lives in the
	// collectors, which report a tool-missing outcome. Without the CLI the
	// resolver simply records misses.
	resolver, err := NewMavenDependencyResolver()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if resolver == nil {
		t.Fatal("Expected non-nil resolver")
	}
	defer resolver.Close()

	if !isCLIAvailable() {
		if resolver.ResolveClass("java.lang.String") != nil {
			t.Error("Expected nil class when CLI unavailable")
		}
	}
}

func TestMavenDependencyResolver_ResolveTypeFieldFiltering(t *testing.T) {
	// Dependency classes go through the same static/final rules as workspace
	// classes: statics are dropped, final instance fields (immutable DTOs,
	// Lombok @Value) are kept.
	r := &MavenDependencyResolver{cache: map[string]*parser.Class{
		"com.example.lib.ImmutableProduct": {
			Name: "com.example.lib.ImmutableProduct",
			Fields: []parser.Field{
				{Name: "TYPE", Type: "String", IsStatic: true, IsFinal: true, HasInitializer: true},
				{Name: "sku", Type: "String", IsFinal: true},
				{Name: "price", Type: "long", IsFinal: true},
				{Name: "memo", Type: "String"},
			},
		},
	}}

	rt := r.ResolveType("com.example.lib.ImmutableProduct")
	if rt == nil {
		t.Fatal("Expected resolved type")
	}
	if len(rt.Fields) != 3 {
		t.Fatalf("Expected 3 fields, got %d: %#v", len(rt.Fields), rt.Fields)
	}

	names := make(map[string]bool, len(rt.Fields))
	for _, f := range rt.Fields {
		names[f.Name] = true
	}
	if names["TYPE"] {
		t.Error("Expected static final constant 'TYPE' to stay excluded")
	}
	for _, want := range []string{"sku", "price", "memo"} {
		if !names[want] {
			t.Errorf("Expected field '%s' to be kept", want)
		}
	}
}

const pageSource = `package com.example.lib;

import java.util.List;

public class Page<T> {
    private int page;
    private List<T> rows;

    public static class Sort {
        private String field;
    }
}`

// exploreAnswer builds the JSON `maven-indexer-cli --json explore` prints for
// one class.
func exploreAnswer(className, artifact, language, source, policy, resolution string) string {
	answer := map[string]any{
		"classes": []map[string]any{{
			"className": className, "artifact": artifact, "language": language, "source": source,
		}},
		"_meta": map[string]any{
			"resolved": map[string]any{className: map[string]any{"artifact": artifact, "policy": policy}},
			"project":  map[string]any{"kind": "maven", "tree": "resolved", "resolution": resolution},
		},
	}
	out, _ := json.Marshal(answer)
	return string(out)
}

// newTestResolver returns a resolver whose CLI answers with answer and records
// the arguments of each call.
func newTestResolver(t *testing.T, answer string, calls *[][]string) *MavenDependencyResolver {
	t.Helper()
	r, err := NewMavenDependencyResolver()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(r.Close)
	r.run = func(args ...string) ([]byte, error) {
		*calls = append(*calls, args)
		return []byte(answer), nil
	}
	return r
}

func fieldNames(c *parser.Class) []string {
	var names []string
	for _, f := range c.Fields {
		names = append(names, f.Name)
	}
	return names
}

func TestResolveClass_ExploresTheClassPinnedToTheProject(t *testing.T) {
	var calls [][]string
	r := newTestResolver(t, exploreAnswer("com.example.lib.Page", "com.example:lib:2.0.0", "java", pageSource, "project-pinned", "project-pinned"), &calls)
	r.SetProjectPath("/work/service")

	class := r.ResolveClass("com.example.lib.Page")
	if class == nil || class.Name != "Page" || !reflect.DeepEqual(fieldNames(class), []string{"page", "rows"}) {
		t.Fatalf("class = %+v", class)
	}
	want := []string{"--json", "explore", "com.example.lib.Page", "--include", "source", "--max-lines", maxSourceLines, "--project-path", "/work/service"}
	if len(calls) != 1 || !reflect.DeepEqual(calls[0], want) {
		t.Fatalf("calls = %q, want one call %q", calls, want)
	}
	if got := r.Unpinned(); len(got) != 0 {
		t.Errorf("a pinned class is not reported: %v", got)
	}

	// The answer is cached.
	r.ResolveClass("com.example.lib.Page")
	if len(calls) != 1 {
		t.Errorf("calls = %d, want the cached class", len(calls))
	}
}

func TestResolveClass_PicksTheNamedDeclarationOfTheSource(t *testing.T) {
	var calls [][]string
	r := newTestResolver(t, exploreAnswer("com.example.lib.Page$Sort", "com.example:lib:2.0.0", "java", pageSource, "project-pinned", "project-pinned"), &calls)

	class := r.ResolveClass("com.example.lib.Page.Sort")
	if class == nil || class.Name != "Sort" {
		t.Fatalf("class = %+v, want the nested Sort", class)
	}
	if pickClass([]parser.Class{{Name: "Page"}}, "com.example.lib.Other") != nil {
		t.Error("a source that does not declare the class must not answer for it")
	}
}

func TestResolveClass_ReportsAClassTheIndexerCouldNotPin(t *testing.T) {
	var calls [][]string
	r := newTestResolver(t, exploreAnswer("com.example.lib.Page", "com.example:lib:1.0.0", "java", pageSource, "cache-wide", "project-pinned"), &calls)

	if r.ResolveClass("com.example.lib.Page") == nil {
		t.Fatal("the class is still resolved")
	}
	want := []string{"com.example.lib.Page (com.example:lib:1.0.0)"}
	if got := r.Unpinned(); !reflect.DeepEqual(got, want) {
		t.Errorf("Unpinned = %v, want %v", got, want)
	}
}

func TestResolveClass_WithoutAProjectTreeNothingIsListedAsUnpinned(t *testing.T) {
	// Without a tree every class is read cache-wide; the dependency status
	// says so once instead of listing every class.
	var calls [][]string
	r := newTestResolver(t, exploreAnswer("com.example.lib.Page", "com.example:lib:1.0.0", "java", pageSource, "cache-wide", "cache-wide"), &calls)

	if r.ResolveClass("com.example.lib.Page") == nil {
		t.Fatal("the class is still resolved")
	}
	if got := r.Unpinned(); len(got) != 0 {
		t.Errorf("Unpinned = %v, want none", got)
	}
}

func TestResolveClass_ASourceGoesToTheParserOfItsLanguage(t *testing.T) {
	var gotName, gotSource string
	RegisterSourceParser("kotlin", func(name string, source []byte) ([]parser.Class, error) {
		gotName, gotSource = name, string(source)
		return []parser.Class{{Name: "Wrapper", Package: "com.example.lib"}}, nil
	})
	t.Cleanup(func() { RegisterSourceParser("kotlin", nil) })

	var calls [][]string
	kotlin := "package com.example.lib\n\nclass Wrapper<T>(var data: T? = null)"
	r := newTestResolver(t, exploreAnswer("com.example.lib.Wrapper", "com.example:lib:2.0.0", "kotlin", kotlin, "project-pinned", "project-pinned"), &calls)

	if class := r.ResolveClass("com.example.lib.Wrapper"); class == nil || class.Name != "Wrapper" {
		t.Fatalf("class = %+v", class)
	}
	if gotName != "com.example.lib.Wrapper" || gotSource != kotlin {
		t.Errorf("parser got (%q, %q)", gotName, gotSource)
	}
}

func TestResolveClass_ASourceWithoutAParserIsAMiss(t *testing.T) {
	var calls [][]string
	r := newTestResolver(t, exploreAnswer("com.example.lib.Box", "com.example:lib:2.0.0", "scala", "case class Box(v: Int)", "project-pinned", "project-pinned"), &calls)

	if class := r.ResolveClass("com.example.lib.Box"); class != nil {
		t.Fatalf("class = %+v, want a miss: the Java grammar cannot read Scala", class)
	}
}

func TestResolveClass_AFailedCallIsAMiss(t *testing.T) {
	r, err := NewMavenDependencyResolver()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	calls := 0
	r.run = func(args ...string) ([]byte, error) {
		calls++
		return nil, errors.New("exit status 1")
	}

	if r.ResolveClass("com.example.lib.Gone") != nil {
		t.Fatal("want a miss")
	}
	r.ResolveClass("com.example.lib.Gone")
	if calls != 1 {
		t.Errorf("calls = %d, want the miss remembered", calls)
	}
}

func TestLoadProjectContext(t *testing.T) {
	context := func(tree string, declared, resolved int, resolveError string) string {
		answer := map[string]any{"kind": "maven", "tree": tree, "declared": make([]map[string]string, declared), "resolveError": resolveError}
		if tree == "resolved" {
			answer["resolved"] = map[string]any{"coordinates": make([]map[string]string, resolved)}
		}
		out, _ := json.Marshal(answer)
		return string(out)
	}

	for _, tc := range []struct {
		name    string
		answers []string
		calls   int
		want    ProjectContext
	}{
		{"a cold project is asked again once its tree is resolved",
			[]string{context("declared", 0, 0, ""), context("resolved", 0, 344, "")},
			2, ProjectContext{Kind: "maven", Tree: "resolved", Resolved: 344}},
		{"a cached tree answers at once",
			[]string{context("resolved", 1, 344, "")},
			1, ProjectContext{Kind: "maven", Tree: "resolved", Declared: 1, Resolved: 344}},
		{"a failed resolution is reported, not retried",
			[]string{context("declared", 2, 0, "mvn exited with code 1\nmore")},
			1, ProjectContext{Kind: "maven", Tree: "declared", Declared: 2, ResolveError: "mvn exited with code 1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls [][]string
			run := func(args ...string) ([]byte, error) {
				calls = append(calls, args)
				return []byte(tc.answers[len(calls)-1]), nil
			}
			got, err := loadProjectContext(run, "/work/service")
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("context = %+v, want %+v", got, tc.want)
			}
			if len(calls) != tc.calls {
				t.Errorf("calls = %d, want %d", len(calls), tc.calls)
			}
			if !reflect.DeepEqual(calls[0], []string{"--json", "project-context", "/work/service"}) {
				t.Errorf("args = %q", calls[0])
			}
		})
	}

	t.Run("a CLI without the command is an error", func(t *testing.T) {
		_, err := loadProjectContext(func(args ...string) ([]byte, error) {
			return nil, errors.New("exit status 1: error: unknown command 'project-context'")
		}, "/work/service")
		if err == nil || !strings.Contains(err.Error(), "unknown command") {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestDeclaresModules(t *testing.T) {
	write := func(t *testing.T, name, content string) string {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	if !DeclaresModules(write(t, "pom.xml", "<project><modules><module>api</module></modules></project>")) {
		t.Error("a pom with modules aggregates them")
	}
	if DeclaresModules(write(t, "pom.xml", "<project><artifactId>demo</artifactId></project>")) {
		t.Error("a pom without modules does not")
	}
	if !DeclaresModules(write(t, "settings.gradle", "include 'api'")) {
		t.Error("a Gradle settings file aggregates projects")
	}
}
