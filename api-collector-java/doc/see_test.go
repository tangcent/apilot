package doc

import (
	"reflect"
	"testing"
)

func TestSeeReferences(t *testing.T) {
	for _, tc := range []struct {
		name string
		doc  string
		want []string
	}{
		{"none", "/** State name. */", nil},
		{"one tag", "/**\n * State name.\n *\n * @see OrderState\n */", []string{"OrderState"}},
		{"words after the reference", "/**\n * @see com.example.OrderState the state table\n */", []string{"com.example.OrderState"}},
		{"several tags, in order", "/**\n * @see OrderState\n * @see Other#code\n */", []string{"OrderState", "Other#code"}},
		{"one line", "/** @see OrderState */", []string{"OrderState"}},
		{"KDoc link", "/** @see [OrderState] */", []string{"OrderState"}},
		{"a quoted string names nothing", `/** @see "the state table" */`, nil},
		{"a URL names nothing", `/** @see <a href="https://example.com">states</a> */`, nil},
		{"a tag without a reference", "/**\n * @see\n */", nil},
		{"a word that only starts like the tag", "/** @seealso OrderState */", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := SeeReferences(tc.doc); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("SeeReferences = %q, want %q", got, tc.want)
			}
		})
	}
}
