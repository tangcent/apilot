package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var (
	binaryPath string

	// updateGolden rewrites the golden files with the current CLI output.
	// Use it deliberately: `go test ./apilot-cli/... -update`.
	updateGolden = flag.Bool("update", false, "update golden files with the current CLI output")
)

func TestMain(m *testing.M) {
	flag.Parse()

	var err error
	binaryPath, err = buildBinary()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build binary: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(filepath.Dir(binaryPath))

	os.Exit(m.Run())
}

func buildBinary() (string, error) {
	_, filename, _, _ := runtime.Caller(0)
	cliDir := filepath.Dir(filename)

	tmpDir, err := os.MkdirTemp("", "apilot-test-*")
	if err != nil {
		return "", err
	}

	binaryName := "apilot-test"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(tmpDir, binaryName)

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = cliDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("build failed: %v\n%s", err, output)
	}

	return binaryPath, nil
}

func runCLI(args ...string) (string, error) {
	cmd := exec.Command(binaryPath, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func runCLIWithExitCode(args ...string) (string, int) {
	cmd := exec.Command(binaryPath, args...)
	output, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}
	return string(output), exitCode
}

// runCLIOnStdout runs the CLI and returns only what it wrote to stdout.
// Collector diagnostics and the unresolved-type summary go to stderr, so
// mixing them into a golden comparison would make the baseline depend on
// log settings rather than on the exported API description.
func runCLIOnStdout(args ...string) (string, int) {
	stdout, _, exitCode := runCLISeparate(args...)
	return stdout, exitCode
}

// runCLISeparate runs the CLI and returns stdout and stderr separately.
func runCLISeparate(args ...string) (string, string, int) {
	cmd := exec.Command(binaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	exitCode := 0
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}
	return stdout.String(), stderr.String(), exitCode
}

func testdataProject() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", "goproject")
}

func goldenFile(name string) string {
	return filepath.Join(testdataProject(), name+".golden")
}

// normalizeNewlines folds CRLF and CR into LF. Golden files are text, so a
// checkout with core.autocrlf must not change what the comparison sees.
func normalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

// assertGolden runs the CLI with args and compares stdout against the golden
// file of the given name, rewriting the file instead when -update is set.
func assertGolden(t *testing.T, name string, args ...string) {
	t.Helper()

	output, exitCode := runCLIOnStdout(args...)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	path := goldenFile(name)
	if *updateGolden {
		// Store LF only. The markdown templates are embedded from disk, so
		// their line endings follow the checkout and would otherwise leak a
		// platform difference into every golden file.
		if err := os.WriteFile(path, []byte(normalizeNewlines(output)), 0o644); err != nil {
			t.Fatalf("failed to update golden file %s: %v", path, err)
		}
		return
	}

	baseline, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read golden file %s (run 'go test ./apilot-cli/... -update' to create it): %v", path, err)
	}

	got, want := normalizeNewlines(output), normalizeNewlines(string(baseline))
	if got != want {
		t.Errorf("%s output does not match golden file %s.\n"+
			"If the change is intended, refresh the baseline with 'go test ./apilot-cli/... -update'.\n\n%s",
			name, path, diffLines(want, got))
	}
}

// diffLines renders a compact line diff, stopping after maxHunks differing
// lines so a wholesale rewrite does not bury the test output.
func diffLines(want, got string) string {
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")
	limit := len(wantLines)
	if len(gotLines) > limit {
		limit = len(gotLines)
	}

	const maxHunks = 10
	var b strings.Builder
	hunks, differing := 0, 0
	for i := 0; i < limit; i++ {
		var w, g string
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if w == g {
			continue
		}
		differing++
		if hunks < maxHunks {
			fmt.Fprintf(&b, "  line %d:\n    want: %q\n    got:  %q\n", i+1, w, g)
			hunks++
		}
	}
	if hunks < differing {
		fmt.Fprintf(&b, "  ... (%d more differing lines)\n", differing-hunks)
	}
	return b.String()
}

func TestHelpOutput(t *testing.T) {
	output, err := runCLI("--help")
	if err != nil {
		t.Fatalf("--help should not error: %v", err)
	}

	if !strings.Contains(output, "Usage: apilot <source-path> [flags]") {
		t.Error("help output should contain usage line")
	}

	if !strings.Contains(output, "--collector") {
		t.Error("help output should contain --collector flag")
	}

	if !strings.Contains(output, "--formatter") {
		t.Error("help output should contain --formatter flag")
	}

	if !strings.Contains(output, "--no-deps") {
		t.Error("help output should contain --no-deps flag")
	}

	if !strings.Contains(output, "Registered collectors:") {
		t.Error("help output should list registered collectors")
	}

	if !strings.Contains(output, "Registered formatters:") {
		t.Error("help output should list registered formatters")
	}
}

