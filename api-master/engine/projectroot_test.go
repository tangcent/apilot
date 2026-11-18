package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTree creates the files of a project layout under root. The keys are
// slash-separated paths relative to root, the values the contents.
func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

// pomListing is a pom.xml that lists the given modules.
func pomListing(modules ...string) string {
	var sb strings.Builder
	sb.WriteString("<project>\n  <modules>\n")
	for _, module := range modules {
		sb.WriteString("    <module>" + module + "</module>\n")
	}
	sb.WriteString("  </modules>\n</project>\n")
	return sb.String()
}

// pomWithParent is a pom.xml of a module whose parent sits at relativePath.
func pomWithParent(relativePath string) string {
	return "<project>\n  <parent>\n    <groupId>g</groupId>\n    <artifactId>p</artifactId>\n" +
		"    <relativePath>" + relativePath + "</relativePath>\n  </parent>\n</project>\n"
}

func assertRoot(t *testing.T, dir, want string) {
	t.Helper()
	got, err := findProjectRoot(dir)
	if err != nil {
		t.Fatalf("findProjectRoot(%q) failed: %v", dir, err)
	}
	if got != want {
		t.Errorf("findProjectRoot(%q) = %q, want %q", dir, got, want)
	}
}

func TestFindProjectRoot_MavenModuleSpellings(t *testing.T) {
	for _, module := range []string{
		"user-service",
		"./user-service",
		"user-service/",
		"sub/../user-service",
		"  user-service  ",
	} {
		t.Run(module, func(t *testing.T) {
			root := t.TempDir()
			writeTree(t, root, map[string]string{
				"pom.xml":                    pomListing(module),
				"user-service/pom.xml":       "",
				"user-service/src/Main.java": "",
			})
			assertRoot(t, filepath.Join(root, "user-service", "src"), root)
		})
	}
}

func TestFindProjectRoot_NestedAggregators(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"pom.xml":                          pomListing("services"),
		"services/pom.xml":                 pomListing("user-service", "order-service"),
		"services/user-service/pom.xml":    "",
		"services/order-service/pom.xml":   "",
		"services/user-service/src/A.java": "",
	})
	assertRoot(t, filepath.Join(root, "services", "user-service", "src"), root)
	assertRoot(t, filepath.Join(root, "services", "order-service"), root)
	assertRoot(t, filepath.Join(root, "services"), root)
}

// A module can be listed by a path of several segments, from a directory above
// its parent. The directories in between need not be modules themselves.
func TestFindProjectRoot_NestedModulePath(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"pom.xml":                       pomListing("services/user-service"),
		"services/user-service/pom.xml": "",
	})
	assertRoot(t, filepath.Join(root, "services", "user-service"), root)

	// A pom in between that lists nothing does not hide the aggregator above.
	writeTree(t, root, map[string]string{"services/pom.xml": ""})
	assertRoot(t, filepath.Join(root, "services", "user-service"), root)
}

// An aggregator may sit beside its modules and list them as ../x. It is found
// through the <relativePath> of the module's own parent declaration.
func TestFindProjectRoot_AggregatorBesideItsModules(t *testing.T) {
	for name, relativePath := range map[string]string{
		"pom file":  "../build/pom.xml",
		"directory": "../build",
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeTree(t, root, map[string]string{
				"build/pom.xml":         pomListing("../user-service", "../order-service"),
				"user-service/pom.xml":  pomWithParent(relativePath),
				"order-service/pom.xml": pomWithParent(relativePath),
			})
			assertRoot(t, filepath.Join(root, "user-service"), filepath.Join(root, "build"))
			assertRoot(t, filepath.Join(root, "order-service"), filepath.Join(root, "build"))
		})
	}
}

// A pom beside the module that does not name it as a module is not its
// aggregator, nor is a parent declaration that points at a pom without it.
func TestFindProjectRoot_AggregatorThatDoesNotListTheModuleIsIgnored(t *testing.T) {
	for name, files := range map[string]map[string]string{
		"empty parent pom":       {"pom.xml": ""},
		"parent lists a sibling": {"pom.xml": pomListing("order-service")},
		"prefix of the name":     {"pom.xml": pomListing("user-service-api")},
		"module of another dir":  {"pom.xml": pomListing("other/user-service")},
		"parent without modules": {"pom.xml": "<project><artifactId>p</artifactId></project>"},
		"malformed parent pom":   {"pom.xml": "<project><modules><module>user-service"},
		"relative path elsewhere": {
			"pom.xml":              "",
			"build/pom.xml":        pomListing("order-service"),
			"user-service/pom.xml": pomWithParent("../build/pom.xml"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeTree(t, root, map[string]string{"user-service/pom.xml": "", "user-service/src/Main.java": ""})
			writeTree(t, root, files)
			module := filepath.Join(root, "user-service")
			assertRoot(t, filepath.Join(module, "src"), module)
		})
	}
}

