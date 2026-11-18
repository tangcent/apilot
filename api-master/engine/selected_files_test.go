package engine

import (
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

// recordingCollector remembers the context it was asked to collect and answers
// with fixed endpoints. It does not implement collector.FileSelector, so the
// engine must tell the selected files' endpoints apart by file name.
type recordingCollector struct {
	name      string
	endpoints []collector.ApiEndpoint
	got       collector.CollectContext
}

func (m *recordingCollector) Name() string                 { return m.name }
func (m *recordingCollector) SupportedLanguages() []string { return []string{m.name} }

func (m *recordingCollector) Collect(ctx collector.CollectContext) ([]collector.ApiEndpoint, error) {
	m.got = ctx
	return m.endpoints, nil
}

// selectingCollector is a recordingCollector that honours the selected files.
type selectingCollector struct{ recordingCollector }

func (m *selectingCollector) SelectsFiles() bool { return true }

// writeProject creates a project directory that holds a build file and the
// given relative files, and returns the directory.
func writeProject(t *testing.T, files ...string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pom.xml"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		path := filepath.Join(dir, file)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("class X {}"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func abs(t *testing.T, path string) string {
	t.Helper()
	out, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// runCaptured runs cfg against c and returns the endpoints the formatter got.
func runCaptured(t *testing.T, c collector.Collector, cfg Config) ([]collector.ApiEndpoint, error) {
	t.Helper()
	RegisterCollector(c)
	var captured []collector.ApiEndpoint
	RegisterFormatter(&captureFormatter{name: "capture-selected", capture: &captured})
	cfg.CollectorName = c.Name()
	cfg.FormatterName = "capture-selected"
	var err error
	captureStderr(t, func() { err = Run(cfg) })
	return captured, err
}

func folders(endpoints []collector.ApiEndpoint) string {
	var names []string
	for _, ep := range endpoints {
		names = append(names, ep.Folder+"."+ep.Name)
	}
	return strings.Join(names, ",")
}

var threeControllers = []collector.ApiEndpoint{
	{Name: "list", Folder: "UserController", Path: "/users", Method: "GET", Protocol: "http"},
	{Name: "list", Folder: "OrderController", Path: "/orders", Method: "GET", Protocol: "http"},
	{Name: "get", Folder: "OrderController", Path: "/orders/{id}", Method: "GET", Protocol: "http"},
	{Name: "list", Folder: "ItemController", Path: "/items", Method: "GET", Protocol: "http"},
}

func TestRun_SelectionReachesTheCollector(t *testing.T) {
	dir := writeProject(t, "UserController.java", "OrderController.java")
	user := abs(t, filepath.Join(dir, "UserController.java"))
	order := abs(t, filepath.Join(dir, "OrderController.java"))

	for _, tc := range []struct {
		name  string
		paths []string
		file  string
		files []string
	}{
		// SourceFile stays set for one file, so a collector written before
		// SourceFiles existed keeps working.
		{"single file", []string{user}, user, []string{user}},
		// With several files SourceFile would stand for one of many, so it is
		// left empty and the whole selection travels in SourceFiles.
		{"several files", []string{user, order}, "", []string{user, order}},
		{"the same file twice is one file", []string{user, user}, user, []string{user}},
		{"a directory selects no file", []string{dir}, "", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &recordingCollector{name: "recording", endpoints: threeControllers}
			_, err := runCaptured(t, c, Config{SourcePaths: tc.paths, AllowEmpty: true})
			if err != nil {
				t.Fatalf("Run failed: %v", err)
			}
			if c.got.SourceFile != tc.file {
				t.Errorf("SourceFile = %q, want %q", c.got.SourceFile, tc.file)
			}
			if !reflect.DeepEqual(c.got.SourceFiles, tc.files) {
				t.Errorf("SourceFiles = %v, want %v", c.got.SourceFiles, tc.files)
			}
			if !reflect.DeepEqual(c.got.SelectedFiles(), tc.files) {
				t.Errorf("SelectedFiles() = %v, want %v", c.got.SelectedFiles(), tc.files)
			}
			if got, want := abs(t, c.got.SourceDir), abs(t, dir); got != want {
				t.Errorf("SourceDir = %q, want the project root %q", got, want)
			}
		})
	}
}

// SourceDir alone still names a single file, as it did before SourcePaths.
func TestRun_SourceDirNamesASingleFile(t *testing.T) {
	dir := writeProject(t, "UserController.java")
	user := abs(t, filepath.Join(dir, "UserController.java"))

	c := &recordingCollector{name: "recording", endpoints: threeControllers}
	endpoints, err := runCaptured(t, c, Config{SourceDir: user})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if c.got.SourceFile != user || !reflect.DeepEqual(c.got.SourceFiles, []string{user}) {
		t.Errorf("context = %+v, want the single file %s", c.got, user)
	}
	if got := folders(endpoints); got != "UserController.list" {
		t.Errorf("endpoints = %s, want UserController.list", got)
	}
}

func TestRun_CollectorWithoutSelectionIsFilteredByFileName(t *testing.T) {
	dir := writeProject(t, "UserController.java", "ItemController.java", "OrderController.java")
	files := []string{filepath.Join(dir, "ItemController.java"), filepath.Join(dir, "UserController.java")}

	c := &recordingCollector{name: "recording", endpoints: threeControllers}
	endpoints, err := runCaptured(t, c, Config{SourcePaths: files})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if got, want := folders(endpoints), "UserController.list,ItemController.list"; got != want {
		t.Errorf("endpoints = %s, want %s", got, want)
	}
}

// A collector that selects by file knows which file declared each endpoint, so
// the engine does not second-guess it with the file name: a file may declare a
// controller named otherwise.
func TestRun_SelectingCollectorIsTrusted(t *testing.T) {
	dir := writeProject(t, "Api.java")

	c := &selectingCollector{recordingCollector{name: "selecting", endpoints: threeControllers[:2]}}
	endpoints, err := runCaptured(t, c, Config{SourceDir: filepath.Join(dir, "Api.java")})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if got, want := folders(endpoints), "UserController.list,OrderController.list"; got != want {
		t.Errorf("endpoints = %s, want %s", got, want)
	}
}

func TestRun_MethodFilterAppliesToEverySelectedFile(t *testing.T) {
	dir := writeProject(t, "UserController.java", "OrderController.java", "ItemController.java")
	files := []string{filepath.Join(dir, "UserController.java"), filepath.Join(dir, "OrderController.java")}

	c := &selectingCollector{recordingCollector{name: "selecting", endpoints: threeControllers[:3]}}
	endpoints, err := runCaptured(t, c, Config{SourcePaths: files, MethodFilter: "list"})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if got, want := folders(endpoints), "UserController.list,OrderController.list"; got != want {
		t.Errorf("endpoints = %s, want %s", got, want)
	}
}

func TestRun_FilesOfDifferentProjectsFail(t *testing.T) {
	first := writeProject(t, "UserController.java")
	second := writeProject(t, "OrderController.java")

	c := &recordingCollector{name: "recording", endpoints: threeControllers}
	_, err := runCaptured(t, c, Config{SourcePaths: []string{
		filepath.Join(first, "UserController.java"),
		filepath.Join(second, "OrderController.java"),
	}})
	if err == nil {
		t.Fatal("expected files of two projects to fail")
	}
	for _, want := range []string{"different projects", "UserController.java", "OrderController.java", "--project-root"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q, got: %v", want, err)
		}
	}
}

