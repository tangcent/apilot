package kotlincollector

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	javaparser "github.com/tangcent/apilot/api-collector-java/parser"
	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
	"github.com/tangcent/apilot/api-collector-kotlin/resolver"
	"github.com/tangcent/apilot/api-collector-kotlin/springmvc"
)

// constantNameExpectation is the export of testdata/constant: each parameter
// of the handler by the name it is bound by, and where it is bound from. Every
// binding annotation but the last names a String constant instead of spelling
// the string, and Spring binds the string the constant holds: a const val of
// the controller's companion object, of an interface's companion object, one
// defined by a const val of an object, of an object named by its simple and by
// its qualified name, and top-level const vals, one imported by name and one
// of the controller's own package. A constant that cannot be found leaves the
// parameter under its own name (fallback), with a warning; a literal is read
// as before.
//
// The Java collector asserts the same table for the same handler written in
// Java (api-collector-java/constant_test.go): keep the two in sync.
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
	endpoints, c := collectDir(t, "testdata/constant")
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

	var unresolved []string
	for _, w := range c.Warnings() {
		if strings.Contains(w, "could not be resolved") {
			unresolved = append(unresolved, w)
		}
	}
	if len(unresolved) != 1 || !strings.Contains(unresolved[0], "ConstantNameController.find: parameter fallback") ||
		!strings.Contains(unresolved[0], "Missing.NAME") {
		t.Errorf("warnings = %q, want one for the fallback parameter naming Missing.NAME", c.Warnings())
	}
}

// constantLibrarySources are the constants classes of a library: one from its
// sources JAR, Kotlin as written, and one decompiled by CFR from Kotlin for a
// library that ships no sources, where a const val of an interface's
// companion object is a static field of the interface.
var constantLibrarySources = map[string]struct{ language, source string }{
	"com.example.lib.LibHeaders": {"kotlin", `package com.example.lib

object LibHeaders {
    const val CHANNEL = "X-Channel"
}
`},
	"com.example.lib.CompiledHeaders": {"java", `/*
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
`},
}

// constantDependencyExpectation is the export of testdata/constant-dependency,
// whose binding names are constants of the library. The Java collector
// asserts the same table (api-collector-java/constant_test.go).
var constantDependencyExpectation = map[string]string{
	"X-Channel": "header",
	"X-Device":  "header",
}

// constantLibrary answers dependency lookups from constantLibrarySources,
// each through the parser of its language.
type constantLibrary struct {
	t       *testing.T
	java    *javaparser.Parser
	queried []string
}

func (l *constantLibrary) ResolveClass(className string) *javaparser.Class {
	l.queried = append(l.queried, className)
	entry, ok := constantLibrarySources[className]
	if !ok {
		return nil
	}
	var classes []javaparser.Class
	var err error
	if entry.language == "kotlin" {
		classes, err = parseDependencySource(className, []byte(entry.source))
	} else {
		classes, err = l.java.ParseSource([]byte(entry.source))
	}
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
	p, err := ktparser.NewParser()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	dir, err := filepath.Abs("testdata/constant-dependency")
	if err != nil {
		t.Fatal(err)
	}
	results, err := p.ParseDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	java, err := javaparser.NewParser(javaparser.ParserOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer java.Close()

	typeResolver := resolver.New(flattenClasses(results))
	library := &constantLibrary{t: t, java: java}
	typeResolver.SetDependencyResolver(library)
	sm := springmvc.NewParser(typeResolver)
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
