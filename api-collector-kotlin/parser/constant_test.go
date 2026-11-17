package parser

import (
	"reflect"
	"testing"
)

func parseSource(t *testing.T, path, source string) *ParseResult {
	t.Helper()
	p, err := NewParser()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	result, err := p.ParseSource(path, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	return result
}

func constants(c Class) map[string]string {
	out := map[string]string{}
	for _, f := range c.Fields {
		if f.IsConst {
			out[f.Name] = f.DefaultValue
		}
	}
	return out
}

func TestConstValsAreReadWhereTheJVMHoldsThem(t *testing.T) {
	result := parseSource(t, "src/Headers.kt", `package com.example

const val TOP = "X-Top"
val notConst = "plain"

interface Headers {
    companion object {
        const val VERSION = "X-Version"
        val notConstEither = "plain"
        fun helper() = 1
    }
}

object Params {
    const val PAGE = "page"
    val instance = "plain"
}

class Request(val id: Long) {
    var name: String? = null

    companion object Named {
        const val KIND = Params.PAGE
    }
}
`)
	byName := map[string]Class{}
	for _, c := range result.Classes {
		byName[c.Name] = c
	}

	// The const vals of a companion object are static fields of the class it
	// belongs to; nothing else of the companion is read.
	if got := constants(byName["Headers"]); !reflect.DeepEqual(got, map[string]string{"VERSION": `"X-Version"`}) {
		t.Errorf("Headers constants = %v", got)
	}
	if len(byName["Headers"].Fields) != 1 {
		t.Errorf("Headers fields = %+v, want only the const val", byName["Headers"].Fields)
	}
	if got := constants(byName["Params"]); !reflect.DeepEqual(got, map[string]string{"PAGE": `"page"`}) {
		t.Errorf("Params constants = %v", got)
	}
	request := byName["Request"]
	if got := constants(request); !reflect.DeepEqual(got, map[string]string{"KIND": "Params.PAGE"}) {
		t.Errorf("Request constants = %v", got)
	}
	if len(request.Fields) != 3 {
		t.Errorf("Request fields = %+v, want id, name and KIND", request.Fields)
	}
	if _, ok := byName["Named"]; ok {
		t.Errorf("the companion object is read as a class of its own")
	}

	// A top-level const val is a field of the file facade.
	facade, ok := byName["HeadersKt"]
	if !ok || !facade.IsFileFacade || facade.Package != "com.example" {
		t.Fatalf("file facade = %+v, classes %v", facade, result.Classes)
	}
	if got := constants(facade); !reflect.DeepEqual(got, map[string]string{"TOP": `"X-Top"`}) || len(facade.Fields) != 1 {
		t.Errorf("facade fields = %+v", facade.Fields)
	}
}

func TestAFileWithoutTopLevelConstantsHasNoFacade(t *testing.T) {
	result := parseSource(t, "src/Plain.kt", "package com.example\n\nval notConst = \"plain\"\n\nclass Plain(val id: Long)\n")
	for _, c := range result.Classes {
		if c.IsFileFacade {
			t.Errorf("classes = %+v, want no facade", result.Classes)
		}
	}
}

func TestAnnotationArgumentsWrittenAsNamesAreMarked(t *testing.T) {
	result := parseSource(t, "src/Handler.kt", `package com.example

class Handler {
    fun handle(
        @RequestHeader(TENANT) tenant: String,
        @RequestHeader(value = Headers.VERSION, required = false) version: String?,
        @RequestParam(com.example.Params.PAGE) page: Int,
        @RequestParam(name = "size", defaultValue = "20") size: Int,
        @RequestParam("q") q: String
    ) {
    }
}
`)
	params := result.Classes[0].Methods[0].Parameters
	tests := []struct {
		param      string
		wantParams map[string]string
		wantRefs   map[string]bool
	}{
		// A lone name used to be dropped.
		{"tenant", map[string]string{"value": "TENANT"}, map[string]bool{"value": true}},
		{"version", map[string]string{"value": "Headers.VERSION", "required": "false"}, map[string]bool{"value": true}},
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
