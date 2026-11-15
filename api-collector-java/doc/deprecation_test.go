package doc

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

func TestDeprecationOf(t *testing.T) {
	for _, tc := range []struct {
		name        string
		annotations []parser.Annotation
		doc         string
		want        Deprecation
	}{
		{"Java's annotation carries no note",
			[]parser.Annotation{{Name: "Deprecated"}}, "",
			Deprecation{Deprecated: true}},
		{"Kotlin's positional message",
			[]parser.Annotation{{Name: "Deprecated", Params: map[string]string{"value": "Use v2", "level": "DeprecationLevel.WARNING"}}}, "",
			Deprecation{Deprecated: true, Note: "Use v2"}},
		{"Kotlin's named message, qualified",
			[]parser.Annotation{{Name: "kotlin.Deprecated", Params: map[string]string{"message": "Use v2"}}}, "",
			Deprecation{Deprecated: true, Note: "Use v2"}},
		{"a doc tag, continuation included",
			nil, "/**\n * Old lookup.\n *\n * @deprecated use the search\n *     endpoint instead\n * @return the user\n */",
			Deprecation{Deprecated: true, Note: "use the search\nendpoint instead"}},
		{"a bare doc tag",
			nil, "/** @deprecated */",
			Deprecation{Deprecated: true}},
		{"the annotation's message wins over the tag",
			[]parser.Annotation{{Name: "Deprecated", Params: map[string]string{"message": "annotation"}}}, "/** @deprecated tag */",
			Deprecation{Deprecated: true, Note: "annotation"}},
		{"Java's annotation takes the tag's text",
			[]parser.Annotation{{Name: "Deprecated"}}, "/** @deprecated tag */",
			Deprecation{Deprecated: true, Note: "tag"}},
		{"another tag or annotation is not a deprecation",
			[]parser.Annotation{{Name: "DeprecatedSoon"}}, "/** @deprecatedSince 2 */",
			Deprecation{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := DeprecationOf(tc.annotations, tc.doc); got != tc.want {
				t.Errorf("DeprecationOf = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestEndpointDeprecation(t *testing.T) {
	method := Deprecation{Deprecated: true, Note: "method"}
	class := Deprecation{Deprecated: true, Note: "class"}
	if got := EndpointDeprecation(method, class); got != method {
		t.Errorf("the method's own deprecation comes first: %+v", got)
	}
	if got := EndpointDeprecation(Deprecation{}, Deprecation{}, class); got != class {
		t.Errorf("an endpoint of a deprecated class is deprecated: %+v", got)
	}
	if got := EndpointDeprecation(Deprecation{}, Deprecation{}); got.Deprecated {
		t.Errorf("nothing deprecated: %+v", got)
	}
}