func TestFindProjectRoot_MavenModuleInProfile(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"pom.xml": `<project>
  <modules><module>core</module></modules>
  <profiles>
    <profile>
      <id>web</id>
      <modules><module>user-service</module></modules>
    </profile>
  </profiles>
</project>`,
		"core/pom.xml":         "",
		"user-service/pom.xml": "",
	})
	assertRoot(t, filepath.Join(root, "user-service"), root)
	assertRoot(t, filepath.Join(root, "core"), root)
}

func TestFindProjectRoot_MavenModuleThatCannotBeResolvedIsSkipped(t *testing.T) {
	for name, pom := range map[string]string{
		"commented out": "<project><modules><!-- <module>user-service</module> --></modules></project>",
		"property":      pomListing("${service.dir}"),
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeTree(t, root, map[string]string{
				"pom.xml":              pom,
				"user-service/pom.xml": "",
			})
			assertRoot(t, filepath.Join(root, "user-service"), filepath.Join(root, "user-service"))
		})
	}
}

// A pom as Maven tooling writes it: a BOM, an XML declaration, a namespace and
// an encoding other than UTF-8.
func TestFindProjectRoot_MavenPomAsToolsWriteIt(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"pom.xml": "\xef\xbb\xbf<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?>\n" +
			"<project xmlns=\"http://maven.apache.org/POM/4.0.0\"\n" +
			"         xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\"\n" +
			"         xsi:schemaLocation=\"http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd\">\n" +
			"  <modelVersion>4.0.0</modelVersion>\n" +
			"  <name>caf\xe9 &amp; co</name>\n" +
			"  <modules>\n    <module>user-service</module>\n  </modules>\n</project>\n",
		"user-service/pom.xml": "",
	})
	assertRoot(t, filepath.Join(root, "user-service"), root)
}

func TestFindProjectRoot_GradleSettings(t *testing.T) {
	for _, settings := range []string{"settings.gradle", "settings.gradle.kts"} {
		for _, build := range []string{"build.gradle", "build.gradle.kts"} {
			t.Run(settings+" "+build, func(t *testing.T) {
				root := t.TempDir()
				writeTree(t, root, map[string]string{
					settings:                  "include(':app', ':libs:base')",
					build:                     "",
					"app/" + build:            "",
					"app/src/Main.java":       "",
					"libs/base/" + build:      "",
					"libs/base/src/Base.java": "",
				})
				assertRoot(t, filepath.Join(root, "app", "src"), root)
				assertRoot(t, filepath.Join(root, "libs", "base", "src"), root)
				assertRoot(t, filepath.Join(root, "app"), root)
			})
		}
	}
}

// The root of a Gradle build is where Gradle finds its settings file, even when
// a project in between has a build script of its own.
func TestFindProjectRoot_GradleSettingsAboveAnIntermediateProject(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"settings.gradle":      "include 'a', 'a:b'",
		"build.gradle":         "",
		"a/build.gradle":       "",
		"a/b/build.gradle.kts": "",
		"a/b/src/Main.java":    "",
	})
	assertRoot(t, filepath.Join(root, "a", "b", "src"), root)
}

func TestFindProjectRoot_GradleStandaloneKeepsItsOwnRoot(t *testing.T) {
	t.Run("own settings file", func(t *testing.T) {
		root := t.TempDir()
		writeTree(t, root, map[string]string{
			"settings.gradle":          "include 'app'",
			"build.gradle":             "",
			"tool/settings.gradle.kts": "",
			"tool/build.gradle.kts":    "",
			"tool/src/Main.kt":         "",
		})
		assertRoot(t, filepath.Join(root, "tool", "src"), filepath.Join(root, "tool"))
	})
	t.Run("no settings file", func(t *testing.T) {
		root := t.TempDir()
		writeTree(t, root, map[string]string{
			"build.gradle":      "",
			"app/build.gradle":  "",
			"app/src/Main.java": "",
		})
		assertRoot(t, filepath.Join(root, "app", "src"), filepath.Join(root, "app"))
	})
	t.Run("settings file in a directory without a build file", func(t *testing.T) {
		root := t.TempDir()
		writeTree(t, root, map[string]string{
			"settings.gradle":   "include 'app'",
			"app/build.gradle":  "",
			"app/src/Main.java": "",
		})
		assertRoot(t, filepath.Join(root, "app", "src"), filepath.Join(root, "app"))
	})
}

