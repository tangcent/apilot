package resolver

import (
	"testing"

	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
)

// What is shown of a type that cannot be resolved, and what is reported, is the
// name the file wrote for its classes, not the class of the JDK that the adapter
// writes for a standard type of Kotlin: `Pair<Int, String>` is `Pair` of a
// class that nothing declares, whose arguments are the classes of the JDK
// under the names the shared resolver has always shown them by.
func TestResolve_AnUnresolvedTypeIsShownByTheNamesItsArgumentsAreWrittenWith(t *testing.T) {
	item := scopeShadow("com.example.kcdisplay", "Item", "own")
	source := ktparser.Class{
		Name: "Caller", Package: "com.example.kcdisplay", FilePath: "display/Caller.kt",
		Imports: []ktparser.Import{{Path: "com.example.kclib.Pair"}, {Path: "com.example.kclib.Optional"}},
	}
	for view := range scopeViews {
		t.Run(view, func(t *testing.T) {
			r := New([]ktparser.Class{item, source})
			for spelling, want := range map[string]string{
				"Pair<Int,String>":          "Pair<Integer,String>",
				"Pair<Int, String>":         "Pair<Integer,String>",
				"Optional<Item>":            "Optional<Item>",
				"Pair<List<Item>, Long>":    "Pair<List<Item>,Long>",
				"Optional<Map<String,Int>>": "Optional<Map<String,Integer>>",
			} {
				got := scopeModelShape(scopeViews[view](r, &source, spelling))
				if got != want {
					t.Errorf("%s shown as %s, want %s", spelling, got, want)
				}
			}
			for name := range r.Unresolved() {
				if len(name) > 0 && name[0] != 'P' && name[0] != 'O' {
					t.Errorf("unresolved %q: reported by a name nobody wrote", name)
				}
			}
		})
	}
}
