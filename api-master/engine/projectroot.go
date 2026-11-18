package engine

import (
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	// gradleBuildFiles are the build scripts of a Gradle project.
	gradleBuildFiles = []string{"build.gradle", "build.gradle.kts"}
	// gradleSettingsFiles are the files that make a directory the root of a
	// Gradle build.
	gradleSettingsFiles = []string{"settings.gradle", "settings.gradle.kts"}
)

// topmostAggregator lifts dir, the nearest directory with a build file, to the
// root of the multi-module build it is a module of. It climbs for as long as
// the build above names the current directory as one of its modules, and stops
// at the topmost such aggregator. A project that no build above lists keeps its
// own directory.
//
// The two build tools are told apart by what they declare:
//   - Maven: a pom.xml whose <modules>, in the project or in any profile, lists
//     the directory. The pom is looked for in every ancestor, because a module
//     may be listed by a nested path (a/b) from above its parent directory, and
//     in the place the module's own <parent><relativePath> names, because an
//     aggregator may sit beside its modules and list them as ../x.
//   - Gradle: the settings.gradle or settings.gradle.kts at or above the
//     directory, the way Gradle itself finds the root of a build, without
//     reading its include statements. The directory holding it must also have
//     a build file, since the collector is chosen from that.
func topmostAggregator(dir string) string {
	visited := map[string]bool{}
	for !visited[dir] {
		visited[dir] = true
		next := mavenAggregator(dir)
		if next == "" {
			next = gradleRoot(dir)
		}
		if next == "" {
			break
		}
		dir = next
	}
	return dir
}

// pomFile holds the parts of a pom.xml that decide which directory aggregates
// which.
type pomFile struct {
	Parent struct {
		RelativePath *string `xml:"relativePath"`
	} `xml:"parent"`
	Modules  []string `xml:"modules>module"`
	Profiles []struct {
		Modules []string `xml:"modules>module"`
	} `xml:"profiles>profile"`
}

// readPom parses the pom.xml at path. A pom that is missing or cannot be read
// as XML is returned as far as it was read, usually empty, so that it lists no
// modules and the search for an aggregator moves on.
func readPom(path string) pomFile {
	var pom pomFile
	data, err := os.ReadFile(path)
	if err != nil {
		return pom
	}
	decoder := xml.NewDecoder(bytes.NewReader(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))))
	decoder.Strict = false
	decoder.CharsetReader = latin1ToUTF8
	_ = decoder.Decode(&pom)
	return pom
}

// latin1ToUTF8 lets a pom that declares an encoding other than UTF-8 be read.
// Only the markup and the module paths matter here, and those are ASCII, which
// every encoding a pom is likely to declare shares, so each byte is taken as
// one character instead of decoding the text properly.
func latin1ToUTF8(_ string, input io.Reader) (io.Reader, error) {
	raw, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}
	runes := make([]rune, len(raw))
	for i, b := range raw {
		runes[i] = rune(b)
	}
	return strings.NewReader(string(runes)), nil
}

// allModules returns the modules the pom lists, those of its profiles included.
func (p pomFile) allModules() []string {
	modules := append([]string(nil), p.Modules...)
	for _, profile := range p.Profiles {
		modules = append(modules, profile.Modules...)
	}
	return modules
}

// mavenAggregator returns the directory whose pom.xml lists dir as a module, or
// "" when dir has no pom.xml or no pom lists it.
func mavenAggregator(dir string) string {
	ownPom := filepath.Join(dir, "pom.xml")
	if !fileExists(ownPom) {
		return ""
	}

	var candidates []string
	for _, ancestor := range ancestors(dir) {
		candidates = append(candidates, filepath.Join(ancestor, "pom.xml"))
	}
	if relativePath := readPom(ownPom).Parent.RelativePath; relativePath != nil {
		if rel := strings.TrimSpace(*relativePath); rel != "" {
			parent := filepath.Join(dir, filepath.FromSlash(rel))
			if info, err := os.Stat(parent); err == nil && info.IsDir() {
				parent = filepath.Join(parent, "pom.xml")
			}
			candidates = append(candidates, parent)
		}
	}

	for _, pom := range candidates {
		if pomListsModule(pom, dir) {
			return filepath.Dir(pom)
		}
	}
	return ""
}

// pomListsModule reports whether the pom at pomPath lists dir as a module. A
// module path is relative to the directory of the pom and may carry . and ..
// segments, so the two are compared once cleaned. A path built from a property
// cannot be resolved without Maven and is skipped.
func pomListsModule(pomPath, dir string) bool {
	base := filepath.Dir(pomPath)
	for _, module := range readPom(pomPath).allModules() {
		module = strings.TrimSpace(module)
		if module == "" || strings.Contains(module, "${") {
			continue
		}
		module = filepath.FromSlash(strings.ReplaceAll(module, `\`, "/"))
		if samePath(filepath.Join(base, module), dir) {
			return true
		}
	}
	return false
}

// gradleRoot returns the root of the Gradle build dir belongs to: the nearest
// directory at or above dir that holds a settings file. It returns "" when dir
// is not a Gradle project, is itself that directory, or the settings file sits
// in a directory with no build file.
func gradleRoot(dir string) string {
	if !hasAnyFile(dir, gradleBuildFiles) && !hasAnyFile(dir, gradleSettingsFiles) {
		return ""
	}
	for current := dir; ; {
		if hasAnyFile(current, gradleSettingsFiles) {
			if current == dir || !hasAnyFile(current, gradleBuildFiles) {
				return ""
			}
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

// ancestors returns the directories above dir, the nearest first.
func ancestors(dir string) []string {
	var out []string
	for current := dir; ; {
		parent := filepath.Dir(current)
		if parent == current {
			return out
		}
		out = append(out, parent)
		current = parent
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func hasAnyFile(dir string, names []string) bool {
	for _, name := range names {
		if fileExists(filepath.Join(dir, name)) {
			return true
		}
	}
	return false
}

// samePath reports whether two paths name the same directory: the same once
// cleaned, or the same file system object, which covers a different letter
// case on a case-insensitive file system and a path that crosses a symlink.
func samePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if a == b {
		return true
	}
	infoA, errA := os.Stat(a)
	infoB, errB := os.Stat(b)
	return errA == nil && errB == nil && os.SameFile(infoA, infoB)
}
