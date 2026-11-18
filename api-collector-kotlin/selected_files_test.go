package kotlincollector

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	javacollector "github.com/tangcent/apilot/api-collector-java"
	"github.com/tangcent/apilot/api-collector-java/maven"
)

// The Kotlin project below has one controller per dependency type it returns:
// a file-level export must look up the types of its own files and no others.
var selectedKotlinProject = map[string]string{
	"src/main/kotlin/com/example/files/FilesAlphaController.kt": `package com.example.files

import com.example.lib.Wrapper
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/files/alpha")
class FilesAlphaController {
    @GetMapping("/one")
    fun one(): Wrapper<AlphaDto> = TODO()
}
`,
	// A project type the controller above refers to, declared in a file of its
	// own: the whole project is parsed for it.
	"src/main/kotlin/com/example/files/AlphaDto.kt": `package com.example.files

data class AlphaDto(val alphaName: String, val alphaCount: Int)
`,
	"src/main/kotlin/com/example/files/FilesBetaController.kt": `package com.example.files

import com.example.lib.PageData
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RestController

@RestController
class FilesBetaController {
    @GetMapping("/files/beta/list")
    fun list(): PageData<AlphaDto> = TODO()
}
`,
	"src/main/kotlin/com/example/files/FilesGammaController.kt": `package com.example.files

import com.example.lib.PageQuery
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RestController

@RestController
class FilesGammaController {
    @PostMapping("/files/gamma/search")
    fun search(@RequestBody query: PageQuery): String = TODO()
}
`,
}

// A mixed tree: a Kotlin controller that returns a Java class and a Java
// controller that returns a Kotlin class, each with a type of the project's
// dependencies as the envelope.
var selectedMixedProject = map[string]string{
	"src/main/kotlin/com/example/mixedfiles/FilesKtController.kt": `package com.example.mixedfiles

import com.example.lib.Wrapper
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RestController

@RestController
class FilesKtController {
    @GetMapping("/files/kt")
    fun address(): Wrapper<FilesAddress> = TODO()
}
`,
	// Controllers nobody selects: one of each language, with no dependency type.
	"src/main/kotlin/com/example/mixedfiles/FilesKtOtherController.kt": `package com.example.mixedfiles

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RestController

@RestController
class FilesKtOtherController {
    @GetMapping("/files/kt/other")
    fun other(): FilesThing = TODO()
}
`,
	"src/main/java/com/example/mixedfiles/FilesJavaOtherController.java": `package com.example.mixedfiles;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class FilesJavaOtherController {
    @GetMapping("/files/java/other")
    public FilesAddress other() { return null; }
}
`,
	"src/main/kotlin/com/example/mixedfiles/FilesThing.kt": `package com.example.mixedfiles

data class FilesThing(val thingName: String)
`,
	"src/main/java/com/example/mixedfiles/FilesAddress.java": `package com.example.mixedfiles;

public class FilesAddress {
    private String city;
}
`,
	"src/main/java/com/example/mixedfiles/FilesJavaController.java": `package com.example.mixedfiles;

import com.example.lib.PageData;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class FilesJavaController {
    @GetMapping("/files/java")
    public PageData<FilesThing> things() { return null; }
}
`,
}

// scriptedIndexer stands in for maven-indexer-cli: it answers explore calls
// from librarySources and records the classes it was asked for.
type scriptedIndexer struct {
	mu    sync.Mutex
	asked []string
}

func (s *scriptedIndexer) run(args ...string) ([]byte, error) {
	if len(args) < 3 || args[0] != "--json" || args[1] != "explore" {
		return nil, fmt.Errorf("unexpected indexer call: %v", args)
	}
	class := args[2]
	s.mu.Lock()
	s.asked = append(s.asked, class)
	s.mu.Unlock()

	classes := []map[string]string{}
	if source, ok := librarySources[class]; ok {
		classes = append(classes, map[string]string{
			"className": class, "artifact": "com.example:lib:1.0.0", "source": source, "language": "kotlin",
		})
	}
	return json.Marshal(map[string]any{"classes": classes})
}

// lookups returns the classes asked for so far, sorted.
func (s *scriptedIndexer) lookups() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]string{}, s.asked...)
	sort.Strings(out)
	return out
}

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// collectSelected collects the project in dir with a fresh indexer. It returns
// the endpoints, the error, the collector and the classes looked up.
func collectSelected(t *testing.T, dir string, ctx collector.CollectContext) ([]collector.ApiEndpoint, error, *KotlinCollector, []string) {
	t.Helper()
	dr, err := maven.NewMavenDependencyResolver()
	if err != nil {
		t.Fatal(err)
	}
	indexer := &scriptedIndexer{}
	dr.SetRunner(indexer.run)
	deps := javacollector.NewDependencies(dr)
	defer deps.Close()

	c := New().(*KotlinCollector)
	c.ShareDependencies(deps)
	if ctx.SourceDir == "" {
		ctx.SourceDir = dir
	}
	endpoints, err := c.Collect(ctx)
	return endpoints, err, c, indexer.lookups()
}

