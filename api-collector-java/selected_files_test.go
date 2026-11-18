package javacollector

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
	"github.com/tangcent/apilot/api-collector-java/maven"
	model "github.com/tangcent/apilot/api-model"
)

// The project below has one controller per framework, each of which returns a
// type of the project's dependencies: a file-level export must look up the
// types of its own files and no others.
var selectedFilesProject = map[string]string{
	"src/main/java/com/example/files/FilesAlphaController.java": `package com.example.files;

import com.example.lib.Wrapper;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/files/alpha")
public class FilesAlphaController {
    @GetMapping("/one")
    public Wrapper<AlphaDto> one() { return null; }
}
`,
	// A project type the controller above refers to, declared in a file of its
	// own: the whole project is parsed for it.
	"src/main/java/com/example/files/AlphaDto.java": `package com.example.files;

public class AlphaDto {
    private String alphaName;
    private int alphaCount;
}
`,
	"src/main/java/com/example/files/FilesBetaResource.java": `package com.example.files;

import com.example.lib.PageData;
import javax.ws.rs.GET;
import javax.ws.rs.Path;

@Path("/files/beta")
public class FilesBetaResource {
    @GET
    @Path("/list")
    public PageData<AlphaDto> list() { return null; }
}
`,
	"src/main/java/com/example/files/FilesGammaClient.java": `package com.example.files;

import com.example.lib.Ticket;
import org.springframework.cloud.openfeign.FeignClient;
import org.springframework.web.bind.annotation.GetMapping;

@FeignClient(name = "gamma")
public interface FilesGammaClient {
    @GetMapping("/files/gamma/ticket")
    Ticket ticket();
}
`,
}

