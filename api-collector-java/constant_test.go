package javacollector

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
	"github.com/tangcent/apilot/api-collector-java/springmvc"
)

// constantNameExpectation is the export of testdata/constant: each parameter
// of the handler by the name it is bound by, and where it is bound from. Every
// binding annotation but the last names a String constant instead of spelling
// the string, and Spring binds the string the constant holds: a constant of
// the controller itself, of an interface, one defined by another constant, of
// a class named by its simple and by its qualified name, and one imported
// with import static, single and on demand. A constant that cannot be found
// leaves the parameter under its own name (fallback), with a warning; a
// literal is read as before.
//
// The Kotlin collector asserts the same table for the same handler written in
// Kotlin (api-collector-kotlin/constant_test.go), where the constants live in
// a companion object, an object, an interface's companion object and
// top-level const vals: keep the two in sync.
var constantNameExpectation = map[string]string{
	"X-Trace-Id":      "header",
	"X-Api-Version":   "header",
	"Accept-Language": "header",
	"page_size":       "query",
	"orderNo":         "path",
	"SESSION":         "cookie",
	"q":               "query",
	"tenant_id":       "query",
	"region":          "query",
	"fallback":        "header",
	"literal":         "query",
}

func TestBindingNamesFromConstants(t *testing.T) {
	abs, err := filepath.Abs("testdata/constant")
	if err != nil {
		t.Fatal(err)
	}
	c := New().(*JavaCollector)
	endpoints, err := c.Collect(collector.CollectContext{SourceDir: abs, NoDeps: true})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(endpoints) != 1 {
		t.Fatalf("endpoints = %+v, want the find endpoint", endpoints)
	}
	got := map[string]string{}
	for _, p := range endpoints[0].Parameters {
		got[p.Name] = p.In
	}
	for _, h := range endpoints[0].Headers {
		got[h.Name] = "header"
	}
	if !reflect.DeepEqual(got, constantNameExpectation) {
		t.Errorf("binding names:\n got  %v\n want %v", got, constantNameExpectation)
	}
	checkUnresolvedNameWarning(t, c.Warnings())
}

// checkUnresolvedNameWarning asserts that the only unresolved constant of
// testdata/constant is reported, naming the parameter and the reference.
func checkUnresolvedNameWarning(t *testing.T, warnings []string) {
	t.Helper()
	var unresolved []string
	for _, w := range warnings {
		if strings.Contains(w, "could not be resolved") {
			unresolved = append(unresolved, w)
		}
	}
	if len(unresolved) != 1 || !strings.Contains(unresolved[0], "ConstantNameController.find: parameter fallback") ||
		!strings.Contains(unresolved[0], "Missing.NAME") {
		t.Errorf("warnings = %q, want one for the fallback parameter naming Missing.NAME", warnings)
	}
}

// constantLibrarySources are the constants classes of a library that ships no
// sources JAR, as the indexer returns them: decompiled by CFR from Kotlin. A
// const val of an object is a static field of its class, and one of an
// interface's companion object a static field of the interface.
var constantLibrarySources = map[string]string{
	"com.example.lib.LibHeaders": `/*
 * Decompiled with CFR 0.152.
 *
 * Could not load the following classes:
 *  kotlin.Metadata
 *  org.jetbrains.annotations.NotNull
 */
package com.example.lib;

import kotlin.Metadata;
import org.jetbrains.annotations.NotNull;

@Metadata(mv={1, 9, 0}, k=1, xi=48, d1={"\u0000\u0014\n\u0002"}, d2={"Lcom/example/lib/LibHeaders;", "", "()V", "CHANNEL", "", "lib"})
public final class LibHeaders {
    @NotNull
    public static final LibHeaders INSTANCE = new LibHeaders();
    @NotNull
    public static final String CHANNEL = "X-Channel";

    private LibHeaders() {
    }
}
`,
	"com.example.lib.CompiledHeaders": compiledHeadersSource,
}

const compiledHeadersSource = `/*
 * Decompiled with CFR 0.152.
 *
 * Could not load the following classes:
 *  kotlin.Metadata
 *  org.jetbrains.annotations.NotNull
 */
package com.example.lib;

import kotlin.Metadata;
import org.jetbrains.annotations.NotNull;

@Metadata(mv={1, 9, 0}, k=1, xi=48, d1={"\u0000\u000e\n\u0002"}, d2={"Lcom/example/lib/CompiledHeaders;", "", "Companion", "lib"})
public interface CompiledHeaders {
    @NotNull
    public static final Companion Companion = Companion.$$INSTANCE;
    @NotNull
    public static final String DEVICE = "X-Device";

    @Metadata(mv={1, 9, 0}, k=1, xi=48, d1={"\u0000\u0014\n\u0002"}, d2={"Lcom/example/lib/CompiledHeaders$Companion;", "", "()V", "DEVICE", "", "lib"})
    public static final class Companion {
        static final /* synthetic */ Companion $$INSTANCE;
        @NotNull
        public static final String DEVICE = "X-Device";

        private Companion() {
        }

        static {
            $$INSTANCE = new Companion();
        }
    }
}
`

// constantDependencyExpectation is the export of testdata/constant-dependency,
// whose binding names are constants of the library. The Kotlin collector
// asserts the same table (api-collector-kotlin/constant_test.go).
var constantDependencyExpectation = map[string]string{
	"X-Channel": "header",
	"X-Device":  "header",
}

// constantLibrary answers dependency lookups from constantLibrarySources.
type constantLibrary struct {
	t       *testing.T
	parser  *parser.Parser
	queried []string
}

func (l *constantLibrary) ResolveClass(className string) *parser.Class {
	l.queried = append(l.queried, className)
	source, ok := constantLibrarySources[className]
	if !ok {
		return nil
	}
	classes, err := l.parser.ParseSource([]byte(source))
	if err != nil {
		l.t.Fatalf("parsing %s: %v", className, err)
	}
	simple := className[strings.LastIndex(className, ".")+1:]
	for i := range classes {
		if classes[i].Name == simple {
			return &classes[i]
		}
	}
	return nil
}

func TestBindingNamesFromConstantsOfADependency(t *testing.T) {
	p, err := parser.NewParser(parser.ParserOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	abs, err := filepath.Abs("testdata/constant-dependency")
	if err != nil {
		t.Fatal(err)
	}
	results, err := p.ParseDirectory(abs)
	if err != nil {
		t.Fatal(err)
	}

	library := &constantLibrary{t: t, parser: p}
	sm := springmvc.NewParser()
	sm.SetDependencyResolver(library)
	controllers := sm.ExtractControllers(results)
	if len(controllers) != 1 || len(controllers[0].Endpoints) != 1 {
		t.Fatalf("controllers = %+v, want the list endpoint", controllers)
	}
	got := map[string]string{}
	for _, param := range controllers[0].Endpoints[0].Parameters {
		got[param.Name] = param.ParamType
	}
	if !reflect.DeepEqual(got, constantDependencyExpectation) {
		t.Errorf("binding names:\n got  %v\n want %v", got, constantDependencyExpectation)
	}
	if warnings := sm.Warnings(); len(warnings) != 0 {
		t.Errorf("warnings = %q", warnings)
	}
	for _, want := range []string{"com.example.lib.LibHeaders", "com.example.lib.CompiledHeaders"} {
		found := false
		for _, q := range library.queried {
			found = found || q == want
		}
		if !found {
			t.Errorf("the library was not asked for %s; asked for %v", want, library.queried)
		}
	}
}
