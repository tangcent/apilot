package pycollector

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-python/fastapi"
)

type PythonDependencyResolver struct {
	mu        sync.Mutex
	sourceDir string
	cache     map[string]*collector.ResolvedType
	misses    map[string]bool
	loaded    bool
}

func NewPythonDependencyResolver(sourceDir string) *PythonDependencyResolver {
	return &PythonDependencyResolver{
		sourceDir: sourceDir,
		cache:     make(map[string]*collector.ResolvedType),
		misses:    make(map[string]bool),
	}
}

func (r *PythonDependencyResolver) DetectDependencies(sourceDir string) ([]collector.Dependency, error) {
	return detectPythonDependencies(sourceDir)
}

func (r *PythonDependencyResolver) ResolveType(typeName string) *collector.ResolvedType {
	r.mu.Lock()
	defer r.mu.Unlock()

	if cached, ok := r.cache[typeName]; ok {
		return cached
	}

	if r.misses[typeName] {
		return nil
	}

	if !r.loaded {
		r.loadFromSitePackages()
		r.loaded = true
	}

	rt := r.resolveFromCache(typeName)
	if rt != nil {
		return rt
	}

	r.misses[typeName] = true
	return nil
}

func (r *PythonDependencyResolver) loadFromSitePackages() {
	sitePackagesDir, err := findSitePackages()
	if err != nil || sitePackagesDir == "" {
		return
	}

	entries, err := os.ReadDir(sitePackagesDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pkgDir := filepath.Join(sitePackagesDir, entry.Name())
		if strings.HasPrefix(entry.Name(), "_") || strings.HasSuffix(entry.Name(), ".dist-info") {
			continue
		}

		r.loadPackageTypes(pkgDir)
	}
}

func (r *PythonDependencyResolver) loadPackageTypes(pkgDir string) {
	pyFiles := findPyFiles(pkgDir)
	if len(pyFiles) == 0 {
		return
	}

	for _, f := range pyFiles {
		models, err := fastapi.ExtractPydanticModelsFromFile(f)
		if err != nil {
			continue
		}
		for name, md := range models {
			rt := pydanticModelToResolvedType(md)
			r.cache[name] = rt
		}
	}
}

func (r *PythonDependencyResolver) resolveFromCache(typeName string) *collector.ResolvedType {
	if rt, ok := r.cache[typeName]; ok {
		return rt
	}
	return nil
}

func pydanticModelToResolvedType(md fastapi.PydanticModel) *collector.ResolvedType {
	rt := &collector.ResolvedType{
		Name:           md.Name,
		IsInterface:    false,
	}

	for _, embedded := range md.EmbeddedTypes {
		rt.Interfaces = append(rt.Interfaces, embedded)
	}

	for _, f := range md.Fields {
		rt.Fields = append(rt.Fields, collector.ResolvedField{
			Name:     f.Name,
			Type:     f.Type,
			Required: f.Required,
		})
	}

	return rt
}

func findPyFiles(dir string) []string {
	var files []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == "__pycache__" || name == ".git" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".py") && !strings.HasSuffix(path, "__init__.py") {
			files = append(files, path)
		}
		return nil
	})
	return files
}

func findSitePackages() (string, error) {
	cmd := exec.Command("python3", "-c", "import site; print(site.getsitepackages()[0])")
	output, err := cmd.Output()
	if err != nil {
		cmd = exec.Command("python", "-c", "import site; print(site.getsitepackages()[0])")
		output, err = cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to find site-packages: %w", err)
		}
	}
	return strings.TrimSpace(string(output)), nil
}

type pipPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func detectPythonDependencies(sourceDir string) ([]collector.Dependency, error) {
	reqFile := filepath.Join(sourceDir, "requirements.txt")
	data, err := os.ReadFile(reqFile)
	if err != nil {
		return nil, nil
	}

	var deps []collector.Dependency
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}

		name, version := parseRequirementLine(line)
		deps = append(deps, collector.Dependency{
			Name:    name,
			Version: version,
		})
	}

	return deps, nil
}

func parseRequirementLine(line string) (name, version string) {
	ops := []string{"==", ">=", "<=", "~=", "!=", "~>", "==="}
	for _, op := range ops {
		if idx := strings.Index(line, op); idx != -1 {
			name = strings.TrimSpace(line[:idx])
			rest := strings.TrimSpace(line[idx+len(op):])
			if commaIdx := strings.Index(rest, ","); commaIdx != -1 {
				version = strings.TrimSpace(rest[:commaIdx])
			} else {
				version = rest
			}
			return
		}
	}

	if idx := strings.Index(line, ">"); idx != -1 {
		name = strings.TrimSpace(line[:idx])
		version = strings.TrimSpace(line[idx+1:])
		return
	}
	if idx := strings.Index(line, "<"); idx != -1 {
		name = strings.TrimSpace(line[:idx])
		version = strings.TrimSpace(line[idx+1:])
		return
	}

	name = strings.TrimSpace(line)
	return
}