func mustCollectSelected(t *testing.T, dir string, ctx collector.CollectContext) ([]collector.ApiEndpoint, *KotlinCollector, []string) {
	t.Helper()
	endpoints, err, c, lookups := collectSelected(t, dir, ctx)
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}
	return endpoints, c, lookups
}

func kotlinPath(dir, file string) string {
	return filepath.Join(dir, "src", "main", "kotlin", "com", "example", "files", file)
}

func mixedPath(dir, language, file string) string {
	return filepath.Join(dir, "src", "main", language, "com", "example", "mixedfiles", file)
}

func folderNames(endpoints []collector.ApiEndpoint) []string {
	var names []string
	for _, ep := range endpoints {
		names = append(names, ep.Folder)
	}
	return names
}

// endpointsOf keeps the endpoints of the given controllers, in order.
func endpointsOf(endpoints []collector.ApiEndpoint, folders ...string) []collector.ApiEndpoint {
	var out []collector.ApiEndpoint
	for _, ep := range endpoints {
		for _, folder := range folders {
			if ep.Folder == folder {
				out = append(out, ep)
			}
		}
	}
	return out
}

func TestDirectoryExportLooksUpEveryDependencyType(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, selectedKotlinProject)

	endpoints, c, lookups := mustCollectSelected(t, dir, collector.CollectContext{})

	if got, want := folderNames(endpoints), []string{"FilesAlphaController", "FilesBetaController", "FilesGammaController"}; !reflect.DeepEqual(got, want) {
		t.Errorf("endpoints of folders %v, want %v", got, want)
	}
	if want := []string{"com.example.lib.PageData", "com.example.lib.PageQuery", "com.example.lib.Wrapper"}; !reflect.DeepEqual(lookups, want) {
		t.Errorf("lookups = %v, want %v", lookups, want)
	}
	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v", unresolved)
	}
}

// A file-level export holds what the directory export holds for the file, and
// looks up the dependency types that file reaches only.
func TestSelectedKotlinFileExtractsOnlyItsControllers(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, selectedKotlinProject)
	all, _, _ := mustCollectSelected(t, dir, collector.CollectContext{})

	for _, tc := range []struct {
		file    string
		folder  string
		lookups []string
	}{
		{"FilesAlphaController.kt", "FilesAlphaController", []string{"com.example.lib.Wrapper"}},
		{"FilesBetaController.kt", "FilesBetaController", []string{"com.example.lib.PageData"}},
		{"FilesGammaController.kt", "FilesGammaController", []string{"com.example.lib.PageQuery"}},
	} {
		t.Run(tc.file, func(t *testing.T) {
			// SourceFile alone selects a file, as it did before SourceFiles.
			endpoints, c, lookups := mustCollectSelected(t, dir, collector.CollectContext{SourceFile: kotlinPath(dir, tc.file)})

			if want := endpointsOf(all, tc.folder); len(want) != 1 || !reflect.DeepEqual(endpoints, want) {
				t.Errorf("endpoints = %v, want the directory export's %v", folderNames(endpoints), folderNames(want))
			}
			if !reflect.DeepEqual(lookups, tc.lookups) {
				t.Errorf("lookups = %v, want %v: only the types the file reaches", lookups, tc.lookups)
			}
			if got := c.Frameworks(); !reflect.DeepEqual(got, []string{"spring-mvc"}) {
				t.Errorf("frameworks = %v, want [spring-mvc]", got)
			}
			if unresolved := c.Unresolved(); len(unresolved) != 0 {
				t.Errorf("unresolved = %v", unresolved)
			}
		})
	}
}

// The type a selected controller refers to may be declared in a file the
// selection does not name: the whole project is parsed for it.
func TestSelectedKotlinFileResolvesTypesOfOtherFiles(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, selectedKotlinProject)

	endpoints, _, _ := mustCollectSelected(t, dir, collector.CollectContext{SourceFiles: []string{kotlinPath(dir, "FilesAlphaController.kt")}})
	if len(endpoints) != 1 {
		t.Fatalf("expected one endpoint, got %v", folderNames(endpoints))
	}
	data := dataSchema(t, &endpoints[0])
	objectField(t, data, "alphaName")
	objectField(t, data, "alphaCount")
}

