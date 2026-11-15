package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBlankUnreadAnnotationArguments(t *testing.T) {
	for _, tc := range []struct {
		name, in, want string
	}{
		{"unread arguments are blanked",
			`@Size(max = 64) var a`,
			`@Size(        ) var a`},
		{"arguments the collector reads are kept",
			`@RequestParam("id") @JsonProperty(value = "x") @Schema(description = "d")`,
			`@RequestParam("id") @JsonProperty(value = "x") @Schema(description = "d")`},
		{"qualified and use-site names normalise like the parser",
			`@jakarta.validation.constraints.Size(max = 1) @field:Size(min = 2) @org.springframework.web.bind.annotation.GetMapping("/a")`,
			`@jakarta.validation.constraints.Size(       ) @field:Size(       ) @org.springframework.web.bind.annotation.GetMapping("/a")`},
		{"a parenthesis in a string does not close the list",
			`@Size(message = "a)b", max = 1) x`,
			`@Size(                        ) x`},
		{"a parenthesis in a character literal does not close the list",
			`@Mark(c = ')', d = '(') x`,
			`@Mark(                ) x`},
		{"a parenthesis in a comment does not close the list",
			`@Size(/* ) */ max = 1) x`,
			`@Size(               ) x`},
		{"a brace in a template does not end the string",
			`@Mark(m = "${f('}')}") x`,
			`@Mark(               ) x`},
		{"a raw string holds its quotes",
			`@Mark(m = """a)"b""") x`,
			`@Mark(              ) x`},
		{"nested parentheses",
			`@Mark(v = f(1, (2))) x`,
			`@Mark(             ) x`},
		{"an @ in a string or comment is not an annotation",
			`val s = "@Size(1)" // @Size(2)`,
			`val s = "@Size(1)" // @Size(2)`},
		{"line breaks are kept",
			"@Size(\n  max = 1\n) var a",
			"@Size(\n         \n) var a"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := string(blankUnreadAnnotationArguments([]byte(tc.in)))
			if got != tc.want {
				t.Fatalf("\n got  %q\n want %q", got, tc.want)
			}
			if len(got) != len(tc.in) {
				t.Fatalf("length changed: %d -> %d", len(tc.in), len(got))
			}
		})
	}
}

func TestDeclaresController(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want bool
	}{
		{"@RestController class A", true},
		{"@Controller class A", true},
		{"@org.springframework.web.bind.annotation.RestController class A", true},
		{"@RestControllerAdvice class A", false},
		{`val s = "@RestController"`, false},
		{`val s = """@RestController"""`, false},
		{"// @RestController\nclass A", false},
		{"/* @Controller */ class A", false},
	} {
		if got := declaresController([]byte(tc.in)); got != tc.want {
			t.Errorf("declaresController(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestTopLevelDeclarations(t *testing.T) {
	src := "@file:JvmName(\"X\")\npackage a.b\n\nimport c.D\n\n" +
		"/** Doc of A. */\n@Ann\n@Other(\"x\")\nclass A(\n    val x: Int,\n) : Base(),\n    Iface {\n    val y = \"}\"\n}\n\n" +
		"// a comment\nfun f() = 1\n\n" +
		"annotation class T(val g: Array<String> = [])\n" +
		"class B\n"
	var got []string
	for _, d := range topLevelDeclarations([]byte(src)) {
		got = append(got, strings.TrimSpace(src[d.start:d.end]))
	}
	// A declaration starts at its first annotation; the doc comment above it
	// is not cut out of the file by isolation, which keeps every comment.
	want := []string{
		"@Ann\n@Other(\"x\")\nclass A(\n    val x: Int,\n) : Base(),\n    Iface {\n    val y = \"}\"\n}\n\n// a comment",
		"fun f() = 1",
		"annotation class T(val g: Array<String> = [])",
		"class B",
	}
	if len(got) != len(want) {
		t.Fatalf("declarations:\n%q\nwant\n%q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("declaration %d:\n got  %q\n want %q", i, got[i], want[i])
		}
	}
}

func TestIsolateKeepsHeaderDeclarationAndComments(t *testing.T) {
	src := "package a\n\nimport b.C\n\n/** A. */\nclass A\n\n/** B. */\nclass B(val s: String = \"/* not a comment\")\n"
	decls := topLevelDeclarations([]byte(src))
	if len(decls) != 2 {
		t.Fatalf("declarations = %d", len(decls))
	}
	got := string(isolate([]byte(src), decls[0]))
	if len(got) != len(src) || strings.Count(got, "\n") != strings.Count(src, "\n") {
		t.Fatal("isolation must keep positions and line breaks")
	}
	for _, kept := range []string{"package a", "import b.C", "/** A. */", "class A", "/** B. */"} {
		if !strings.Contains(got, kept) {
			t.Errorf("isolated source lost %q:\n%s", kept, got)
		}
	}
	for _, gone := range []string{"class B", "not a comment"} {
		if strings.Contains(got, gone) {
			t.Errorf("isolated source kept %q from another declaration:\n%s", gone, got)
		}
	}
}

func TestIsolateDoesNotLeakKDocAcrossDeclarations(t *testing.T) {
	// A syntax error sends the file through the per-declaration recovery in
	// parseDamaged. The KDoc of one declaration must not be attached to a
	// later, undocumented one when the declaration in between is blanked.
	src := "package com.example\n\n" +
		"class First {\n    var a: String? = null\n}\n\n" +
		"/** KDoc of A. */\nclass A {\n    var x: String? = null\n}\n\n" +
		"class B {\n    var b: String? = null\n}\n\n" +
		"class Broken {\n    var c: String? = )\n}\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "Models.kt")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := NewParser()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	result, err := p.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if result.Error != nil {
		t.Fatalf("unexpected parse error: %v", result.Error)
	}
	kdocs := map[string]string{}
	for _, c := range result.Classes {
		kdocs[c.Name] = c.KDoc
	}
	for _, name := range []string{"A", "B"} {
		if _, ok := kdocs[name]; !ok {
			t.Fatalf("class %s was not extracted; kdocs = %v", name, kdocs)
		}
	}
	if !strings.Contains(kdocs["A"], "KDoc of A.") {
		t.Errorf("A lost its own KDoc: %q", kdocs["A"])
	}
	if kdocs["B"] != "" {
		t.Errorf("B inherited another declaration's KDoc: %q", kdocs["B"])
	}
}

func TestSkipRawString(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want int // index just past the string
	}{
		{`"""a""" + x`, 7},
		// A run of three or more quotes closes the string; the extra quotes
		// are content, as the Kotlin lexer reads them.
		{`"""a"""" + x`, 8},
		// A template may hold quotes.
		{`"""${"\"\"\""}""" + x`, 17},
	} {
		if got := skipRawString([]byte(tc.in), 0); got != tc.want {
			t.Errorf("skipRawString(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}