func TestListCollectors(t *testing.T) {
	output, err := runCLI("--list-collectors")
	if err != nil {
		t.Fatalf("--list-collectors should not error: %v", err)
	}

	if !strings.Contains(output, "go:") {
		t.Error("should list go collector")
	}
	if !strings.Contains(output, "java:") {
		t.Error("should list java collector")
	}
	if !strings.Contains(output, "node:") {
		t.Error("should list node collector")
	}
	if !strings.Contains(output, "python:") {
		t.Error("should list python collector")
	}
}

func TestListFormatters(t *testing.T) {
	output, err := runCLI("--list-formatters")
	if err != nil {
		t.Fatalf("--list-formatters should not error: %v", err)
	}

	if !strings.Contains(output, "markdown") {
		t.Error("should list markdown formatter")
	}
	if !strings.Contains(output, "curl") {
		t.Error("should list curl formatter")
	}
	if !strings.Contains(output, "postman") {
		t.Error("should list postman formatter")
	}
}

func TestGoProjectWithMarkdownFormatter(t *testing.T) {
	assertGolden(t, "markdown", testdataProject(), "--collector", "go", "--formatter", "markdown")
}

func TestGoProjectWithDetailedMarkdownFormatter(t *testing.T) {
	assertGolden(t, "markdown-detailed", testdataProject(), "--collector", "go", "--formatter", "markdown", "--format", "detailed")
}

func TestGoProjectWithCurlFormatter(t *testing.T) {
	assertGolden(t, "curl", testdataProject(), "--collector", "go", "--formatter", "curl")
}

func TestGoProjectWithPostmanFormatter(t *testing.T) {
	assertGolden(t, "postman", testdataProject(), "--collector", "go", "--formatter", "postman")
}

func TestAutoDetectCollector(t *testing.T) {
	assertGolden(t, "markdown", testdataProject())
}

func TestOutputToFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "apilot-output-*.md")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	_, exitCode := runCLIWithExitCode(testdataProject(), "--collector", "go", "--output", tmpPath)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0 writing to %s, got %d", tmpPath, exitCode)
	}

	content, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatalf("output file should exist: %v", err)
	}

	// The file must hold the export itself, not merely be created.
	want, err := os.ReadFile(goldenFile("markdown"))
	if err != nil {
		t.Fatalf("failed to read golden file: %v", err)
	}
	if got, want := normalizeNewlines(string(content)), normalizeNewlines(string(want)); got != want {
		t.Errorf("output file does not match golden markdown.\n\n%s", diffLines(want, got))
	}
}

func TestMissingSourcePath(t *testing.T) {
	output, exitCode := runCLIWithExitCode("--collector", "go")
	if exitCode != 1 {
		t.Fatalf("expected exit code 1 for missing source path, got %d, output: %s", exitCode, output)
	}

	if !strings.Contains(output, "source path required") {
		t.Error("output should contain 'source path required' error")
	}
}

func TestNonExistentSourcePath(t *testing.T) {
	output, exitCode := runCLIWithExitCode("/nonexistent/path")
	if exitCode != 1 {
		t.Fatalf("expected exit code 1 for non-existent path, got %d, output: %s", exitCode, output)
	}

	if !strings.Contains(output, "error") {
		t.Error("output should contain error message")
	}
}

// postmanCollection is the subset of the Postman v2.1 schema the assertions
// below need.
type postmanCollection struct {
	Item []postmanNode `json:"item"`
}

type postmanNode struct {
	Name    string        `json:"name"`
	Item    []postmanNode `json:"item"`
	Request *struct {
		Method string `json:"method"`
		URL    struct {
			Raw      string            `json:"raw"`
			Path     []string          `json:"path"`
			Query    []postmanKeyValue `json:"query"`
			Variable []postmanKeyValue `json:"variable"`
		} `json:"url"`
		Body *struct {
			Mode string `json:"mode"`
			Raw  string `json:"raw"`
		} `json:"body"`
	} `json:"request"`
	Response []struct {
		Code int    `json:"code"`
		Body string `json:"body"`
	} `json:"response"`
}

type postmanKeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// flattenPostmanRequests returns every request in the collection, descending
// into folders.
func flattenPostmanRequests(nodes []postmanNode) []postmanNode {
	var out []postmanNode
	for _, n := range nodes {
		if n.Request != nil {
			out = append(out, n)
			continue
		}
		out = append(out, flattenPostmanRequests(n.Item)...)
	}
	return out
}