// Only Maven and Gradle declare modules. A Go or Node project nested in a
// directory with a settings file keeps its own root.
func TestFindProjectRoot_OtherProjectsKeepTheirRoot(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"pom.xml":          pomListing("web"),
		"settings.gradle":  "",
		"build.gradle":     "",
		"web/package.json": "{}",
		"web/src/index.js": "",
		"svc/go.mod":       "module svc",
		"svc/main.go":      "",
	})
	assertRoot(t, filepath.Join(root, "web", "src"), filepath.Join(root, "web"))
	assertRoot(t, filepath.Join(root, "svc"), filepath.Join(root, "svc"))
}

// Two aggregators that name each other as parent do not send the search round
// in circles.
func TestFindProjectRoot_AggregatorsThatListEachOtherTerminate(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"a/pom.xml": "<project><parent><relativePath>../b/pom.xml</relativePath></parent>" +
			"<modules><module>../b</module></modules></project>",
		"b/pom.xml": "<project><parent><relativePath>../a/pom.xml</relativePath></parent>" +
			"<modules><module>../a</module></modules></project>",
	})
	got, err := findProjectRoot(filepath.Join(root, "a"))
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(root, "a") && got != filepath.Join(root, "b") {
		t.Errorf("findProjectRoot = %q, want one of the two aggregators", got)
	}
}

// Files of different modules of one repository resolve to the aggregator they
// share, so they can be exported together.
func TestResolveSources_FilesOfDifferentModulesShareTheAggregator(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"pom.xml":                                pomListing("user-service", "order-service"),
		"user-service/pom.xml":                   "",
		"user-service/src/UserController.java":   "class UserController {}",
		"order-service/pom.xml":                  "",
		"order-service/src/OrderController.java": "class OrderController {}",
	})
	user := filepath.Join(root, "user-service", "src", "UserController.java")
	order := filepath.Join(root, "order-service", "src", "OrderController.java")

	target, err := resolveSources([]string{user, order}, "")
	if err != nil {
		t.Fatalf("resolveSources failed: %v", err)
	}
	if target.dir != root {
		t.Errorf("dir = %q, want the shared aggregator %q", target.dir, root)
	}
	if len(target.files) != 2 {
		t.Errorf("files = %v, want both", target.files)
	}

	// The aggregator is the same whichever file is named first.
	target, err = resolveSources([]string{order, user}, "")
	if err != nil || target.dir != root {
		t.Errorf("dir = %q, err = %v, want %q", target.dir, err, root)
	}
}

// A repository whose modules are not listed by the build above them is not one
// project: each module is its own, as before.
func TestResolveSources_FilesOfModulesThatNoBuildListsAreDifferentProjects(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"pom.xml":                                pomListing("billing"),
		"user-service/pom.xml":                   "",
		"user-service/src/UserController.java":   "class UserController {}",
		"order-service/pom.xml":                  "",
		"order-service/src/OrderController.java": "class OrderController {}",
	})
	_, err := resolveSources([]string{
		filepath.Join(root, "user-service", "src", "UserController.java"),
		filepath.Join(root, "order-service", "src", "OrderController.java"),
	}, "")
	if err == nil || !strings.Contains(err.Error(), "different projects") {
		t.Fatalf("expected different projects, got: %v", err)
	}
}

// --project-root still overrides the detection, and a directory export keeps
// the directory it was given.
func TestResolveSources_ProjectRootAndDirectoryExportAreNotLifted(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"pom.xml":                              pomListing("user-service"),
		"user-service/pom.xml":                 "",
		"user-service/src/UserController.java": "class UserController {}",
	})
	module := filepath.Join(root, "user-service")
	file := filepath.Join(module, "src", "UserController.java")

	target, err := resolveSources([]string{file}, module)
	if err != nil || target.dir != module {
		t.Errorf("dir = %q, err = %v, want the --project-root %q", target.dir, err, module)
	}

	target, err = resolveSources([]string{module}, "")
	if err != nil || target.dir != module || len(target.files) != 0 {
		t.Errorf("dir = %q, files = %v, err = %v, want the given directory %q", target.dir, target.files, err, module)
	}
}
