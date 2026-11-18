package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// testdataJavaFilesProject is a Maven module of three controllers and the
// types they share, all in its sources.
func testdataJavaFilesProject() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", "javafiles")
}

func javaFilesSource(name string) string {
	return filepath.Join(testdataJavaFilesProject(), "src", "main", "java", "com", "example", "files", name)
}

func exportedEndpoints(t *testing.T, document string) []map[string]any {
	t.Helper()
	var doc struct {
		Endpoints []map[string]any `json:"endpoints"`
	}
	if err := json.Unmarshal([]byte(document), &doc); err != nil {
		t.Fatalf("the export should be valid JSON: %v\n%s", err, document)
	}
	return doc.Endpoints
}

// goldenEndpointsOf returns the endpoints of the directory export that belong
// to the given controllers, in the order of that export.
func goldenEndpointsOf(t *testing.T, folders ...string) []map[string]any {
	t.Helper()
	golden, err := os.ReadFile(filepath.Join(testdataJavaFilesProject(), "json.golden"))
	if err != nil {
		t.Fatal(err)
	}
	var out []map[string]any
	for _, ep := range exportedEndpoints(t, string(golden)) {
		for _, folder := range folders {
			if ep["folder"] == folder {
				out = append(out, ep)
			}
		}
	}
	return out
}

func endpointKeys(endpoints []map[string]any) []string {
	var keys []string
	for _, ep := range endpoints {
		keys = append(keys, ep["folder"].(string)+"."+ep["name"].(string))
	}
	return keys
}

// A directory export is what it was before files could be selected: the golden
// was written by the build that preceded the selection of files.
func TestJavaFilesDirectoryExportUnchanged(t *testing.T) {
	dir := testdataJavaFilesProject()
	assertGoldenFile(t, filepath.Join(dir, "json.golden"), dir, "--collector", "java", "--formatter", "json", "--no-deps")
}

// A file-level export holds the endpoints the directory export holds for the
// file, whole: the types they refer to are declared in other files.
func TestSelectedFileExportMatchesTheDirectoryExport(t *testing.T) {
	for _, tc := range []struct {
		file   string
		folder string
		count  string
	}{
		{"FilesUserController.java", "FilesUserController", "endpoints: 2"},
		{"FilesOrderController.java", "FilesOrderController", "endpoints: 2"},
		{"FilesItemController.java", "FilesItemController", "endpoints: 1"},
	} {
		t.Run(tc.file, func(t *testing.T) {
			stdout, stderr, exitCode := runCLISeparate(javaFilesSource(tc.file), "--formatter", "json", "--no-deps")
			if exitCode != 0 {
				t.Fatalf("exit code %d, stderr: %s", exitCode, stderr)
			}
			got, want := exportedEndpoints(t, stdout), goldenEndpointsOf(t, tc.folder)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("endpoints = %v, want the directory export's %v", endpointKeys(got), endpointKeys(want))
			}
			if !strings.Contains(stderr, tc.count) {
				t.Errorf("stderr should report %q, got: %s", tc.count, stderr)
			}
		})
	}
}

// Several files of one project go in one run. Flags may sit before, between and
// after them, a boolean flag included, and the document follows the order of
// the directory export however the files are named.
func TestSeveralSelectedFilesExportInOneRun(t *testing.T) {
	user, item := javaFilesSource("FilesUserController.java"), javaFilesSource("FilesItemController.java")

	stdout, stderr, exitCode := runCLISeparate("--no-deps", user, "--formatter", "json", item)
	if exitCode != 0 {
		t.Fatalf("exit code %d, stderr: %s", exitCode, stderr)
	}
	got, want := exportedEndpoints(t, stdout), goldenEndpointsOf(t, "FilesItemController", "FilesUserController")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("endpoints = %v, want the directory export's %v", endpointKeys(got), endpointKeys(want))
	}
	if !strings.Contains(stderr, "endpoints: 3") {
		t.Errorf("stderr should report 3 endpoints, got: %s", stderr)
	}

	reversed, _, exitCode := runCLISeparate(item, user, "--formatter", "json", "--no-deps")
	if exitCode != 0 {
		t.Fatalf("exit code %d for the reversed order", exitCode)
	}
	if reversed != stdout {
		t.Errorf("the same files in another order should export the same document")
	}
}

func TestSelectedFilesWithMethodFilter(t *testing.T) {
	files := []string{
		javaFilesSource("FilesUserController.java"),
		javaFilesSource("FilesOrderController.java"),
		javaFilesSource("FilesItemController.java"),
	}

	// The filter applies to every selected file.
	args := append(append([]string{}, files...), "--method", "list", "--formatter", "json", "--no-deps")
	stdout, stderr, exitCode := runCLISeparate(args...)
	if exitCode != 0 {
		t.Fatalf("exit code %d, stderr: %s", exitCode, stderr)
	}
	want := []string{"FilesItemController.list", "FilesOrderController.list", "FilesUserController.list"}
	if got := endpointKeys(exportedEndpoints(t, stdout)); !reflect.DeepEqual(got, want) {
		t.Errorf("endpoints = %v, want %v", got, want)
	}

	// A file with a method of that name alone.
	stdout, stderr, exitCode = runCLISeparate(files[0], files[1], "--method", "create", "--formatter", "json", "--no-deps")
	if exitCode != 0 {
		t.Fatalf("exit code %d, stderr: %s", exitCode, stderr)
	}
	if got := endpointKeys(exportedEndpoints(t, stdout)); !reflect.DeepEqual(got, []string{"FilesOrderController.create"}) {
		t.Errorf("endpoints = %v, want FilesOrderController.create", got)
	}
}

func TestSelectedFilesOfDifferentProjectsFail(t *testing.T) {
	other := t.TempDir()
	if err := os.WriteFile(filepath.Join(other, "pom.xml"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	stray := filepath.Join(other, "StrayController.java")
	if err := os.WriteFile(stray, []byte("class StrayController {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, exitCode := runCLISeparate(javaFilesSource("FilesUserController.java"), stray, "--formatter", "json", "--no-deps")
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d\nstdout: %s\nstderr: %s", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Errorf("no document should be written on failure, got: %q", stdout)
	}
	for _, want := range []string{"different projects", "FilesUserController.java", "StrayController.java", "--project-root"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr should mention %q, got: %s", want, stderr)
		}
	}
}

func TestSelectedFilesWithADirectoryFail(t *testing.T) {
	_, stderr, exitCode := runCLISeparate(testdataJavaFilesProject(), javaFilesSource("FilesUserController.java"), "--no-deps")
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d, stderr: %s", exitCode, stderr)
	}
	if !strings.Contains(stderr, "one directory or one or more source files") {
		t.Errorf("stderr should explain what can be exported together, got: %s", stderr)
	}
}
