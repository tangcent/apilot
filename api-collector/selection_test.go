package collector

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestSelectedFiles(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "A.kt"), filepath.Join(dir, "B.kt")

	for _, tc := range []struct {
		name string
		ctx  CollectContext
		want []string
	}{
		{"no file selects the whole directory", CollectContext{SourceDir: dir}, nil},
		{"a single file", CollectContext{SourceDir: dir, SourceFile: a}, []string{a}},
		{"several files", CollectContext{SourceDir: dir, SourceFiles: []string{a, b}}, []string{a, b}},
		{"SourceFiles wins over SourceFile", CollectContext{SourceDir: dir, SourceFile: a, SourceFiles: []string{b}}, []string{b}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.ctx.SelectedFiles(); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("SelectedFiles() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFileSelectionContains(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a", "A.kt"), filepath.Join(dir, "b", "B.kt")

	sel := NewFileSelection([]string{a})
	if !sel.Contains(a) {
		t.Errorf("the selected file should be contained")
	}
	if !sel.Contains(filepath.Join(dir, "a", "..", "a", "A.kt")) {
		t.Errorf("an unclean spelling of the selected file should be contained")
	}
	if sel.Contains(b) {
		t.Errorf("a file that was not selected should not be contained")
	}
	if sel.Contains("") {
		t.Errorf("a class without a file, such as a dependency type, is never selected")
	}
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		if !sel.Contains(strings.ToUpper(a)) {
			t.Errorf("paths compare case-insensitively on %s", runtime.GOOS)
		}
	}
}

func TestNilFileSelectionContainsEverything(t *testing.T) {
	var sel *FileSelection
	if !sel.Contains(filepath.Join(t.TempDir(), "A.kt")) || !sel.Contains("") {
		t.Errorf("a nil selection stands for the whole project and contains every file")
	}
	if missing := sel.Unmatched([]string{"x"}); len(missing) != 0 {
		t.Errorf("a nil selection has nothing to reach, got %v", missing)
	}
	if got := NewFileSelection(nil); got != nil {
		t.Errorf("no paths should give the nil selection, got %+v", got)
	}
	if got := (CollectContext{}).FileSelection(); got != nil {
		t.Errorf("a context without files should give the nil selection, got %+v", got)
	}
}

func TestFileSelectionUnmatched(t *testing.T) {
	dir := t.TempDir()
	a, b, c := filepath.Join(dir, "A.kt"), filepath.Join(dir, "B.kt"), filepath.Join(dir, "build", "C.kt")

	sel := NewFileSelection([]string{a, b, c, a})
	missing := sel.Unmatched([]string{a, "", filepath.Join(dir, "x", "..", "B.kt"), filepath.Join(dir, "Other.kt")})
	if want := []string{c}; !reflect.DeepEqual(missing, want) {
		t.Errorf("Unmatched = %v, want %v: a duplicate selection is one file, and an unclean spelling still reaches it", missing, want)
	}
}

func TestCollectContextJSONKeepsSourceFile(t *testing.T) {
	data, err := json.Marshal(CollectContext{SourceDir: "/p", SourceFile: "/p/A.kt", SourceFiles: []string{"/p/A.kt"}})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), `{"sourceDir":"/p","sourceFile":"/p/A.kt","sourceFiles":["/p/A.kt"]}`; got != want {
		t.Errorf("context JSON = %s, want %s", got, want)
	}

	data, err = json.Marshal(CollectContext{SourceDir: "/p"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), `{"sourceDir":"/p"}`; got != want {
		t.Errorf("a directory export must not grow new keys: context JSON = %s, want %s", got, want)
	}
}
