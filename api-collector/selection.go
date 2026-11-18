package collector

import (
	"path/filepath"
	"runtime"
	"strings"
)

// SelectedFiles returns the source files a file-level export names: SourceFiles
// when it is set, else SourceFile, else nil for an export of all of SourceDir.
// Collectors read the selection through it, so a caller that only sets
// SourceFile keeps working.
func (c CollectContext) SelectedFiles() []string {
	if len(c.SourceFiles) > 0 {
		return c.SourceFiles
	}
	if c.SourceFile != "" {
		return []string{c.SourceFile}
	}
	return nil
}

// FileSelection answers whether a source file is one a file-level export
// named. A collector parses the whole project, which supplies the types the
// selected files refer to, but builds and resolves endpoints only for the
// declarations the selection contains.
//
// A nil *FileSelection stands for an export of the whole project: Contains
// accepts every file and Unmatched reports nothing, so a framework parser can
// take one as an optional setting.
type FileSelection struct {
	files []string
	keys  map[string]bool
}

// NewFileSelection returns the selection of paths, or nil when paths is empty.
// Paths are compared as absolute, cleaned paths, and case-insensitively on the
// platforms whose file systems usually are, so the spelling the user typed
// matches the path a directory walk reports.
func NewFileSelection(paths []string) *FileSelection {
	if len(paths) == 0 {
		return nil
	}
	s := &FileSelection{keys: make(map[string]bool, len(paths))}
	for _, path := range paths {
		if path == "" {
			continue
		}
		key := selectionKey(path)
		if s.keys[key] {
			continue
		}
		s.keys[key] = true
		s.files = append(s.files, path)
	}
	return s
}

// FileSelection returns the selection the context names, or nil when it names
// no file.
func (c CollectContext) FileSelection() *FileSelection {
	return NewFileSelection(c.SelectedFiles())
}

// Contains reports whether path is a selected file. A nil selection contains
// every file; any selection contains no file without a path, such as a type
// read from a dependency.
func (s *FileSelection) Contains(path string) bool {
	if s == nil {
		return true
	}
	if path == "" {
		return false
	}
	return s.keys[selectionKey(path)]
}

// Unmatched returns the selected files that none of paths names, in the order
// they were selected: the files a parse of the project did not reach, such as
// one under a directory the walk skips. A nil selection has none.
func (s *FileSelection) Unmatched(paths []string) []string {
	if s == nil {
		return nil
	}
	reached := make(map[string]bool, len(paths))
	for _, path := range paths {
		if path != "" {
			reached[selectionKey(path)] = true
		}
	}
	var missing []string
	for _, file := range s.files {
		if !reached[selectionKey(file)] {
			missing = append(missing, file)
		}
	}
	return missing
}

func selectionKey(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		path = strings.ToLower(path)
	}
	return path
}

// FileSelector is an optional interface implemented by collectors that honour
// the files a CollectContext selects: they return only the endpoints declared
// in those files, and spend no type resolution on any other.
//
// The engine filters the endpoints of every other collector by file name after
// the fact, which is all a collector that cannot tell where an endpoint was
// declared can offer. It trusts the result of a FileSelector as it is, because
// the file name is only a guess at the controller declared in it: a file may
// declare a controller named otherwise, or several.
//
// Like UnresolvedReporter it is deliberately not part of Collector: subprocess
// plugins cannot say so across the process boundary, so the engine must
// type-assert.
type FileSelector interface {
	// SelectsFiles reports whether Collect limits its result to the files of
	// CollectContext.SelectedFiles.
	SelectsFiles() bool
}