func TestSeveralSelectedKotlinFilesShareOneCollection(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, selectedKotlinProject)
	all, _, _ := mustCollectSelected(t, dir, collector.CollectContext{})

	// The endpoints come in the order of the directory export, not of the
	// arguments: the same files give the same document however they are named.
	files := []string{kotlinPath(dir, "FilesGammaController.kt"), kotlinPath(dir, "FilesAlphaController.kt")}
	endpoints, _, lookups := mustCollectSelected(t, dir, collector.CollectContext{SourceFiles: files})

	if want := endpointsOf(all, "FilesAlphaController", "FilesGammaController"); !reflect.DeepEqual(endpoints, want) {
		t.Errorf("endpoints = %v, want the directory export's %v", folderNames(endpoints), folderNames(want))
	}
	if want := []string{"com.example.lib.PageQuery", "com.example.lib.Wrapper"}; !reflect.DeepEqual(lookups, want) {
		t.Errorf("lookups = %v, want %v: nothing for the file that was not selected", lookups, want)
	}
}

func TestSelectedKotlinFileThatDeclaresNoControllerLooksUpNothing(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, selectedKotlinProject)

	endpoints, _, lookups := mustCollectSelected(t, dir, collector.CollectContext{SourceFiles: []string{kotlinPath(dir, "AlphaDto.kt")}})
	if len(endpoints) != 0 || len(lookups) != 0 {
		t.Errorf("endpoints = %v, lookups = %v, want neither", folderNames(endpoints), lookups)
	}
}

func TestSelectedKotlinFileIsReadOnceWhenListedTwice(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, selectedKotlinProject)

	file := kotlinPath(dir, "FilesAlphaController.kt")
	endpoints, _, _ := mustCollectSelected(t, dir, collector.CollectContext{SourceFiles: []string{file, file}})
	if got := folderNames(endpoints); !reflect.DeepEqual(got, []string{"FilesAlphaController"}) {
		t.Errorf("endpoints of folders %v, want one", got)
	}
}

// A selected file the project walk skips, such as generated code under a build
// directory, is still read.
func TestSelectedKotlinFileInABuildDirectoryIsRead(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, selectedKotlinProject)
	writeTree(t, dir, map[string]string{
		"build/generated/FilesGeneratedController.kt": `package com.example.generated

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RestController

@RestController
class FilesGeneratedController {
    @GetMapping("/files/generated")
    fun generated(): String = TODO()
}
`,
	})

	all, _, _ := mustCollectSelected(t, dir, collector.CollectContext{})
	if got := folderNames(endpointsOf(all, "FilesGeneratedController")); len(got) != 0 {
		t.Fatalf("a directory export does not read build directories, got %v", got)
	}

	file := filepath.Join(dir, "build", "generated", "FilesGeneratedController.kt")
	endpoints, _, _ := mustCollectSelected(t, dir, collector.CollectContext{SourceFile: file})
	if got := folderNames(endpoints); !reflect.DeepEqual(got, []string{"FilesGeneratedController"}) {
		t.Errorf("endpoints of folders %v, want FilesGeneratedController", got)
	}
}

// A controller the grammar cannot read stops a run that exports it, because
// skipping part of it could drop endpoints silently. It does not stop an export
// of other files, and what the parser says of a file nobody asked for is not
// reported as if it mattered.
func TestUnrelatedBrokenFilesDoNotStopAFileExport(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, selectedKotlinProject)
	writeTree(t, dir, map[string]string{
		"src/main/kotlin/com/example/files/FilesBrokenController.kt": `package com.example.files

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RestController

annotation class FilesMarker(val groups: Array<String> = [])

@RestController
class FilesBrokenController {
    @GetMapping("/files/broken")
    fun get(): String = "ok"
}
`,
		// Not a controller: read in part, with a warning of its own.
		"src/main/kotlin/com/example/files/FilesTypeSize.kt": `package com.example.files

annotation class FilesTypeSize(val groups: Array<String> = [])
`,
	})

	_, err, _, _ := collectSelected(t, dir, collector.CollectContext{})
	if err == nil || !strings.Contains(err.Error(), "FilesBrokenController.kt") {
		t.Fatalf("a directory export must stop at the broken controller, got: %v", err)
	}
	_, err, _, _ = collectSelected(t, dir, collector.CollectContext{SourceFile: kotlinPath(dir, "FilesBrokenController.kt")})
	if err == nil || !strings.Contains(err.Error(), "FilesBrokenController.kt") {
		t.Fatalf("exporting the broken controller itself must stop, got: %v", err)
	}

	endpoints, c, _ := mustCollectSelected(t, dir, collector.CollectContext{SourceFile: kotlinPath(dir, "FilesAlphaController.kt")})
	if got := folderNames(endpoints); !reflect.DeepEqual(got, []string{"FilesAlphaController"}) {
		t.Fatalf("endpoints of folders %v, want FilesAlphaController", got)
	}
	warnings := strings.Join(c.Warnings(), "\n")
	if !strings.Contains(warnings, "FilesBrokenController.kt") || !strings.Contains(warnings, "not selected") {
		t.Errorf("the skipped controller should be named in a warning, got:\n%s", warnings)
	}
	if strings.Contains(warnings, "FilesTypeSize.kt") {
		t.Errorf("a parser warning about an unselected file should not be reported, got:\n%s", warnings)
	}

	// The warning of the same file is reported when the file is selected.
	_, c, _ = mustCollectSelected(t, dir, collector.CollectContext{SourceFiles: []string{
		kotlinPath(dir, "FilesAlphaController.kt"), kotlinPath(dir, "FilesTypeSize.kt"),
	}})
	if !strings.Contains(strings.Join(c.Warnings(), "\n"), "FilesTypeSize.kt") {
		t.Errorf("the parser warning of a selected file should be reported, got:\n%s", strings.Join(c.Warnings(), "\n"))
	}
}