// TestGoProjectExportedContent asserts the exported content itself rather than
// the absence of a crash. Golden files catch any byte change; these assertions
// exist so that a regression is reported as "field X disappeared" instead of a
// wall of diff, and so that refreshing the goldens cannot silently accept an
// export that lost its fields.
func TestGoProjectExportedContent(t *testing.T) {
	output, exitCode := runCLIOnStdout(testdataProject(), "--collector", "go", "--formatter", "postman")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	var collection postmanCollection
	if err := json.Unmarshal([]byte(output), &collection); err != nil {
		t.Fatalf("postman output should be valid JSON: %v\n%s", err, output)
	}

	requests := flattenPostmanRequests(collection.Item)
	if len(requests) != 5 {
		t.Fatalf("expected 5 exported endpoints, got %d", len(requests))
	}

	byName := make(map[string]postmanNode, len(requests))
	for _, r := range requests {
		byName[r.Name] = r
	}

	// Every endpoint of the fixture must be present with its method and path.
	for _, want := range []struct {
		name   string
		method string
		path   string
	}{
		{"listUsers", "GET", "/users"},
		{"createUser", "POST", "/users"},
		{"getUser", "GET", "/users/:id"},
		{"updateUser", "PUT", "/users/:id"},
		{"deleteUser", "DELETE", "/users/:id"},
	} {
		r, ok := byName[want.name]
		if !ok {
			t.Errorf("endpoint %s missing from export", want.name)
			continue
		}
		if r.Request.Method != want.method {
			t.Errorf("endpoint %s: method = %q, want %q", want.name, r.Request.Method, want.method)
		}
		got := "/" + strings.Join(r.Request.URL.Path, "/")
		if got != want.path {
			t.Errorf("endpoint %s: path = %q, want %q", want.name, got, want.path)
		}
	}

	// Query parameters: name, required flag and default value.
	listUsers := byName["listUsers"]
	if len(listUsers.Request.URL.Query) != 2 {
		t.Fatalf("listUsers: expected 2 query params, got %d", len(listUsers.Request.URL.Query))
	}
	if listUsers.Request.URL.Query[0].Key != "keyword" {
		t.Errorf("listUsers: query[0] = %q, want %q", listUsers.Request.URL.Query[0].Key, "keyword")
	}
	if listUsers.Request.URL.Query[1].Key != "page" || listUsers.Request.URL.Query[1].Value != "1" {
		t.Errorf("listUsers: query[1] = %q=%q, want %q=%q",
			listUsers.Request.URL.Query[1].Key, listUsers.Request.URL.Query[1].Value, "page", "1")
	}

	// Path parameter of GET /users/:id.
	getUser := byName["getUser"]
	if len(getUser.Request.URL.Variable) != 1 || getUser.Request.URL.Variable[0].Key != "id" {
		t.Errorf("getUser: path variables = %+v, want a single %q", getUser.Request.URL.Variable, "id")
	}

	// Request body field names of POST /users.
	createUser := byName["createUser"]
	if createUser.Request.Body == nil {
		t.Fatalf("createUser: request body missing from export")
	}
	for _, field := range []string{"name", "email", "password", "tags"} {
		if !strings.Contains(createUser.Request.Body.Raw, `"`+field+`"`) {
			t.Errorf("createUser: request body is missing field %q: %s", field, createUser.Request.Body.Raw)
		}
	}

	// Response body field names: the User struct behind every 2xx response.
	for _, name := range []string{"createUser", "getUser", "updateUser"} {
		r := byName[name]
		if len(r.Response) == 0 {
			t.Errorf("%s: response missing from export", name)
			continue
		}
		for _, field := range []string{"id", "name", "email", "age"} {
			if !strings.Contains(r.Response[0].Body, `"`+field+`"`) {
				t.Errorf("%s: response body is missing field %q: %s", name, field, r.Response[0].Body)
			}
		}
	}
}

// TestNoDepsFlag verifies the escape hatch: with --no-deps the export is
// byte-identical to the golden baseline (dependency resolution never changes
// endpoint discovery), while stderr confirms the capability was turned off.
func TestNoDepsFlag(t *testing.T) {
	stdout, stderr, exitCode := runCLISeparate(testdataProject(), "--collector", "go", "--formatter", "markdown", "--no-deps")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr: %s", exitCode, stderr)
	}

	baseline, err := os.ReadFile(goldenFile("markdown"))
	if err != nil {
		t.Fatalf("failed to read golden file: %v", err)
	}
	if got, want := normalizeNewlines(stdout), normalizeNewlines(string(baseline)); got != want {
		t.Errorf("--no-deps output does not match golden file.\n\n%s", diffLines(want, got))
	}

	if !strings.Contains(stderr, "dependency type resolution disabled (--no-deps)") {
		t.Errorf("stderr should confirm dependency resolution was disabled, got %q", stderr)
	}
}

func TestVersionFlag(t *testing.T) {
	output, exitCode := runCLIWithExitCode("--version")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, output: %s", exitCode, output)
	}

	if !strings.Contains(output, "apilot") {
		t.Error("version output should contain 'apilot'")
	}
}