// librarySources are the classes of the project's dependencies, as the indexer
// returns them.
var librarySources = map[string]string{
	"com.example.lib.Wrapper": `package com.example.lib;

public class Wrapper<T> {
    private int code;
    private T data;
}
`,
	"com.example.lib.PageData": `package com.example.lib;

import java.util.List;

public class PageData<T> {
    private long total;
    private List<T> rows;
}
`,
	"com.example.lib.Ticket": `package com.example.lib;

public class Ticket {
    private String ticketNo;
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
			"className": class, "artifact": "com.example:lib:1.0.0", "source": source, "language": "java",
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

func scriptedDependencies(t *testing.T) (*Dependencies, *scriptedIndexer) {
	t.Helper()
	dr, err := maven.NewMavenDependencyResolver()
	if err != nil {
		t.Fatal(err)
	}
	indexer := &scriptedIndexer{}
	dr.SetRunner(indexer.run)
	deps := NewDependencies(dr)
	t.Cleanup(deps.Close)
	return deps, indexer
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

// collectSelected collects the project in dir with a fresh indexer and returns
// the endpoints, the collector and the classes looked up. With no ctx.SourceDir
// the project directory is used.
func collectSelected(t *testing.T, dir string, ctx collector.CollectContext) ([]collector.ApiEndpoint, *JavaCollector, []string) {
	t.Helper()
	deps, indexer := scriptedDependencies(t)
	c := New().(*JavaCollector)
	c.ShareDependencies(deps)
	if ctx.SourceDir == "" {
		ctx.SourceDir = dir
	}
	endpoints, err := c.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}
	return endpoints, c, indexer.lookups()
}

func selectedPath(dir, file string) string {
	return filepath.Join(dir, "src", "main", "java", "com", "example", "files", file)
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
	writeTree(t, dir, selectedFilesProject)

	endpoints, c, lookups := collectSelected(t, dir, collector.CollectContext{})

	if got, want := folderNames(endpoints), []string{"FilesAlphaController", "FilesBetaResource", "FilesGammaClient"}; !reflect.DeepEqual(got, want) {
		t.Errorf("endpoints of folders %v, want %v", got, want)
	}
	if want := []string{"com.example.lib.PageData", "com.example.lib.Ticket", "com.example.lib.Wrapper"}; !reflect.DeepEqual(lookups, want) {
		t.Errorf("lookups = %v, want %v", lookups, want)
	}
	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v", unresolved)
	}
	if got, want := c.Frameworks(), []string{"spring-mvc", "jaxrs", "feign"}; !reflect.DeepEqual(got, want) {
		t.Errorf("frameworks = %v, want %v", got, want)
	}
}

// A file-level export holds what the directory export holds for the file, and
// looks up the dependency types that file reaches only.
func TestSelectedFileExtractsOnlyItsDeclarations(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, selectedFilesProject)
	all, _, _ := collectSelected(t, dir, collector.CollectContext{})

	for _, tc := range []struct {
		file       string
		folder     string
		framework  string
		lookups    []string
		fieldOfDto string
	}{
		{"FilesAlphaController.java", "FilesAlphaController", "spring-mvc", []string{"com.example.lib.Wrapper"}, "alphaName"},
		{"FilesBetaResource.java", "FilesBetaResource", "jaxrs", []string{"com.example.lib.PageData"}, ""},
		{"FilesGammaClient.java", "FilesGammaClient", "feign", []string{"com.example.lib.Ticket"}, ""},
	} {
		t.Run(tc.file, func(t *testing.T) {
			// SourceFile alone selects a file, as it did before SourceFiles.
			endpoints, c, lookups := collectSelected(t, dir, collector.CollectContext{SourceFile: selectedPath(dir, tc.file)})

			if want := endpointsOf(all, tc.folder); len(want) != 1 || !reflect.DeepEqual(endpoints, want) {
				t.Errorf("endpoints = %+v, want the directory export's %+v", endpoints, want)
			}
			if !reflect.DeepEqual(lookups, tc.lookups) {
				t.Errorf("lookups = %v, want %v: only the types the file reaches", lookups, tc.lookups)
			}
			if got := c.Frameworks(); !reflect.DeepEqual(got, []string{tc.framework}) {
				t.Errorf("frameworks = %v, want [%s]", got, tc.framework)
			}
			if unresolved := c.Unresolved(); len(unresolved) != 0 {
				t.Errorf("unresolved = %v", unresolved)
			}
			if tc.fieldOfDto != "" {
				// The type the controller refers to is declared in another
				// file of the project, which the selection does not name.
				body := endpoints[0].Response.Body
				data, ok := body.Fields["data"]
				if !ok || data.Model == nil || data.Model.Fields[tc.fieldOfDto] == nil {
					t.Errorf("the response should expand the project type with %s: %s", tc.fieldOfDto, describe(body))
				}
			}
		})
	}
}

func describe(object *model.ObjectModel) string {
	data, _ := json.Marshal(object)
	return string(data)
}

func TestSeveralSelectedFilesShareOneCollection(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, selectedFilesProject)
	all, _, _ := collectSelected(t, dir, collector.CollectContext{})

	files := []string{selectedPath(dir, "FilesGammaClient.java"), selectedPath(dir, "FilesAlphaController.java")}
	endpoints, c, lookups := collectSelected(t, dir, collector.CollectContext{SourceFiles: files})

	if want := endpointsOf(all, "FilesAlphaController", "FilesGammaClient"); !reflect.DeepEqual(endpoints, want) {
		t.Errorf("endpoints = %v, want the directory export's %v", folderNames(endpoints), folderNames(want))
	}
	if want := []string{"com.example.lib.Ticket", "com.example.lib.Wrapper"}; !reflect.DeepEqual(lookups, want) {
		t.Errorf("lookups = %v, want %v: nothing for the file that was not selected", lookups, want)
	}
	if got, want := c.Frameworks(), []string{"spring-mvc", "feign"}; !reflect.DeepEqual(got, want) {
		t.Errorf("frameworks = %v, want %v", got, want)
	}
}

func TestSelectedFileThatDeclaresNoEndpointLooksUpNothing(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, selectedFilesProject)

	endpoints, _, lookups := collectSelected(t, dir, collector.CollectContext{SourceFiles: []string{selectedPath(dir, "AlphaDto.java")}})
	if len(endpoints) != 0 || len(lookups) != 0 {
		t.Errorf("endpoints = %v, lookups = %v, want neither", folderNames(endpoints), lookups)
	}
}

func TestSelectedFileIsReadOnceWhenListedTwice(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, selectedFilesProject)

	file := selectedPath(dir, "FilesAlphaController.java")
	endpoints, _, _ := collectSelected(t, dir, collector.CollectContext{SourceFiles: []string{file, file}})
	if got := folderNames(endpoints); !reflect.DeepEqual(got, []string{"FilesAlphaController"}) {
		t.Errorf("endpoints of folders %v, want one", got)
	}
}

// A file the project walk does not reach, such as one outside the project
// directory, is still read.
func TestSelectedFileOutsideTheProjectIsRead(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, selectedFilesProject)
	elsewhere := t.TempDir()
	writeTree(t, elsewhere, map[string]string{
		"FilesOutsideController.java": `package com.example.outside;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class FilesOutsideController {
    @GetMapping("/files/outside")
    public String outside() { return null; }
}
`,
	})

	file := filepath.Join(elsewhere, "FilesOutsideController.java")
	endpoints, _, _ := collectSelected(t, dir, collector.CollectContext{SourceFile: file})
	if got := folderNames(endpoints); !reflect.DeepEqual(got, []string{"FilesOutsideController"}) {
		t.Errorf("endpoints of folders %v, want FilesOutsideController", got)
	}
}

func TestSelectedKotlinFileIsRejected(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, selectedFilesProject)
	kt := filepath.Join(dir, "FilesKtController.kt")
	if err := os.WriteFile(kt, []byte("class FilesKtController"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := New()
	_, err := c.Collect(collector.CollectContext{
		SourceDir:   dir,
		NoDeps:      true,
		SourceFiles: []string{selectedPath(dir, "FilesAlphaController.java"), kt},
	})
	if err == nil || !strings.Contains(err.Error(), kt) || !strings.Contains(err.Error(), "Kotlin") {
		t.Fatalf("expected the Kotlin file to be named in an error, got: %v", err)
	}
}

func TestJavaCollectorSelectsFiles(t *testing.T) {
	var c collector.Collector = New()
	selector, ok := c.(collector.FileSelector)
	if !ok || !selector.SelectsFiles() {
		t.Fatal("the java collector limits its result to the selected files, and must say so")
	}
}