func TestSelectedJavaFileInAKotlinTreeWithoutJavaIsRejected(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, selectedKotlinProject)
	java := filepath.Join(dir, "Stray.java")
	if err := os.WriteFile(java, []byte("class Stray {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	// The Java file is outside the tree the Kotlin collector reads as mixed.
	sub := filepath.Join(dir, "build")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(sub, "Stray.java")
	if err := os.Rename(java, moved); err != nil {
		t.Fatal(err)
	}

	_, err, _, _ := collectSelected(t, dir, collector.CollectContext{SourceFiles: []string{moved}})
	if err == nil || !strings.Contains(err.Error(), "is not a Kotlin source file") {
		t.Fatalf("expected the Java file to be rejected, got: %v", err)
	}
}

func TestSelectedFilesOfAMixedTree(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, selectedMixedProject)
	all, c, lookups := mustCollectSelected(t, dir, collector.CollectContext{})
	if got, want := folderNames(all), []string{"FilesKtController", "FilesKtOtherController", "FilesJavaController", "FilesJavaOtherController"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("endpoints of folders %v, want %v", got, want)
	}
	if want := []string{"com.example.lib.PageData", "com.example.lib.Wrapper"}; !reflect.DeepEqual(lookups, want) {
		t.Errorf("lookups = %v, want %v", lookups, want)
	}
	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Fatalf("unresolved = %v", unresolved)
	}

	kt := mixedPath(dir, "kotlin", "FilesKtController.kt")
	java := mixedPath(dir, "java", "FilesJavaController.java")
	for _, tc := range []struct {
		name    string
		files   []string
		folders []string
		lookups []string
	}{
		// The Java half is parsed as types and extracts nothing: its
		// controller makes no lookup and the Java class stays visible.
		{"a Kotlin file", []string{kt}, []string{"FilesKtController"}, []string{"com.example.lib.Wrapper"}},
		// The Kotlin half extracts nothing, and its class is visible as a type.
		{"a Java file", []string{java}, []string{"FilesJavaController"}, []string{"com.example.lib.PageData"}},
		{"a file of each language", []string{java, kt}, []string{"FilesKtController", "FilesJavaController"}, []string{"com.example.lib.PageData", "com.example.lib.Wrapper"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			endpoints, c, lookups := mustCollectSelected(t, dir, collector.CollectContext{SourceFiles: tc.files})
			if want := endpointsOf(all, tc.folders...); !reflect.DeepEqual(endpoints, want) || len(want) != len(tc.folders) {
				t.Errorf("endpoints = %v, want the directory export's %v", folderNames(endpoints), tc.folders)
			}
			if !reflect.DeepEqual(lookups, tc.lookups) {
				t.Errorf("lookups = %v, want %v", lookups, tc.lookups)
			}
			if unresolved := c.Unresolved(); len(unresolved) != 0 {
				t.Errorf("unresolved = %v: types of the other language must stay visible", unresolved)
			}
		})
	}
}

func TestKotlinCollectorSelectsFiles(t *testing.T) {
	var c collector.Collector = New()
	selector, ok := c.(collector.FileSelector)
	if !ok || !selector.SelectsFiles() {
		t.Fatal("the kotlin collector limits its result to the selected files, and must say so")
	}
}
