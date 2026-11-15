package maven

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// runner runs the indexer CLI with the given arguments and returns its
// standard output. Tests replace it to answer without the CLI.
type runner func(args ...string) ([]byte, error)

func runCLI(args ...string) ([]byte, error) {
	out, err := exec.Command(cliName, args...).Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
		// Progress goes to stderr too; the error is its last line.
		return out, fmt.Errorf("%w: %s", err, lastLine(string(exitErr.Stderr)))
	}
	return out, err
}

func lastLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.LastIndexAny(s, "\r\n"); i >= 0 {
		return s[i+1:]
	}
	return s
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		return s[:i]
	}
	return s
}

// ProjectContext is the indexer's view of a project's dependencies, from
// `maven-indexer-cli project-context`.
type ProjectContext struct {
	// Kind is the build system the indexer found: maven, gradle or none.
	Kind string
	// Tree says which dependency view is known: "resolved" (the full tree,
	// transitives included, from `mvn dependency:tree` or `gradle
	// dependencies`), "declared" (the dependencies the build file lists
	// itself) or "none".
	Tree string
	// Declared and Resolved count the dependencies of each view.
	Declared int
	Resolved int
	// ResolveError tells why the resolved view is missing, when it failed.
	ResolveError string
}

// HasDependencies reports whether the indexer knows any dependency of the
// project.
func (c ProjectContext) HasDependencies() bool {
	return c.Declared+c.Resolved > 0
}

type projectContextJSON struct {
	Kind     string            `json:"kind"`
	Tree     string            `json:"tree"`
	Declared []json.RawMessage `json:"declared"`
	Resolved *struct {
		Coordinates []json.RawMessage `json:"coordinates"`
	} `json:"resolved"`
	ResolveError string `json:"resolveError"`
}

// LoadProjectContext asks the indexer for the dependency tree of the project
// at projectPath. It needs maven-indexer-cli 1.2.0 or later.
//
// On a project the indexer has not seen, the first answer is the declared
// view: the indexer resolves the full tree before it exits and caches it per
// build file, so a second call reads it. Every later class lookup then pins
// versions against the full tree, which is what a multi-module project needs:
// its root build file usually declares nothing itself.
func LoadProjectContext(projectPath string) (ProjectContext, error) {
	return loadProjectContext(runCLI, projectPath)
}

func loadProjectContext(run runner, projectPath string) (ProjectContext, error) {
	ctx, err := queryProjectContext(run, projectPath)
	if err != nil {
		return ctx, err
	}
	if ctx.Kind != "none" && ctx.Tree == "declared" && ctx.ResolveError == "" {
		if again, err := queryProjectContext(run, projectPath); err == nil {
			ctx = again
		}
	}
	return ctx, nil
}

func queryProjectContext(run runner, projectPath string) (ProjectContext, error) {
	out, err := run("--json", "project-context", projectPath)
	if err != nil {
		return ProjectContext{}, fmt.Errorf("%s project-context: %w", cliName, err)
	}
	var raw projectContextJSON
	if err := json.Unmarshal(out, &raw); err != nil {
		return ProjectContext{}, fmt.Errorf("%s project-context: unexpected output: %w", cliName, err)
	}
	ctx := ProjectContext{
		Kind:         raw.Kind,
		Tree:         raw.Tree,
		Declared:     len(raw.Declared),
		ResolveError: firstLine(raw.ResolveError),
	}
	if raw.Resolved != nil {
		ctx.Resolved = len(raw.Resolved.Coordinates)
	}
	return ctx, nil
}

// DeclaresModules reports whether the build file in sourceDir aggregates
// modules: a pom.xml with <modules>, or a Gradle settings file. Such a root
// usually declares no dependency itself, while its modules do.
func DeclaresModules(sourceDir string) bool {
	for _, f := range []string{"settings.gradle", "settings.gradle.kts"} {
		if _, err := os.Stat(filepath.Join(sourceDir, f)); err == nil {
			return true
		}
	}
	data, err := os.ReadFile(filepath.Join(sourceDir, "pom.xml"))
	if err != nil {
		return false
	}
	var pom struct {
		Modules struct {
			Module []string `xml:"module"`
		} `xml:"modules"`
	}
	if err := xml.Unmarshal(data, &pom); err != nil {
		return false
	}
	return len(pom.Modules.Module) > 0
}