func TestRun_ProjectRootExportsFilesOfSeveralModulesTogether(t *testing.T) {
	root := writeProject(t, "user/UserController.java", "order/OrderController.java")
	for _, module := range []string{"user", "order"} {
		if err := os.WriteFile(filepath.Join(root, module, "pom.xml"), nil, 0644); err != nil {
			t.Fatal(err)
		}
	}
	files := []string{filepath.Join(root, "user", "UserController.java"), filepath.Join(root, "order", "OrderController.java")}

	c := &recordingCollector{name: "recording", endpoints: threeControllers}
	if _, err := runCaptured(t, c, Config{SourcePaths: files}); err == nil || !strings.Contains(err.Error(), "different projects") {
		t.Fatalf("modules with a build file each are different projects, got: %v", err)
	}

	endpoints, err := runCaptured(t, c, Config{SourcePaths: files, ProjectRoot: root})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if c.got.SourceDir != root {
		t.Errorf("SourceDir = %q, want %q", c.got.SourceDir, root)
	}
	if got, want := folders(endpoints), "UserController.list,OrderController.list,OrderController.get"; got != want {
		t.Errorf("endpoints = %s, want %s", got, want)
	}
}

func TestRun_DirectoryAndFileTogetherFail(t *testing.T) {
	dir := writeProject(t, "UserController.java")

	c := &recordingCollector{name: "recording", endpoints: threeControllers}
	_, err := runCaptured(t, c, Config{SourcePaths: []string{dir, filepath.Join(dir, "UserController.java")}})
	if err == nil || !strings.Contains(err.Error(), "one directory or one or more source files") {
		t.Fatalf("expected a directory with a file to fail, got: %v", err)
	}
}

func TestRun_MissingSelectedFileFails(t *testing.T) {
	dir := writeProject(t, "UserController.java")

	c := &recordingCollector{name: "recording", endpoints: threeControllers}
	_, err := runCaptured(t, c, Config{SourcePaths: []string{
		filepath.Join(dir, "UserController.java"),
		filepath.Join(dir, "Missing.java"),
	}})
	if err == nil || !strings.Contains(err.Error(), "Missing.java") {
		t.Fatalf("expected the missing file to be named, got: %v", err)
	}
}

func TestParseInterleaved(t *testing.T) {
	for _, tc := range []struct {
		name      string
		args      []string
		paths     []string
		formatter string
		noDeps    bool
	}{
		{"paths then flags", []string{"a.kt", "b.kt", "--formatter", "json"}, []string{"a.kt", "b.kt"}, "json", false},
		{"flags then paths", []string{"--formatter", "json", "a.kt", "b.kt"}, []string{"a.kt", "b.kt"}, "json", false},
		// A boolean flag must not take the path after it as its value.
		{"boolean flag before a path", []string{"--no-deps", "a.kt", "--formatter", "json"}, []string{"a.kt"}, "json", true},
		{"flags between paths", []string{"a.kt", "--no-deps", "b.kt", "--formatter=json", "c.kt"}, []string{"a.kt", "b.kt", "c.kt"}, "json", true},
		{"one path", []string{"a.kt"}, []string{"a.kt"}, "markdown", false},
		{"no path", []string{"--no-deps"}, nil, "markdown", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := flag.NewFlagSet("export", flag.ContinueOnError)
			formatter := fs.String("formatter", "markdown", "")
			noDeps := fs.Bool("no-deps", false, "")

			paths, err := parseInterleaved(fs, tc.args)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(paths, tc.paths) {
				t.Errorf("paths = %v, want %v", paths, tc.paths)
			}
			if *formatter != tc.formatter || *noDeps != tc.noDeps {
				t.Errorf("formatter = %q, no-deps = %v, want %q, %v", *formatter, *noDeps, tc.formatter, tc.noDeps)
			}
		})
	}
}
