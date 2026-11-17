package parser

import (
	"reflect"
	"testing"
)

func TestParser_AnnotationArgumentsWrittenAsNamesAreMarked(t *testing.T) {
	p, err := NewParser(ParserOptions{LogLevel: LogLevelError})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	classes, err := p.ParseSource([]byte(`package com.example;

import static com.example.Headers.VERSION;

class Handler {
    void handle(@RequestHeader(Headers.TENANT) String tenant,
                @RequestHeader(value = VERSION, required = false) String version,
                @RequestParam(com.example.Params.PAGE) int page,
                @RequestParam(name = "size", defaultValue = "20") int size,
                @RequestParam("q") String q) {
    }
}
`))
	if err != nil {
		t.Fatal(err)
	}
	params := classes[0].Methods[0].Parameters
	tests := []struct {
		param      string
		wantParams map[string]string
		wantRefs   map[string]bool
	}{
		{"tenant", map[string]string{"value": "Headers.TENANT"}, map[string]bool{"value": true}},
		// An identifier value used to overwrite the key it was paired with.
		{"version", map[string]string{"value": "VERSION", "required": "false"}, map[string]bool{"value": true}},
		{"page", map[string]string{"value": "com.example.Params.PAGE"}, map[string]bool{"value": true}},
		{"size", map[string]string{"name": "size", "defaultValue": "20"}, nil},
		{"q", map[string]string{"value": "q"}, nil},
	}
	for i, tt := range tests {
		ann := params[i].Annotations[0]
		if params[i].Name != tt.param {
			t.Fatalf("parameter %d = %s, want %s", i, params[i].Name, tt.param)
		}
		if !reflect.DeepEqual(ann.Params, tt.wantParams) {
			t.Errorf("%s: params = %v, want %v", tt.param, ann.Params, tt.wantParams)
		}
		if !reflect.DeepEqual(ann.Refs, tt.wantRefs) {
			t.Errorf("%s: refs = %v, want %v", tt.param, ann.Refs, tt.wantRefs)
		}
	}
}

func TestParser_KeepsTheInitializersOfStringConstants(t *testing.T) {
	p, err := NewParser(ParserOptions{LogLevel: LogLevelError})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	classes, err := p.ParseSource([]byte(`package com.example;

public final class Headers {
    public static final String VERSION = "X-Version";
    public static final String ALIAS = Base.VERSION;
    static final String QUOTED = "a\"b";
    public static final String JOINED = "X-" + "Joined";
    public static final int LIMIT = 10;
    public final String instance = "not a constant";
    private String plain;
}

interface Names {
    String TENANT = "X-Tenant";
    String SAME = TENANT;
}
`))
	if err != nil {
		t.Fatal(err)
	}
	constants := func(c Class) map[string]string {
		out := map[string]string{}
		for _, f := range c.Fields {
			out[f.Name] = f.Constant
		}
		return out
	}

	want := map[string]string{
		"VERSION":  `"X-Version"`,
		"ALIAS":    "Base.VERSION",
		"QUOTED":   `"a\"b"`,
		"JOINED":   "",
		"LIMIT":    "",
		"instance": "",
		"plain":    "",
	}
	if got := constants(classes[0]); !reflect.DeepEqual(got, want) {
		t.Errorf("Headers constants = %v, want %v", got, want)
	}

	// The fields of an interface are constants whether or not they say so.
	names := classes[1]
	if got, want := constants(names), map[string]string{"TENANT": `"X-Tenant"`, "SAME": "TENANT"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Names constants = %v, want %v", got, want)
	}
	for _, f := range names.Fields {
		if !f.IsStatic || !f.IsFinal {
			t.Errorf("interface field %s: static %v, final %v; want both", f.Name, f.IsStatic, f.IsFinal)
		}
	}
}
