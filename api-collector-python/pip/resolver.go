package pip

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-python/fastapi"
)

type PipTypeResolver struct {
	mu         sync.Mutex
	sourceDir  string
	cache      map[string]*collector.ResolvedType
	misses     map[string]bool
	deps       []string
	loadedPkgs map[string]bool
	depsParsed bool
}

func NewPipTypeResolver(sourceDir string) *PipTypeResolver {
	return &PipTypeResolver{
		sourceDir:  sourceDir,
		cache:      make(map[string]*collector.ResolvedType),
		misses:     make(map[string]bool),
		loadedPkgs: make(map[string]bool),
	}
}

func (r *PipTypeResolver) DetectDependencies(sourceDir string) ([]collector.Dependency, error) {
	return DetectPipDependencies(sourceDir)
}

func (r *PipTypeResolver) ResolveType(typeName string) *collector.ResolvedType {
	r.mu.Lock()
	defer r.mu.Unlock()

	if cached, ok := r.cache[typeName]; ok {
		return cached
	}

	if r.misses[typeName] {
		return nil
	}

	if !r.depsParsed {
		r.parseDependencies()
		r.depsParsed = true
	}

	r.loadAllDependencyTypes()

	if rt, ok := r.cache[typeName]; ok {
		return rt
	}

	r.misses[typeName] = true
	return nil
}

func (r *PipTypeResolver) parseDependencies() {
	deps, err := DetectPipDependencies(r.sourceDir)
	if err != nil {
		return
	}

	seen := make(map[string]bool)
	for _, dep := range deps {
		normalized := normalizePackageName(dep.Name)
		if !seen[normalized] {
			seen[normalized] = true
			r.deps = append(r.deps, normalized)
		}
	}
}

func (r *PipTypeResolver) loadAllDependencyTypes() {
	sitePackagesDir, err := FindSitePackages(r.sourceDir)
	if err != nil || sitePackagesDir == "" {
		return
	}

	for _, dep := range r.deps {
		if r.loadedPkgs[dep] {
			continue
		}
		r.loadedPkgs[dep] = true
		r.loadPackageTypes(sitePackagesDir, dep)
	}
}

func (r *PipTypeResolver) loadPackageTypes(sitePackagesDir string, pkgName string) {
	pkgDir := FindPackageDir(sitePackagesDir, pkgName)
	if pkgDir == "" {
		return
	}

	pyFiles := FindPyFiles(pkgDir)
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

func pydanticModelToResolvedType(md fastapi.PydanticModel) *collector.ResolvedType {
	rt := &collector.ResolvedType{
		Name:        md.Name,
		IsInterface: false,
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

func DetectPipDependencies(sourceDir string) ([]collector.Dependency, error) {
	if deps, err := detectFromRequirementsTxt(sourceDir); err == nil && len(deps) > 0 {
		return deps, nil
	}

	if deps, err := detectFromPyprojectToml(sourceDir); err == nil && len(deps) > 0 {
		return deps, nil
	}

	if deps, err := detectFromPipfile(sourceDir); err == nil && len(deps) > 0 {
		return deps, nil
	}

	return nil, nil
}

func detectFromRequirementsTxt(sourceDir string) ([]collector.Dependency, error) {
	reqFile := filepath.Join(sourceDir, "requirements.txt")
	data, err := os.ReadFile(reqFile)
	if err != nil {
		return nil, err
	}

	var deps []collector.Dependency
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}

		name, version := parseRequirementLine(line)
		if name != "" {
			deps = append(deps, collector.Dependency{
				Name:    name,
				Version: version,
			})
		}
	}

	return deps, nil
}

func detectFromPyprojectToml(sourceDir string) ([]collector.Dependency, error) {
	pyprojectFile := filepath.Join(sourceDir, "pyproject.toml")
	data, err := os.ReadFile(pyprojectFile)
	if err != nil {
		return nil, err
	}

	content := string(data)
	var deps []collector.Dependency
	seen := make(map[string]bool)

	inDeps := false
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "dependencies") && strings.Contains(line, "=") {
			inDeps = true
			if strings.Contains(line, "[") {
				parseInlineArray(line, &deps, seen)
			}
			continue
		}

		if inDeps {
			if line == "]" || (!strings.HasPrefix(line, "\"") && !strings.HasPrefix(line, "'") && !strings.HasPrefix(line, "-") && line != "" && !strings.HasPrefix(line, "#")) {
				if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
					continue
				}
				inDeps = false
				continue
			}

			dep := strings.Trim(line, " \t\"',")
			dep = strings.TrimSuffix(dep, ",")
			if dep == "" || strings.HasPrefix(dep, "#") {
				continue
			}

			name, version := parseRequirementLine(dep)
			if name != "" && !seen[normalizePackageName(name)] {
				seen[normalizePackageName(name)] = true
				deps = append(deps, collector.Dependency{
					Name:    name,
					Version: version,
				})
			}
		}
	}

	scanner = bufio.NewScanner(strings.NewReader(content))
	inOptionalDeps := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "[project.optional-dependencies]") || (strings.Contains(line, "optional-dependencies") && strings.Contains(line, "=")) {
			inOptionalDeps = true
			continue
		}

		if inOptionalDeps {
			if strings.HasPrefix(line, "[") && !strings.Contains(line, "optional-dependencies") {
				inOptionalDeps = false
				continue
			}

			dep := strings.Trim(line, " \t\"',")
			dep = strings.TrimSuffix(dep, ",")
			if dep == "" || strings.HasPrefix(dep, "#") {
				continue
			}

			if strings.Contains(dep, "=") || strings.Contains(dep, ">") || strings.Contains(dep, "<") || strings.Contains(dep, "~") || strings.Contains(dep, "!") {
				name, version := parseRequirementLine(dep)
				if name != "" && !seen[normalizePackageName(name)] {
					seen[normalizePackageName(name)] = true
					deps = append(deps, collector.Dependency{
						Name:    name,
						Version: version,
					})
				}
			}
		}
	}

	return deps, nil
}

func parseInlineArray(line string, deps *[]collector.Dependency, seen map[string]bool) {
	startIdx := strings.Index(line, "[")
	endIdx := strings.LastIndex(line, "]")
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return
	}

	inner := line[startIdx+1 : endIdx]
	items := strings.Split(inner, ",")
	for _, item := range items {
		item = strings.TrimSpace(item)
		item = strings.Trim(item, "\"'")
		if item == "" {
			continue
		}
		name, version := parseRequirementLine(item)
		if name != "" && !seen[normalizePackageName(name)] {
			seen[normalizePackageName(name)] = true
			*deps = append(*deps, collector.Dependency{
				Name:    name,
				Version: version,
			})
		}
	}
}

func detectFromPipfile(sourceDir string) ([]collector.Dependency, error) {
	pipfile := filepath.Join(sourceDir, "Pipfile")
	data, err := os.ReadFile(pipfile)
	if err != nil {
		return nil, err
	}

	content := string(data)
	var deps []collector.Dependency
	seen := make(map[string]bool)

	inPackages := false
	inDevPackages := false
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "[packages]" {
			inPackages = true
			inDevPackages = false
			continue
		}
		if line == "[dev-packages]" {
			inDevPackages = true
			inPackages = false
			continue
		}
		if strings.HasPrefix(line, "[") && line != "[packages]" && line != "[dev-packages]" {
			inPackages = false
			inDevPackages = false
			continue
		}

		if inPackages || inDevPackages {
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			name, version := parsePipfileLine(line)
			if name != "" && !seen[normalizePackageName(name)] {
				seen[normalizePackageName(name)] = true
				deps = append(deps, collector.Dependency{
					Name:    name,
					Version: version,
				})
			}
		}
	}

	return deps, nil
}

func parsePipfileLine(line string) (name, version string) {
	eqIdx := strings.Index(line, "=")
	if eqIdx == -1 {
		name = strings.TrimSpace(line)
		return name, ""
	}

	name = strings.TrimSpace(line[:eqIdx])
	value := strings.TrimSpace(line[eqIdx+1:])

	var rawVersion string
	if strings.HasPrefix(value, "\"") || strings.HasPrefix(value, "'") {
		rawVersion = strings.Trim(value, "\"'")
	} else if strings.HasPrefix(value, "{") {
		rawVersion = extractVersionFromPipfileDict(value)
	}

	if rawVersion == "*" || rawVersion == "" {
		return name, ""
	}

	_, version = parseRequirementLine(rawVersion)
	return name, version
}

func extractVersionFromPipfileDict(dictStr string) string {
	versionKey := "version"
	idx := strings.Index(dictStr, versionKey)
	if idx == -1 {
		return ""
	}

	afterKey := dictStr[idx+len(versionKey):]
	afterKey = strings.TrimSpace(afterKey)
	if !strings.HasPrefix(afterKey, "=") {
		return ""
	}
	afterKey = strings.TrimPrefix(afterKey, "=")
	afterKey = strings.TrimSpace(afterKey)

	if strings.HasPrefix(afterKey, "\"") || strings.HasPrefix(afterKey, "'") {
		quote := string(afterKey[0])
		endIdx := strings.Index(afterKey[1:], quote)
		if endIdx != -1 {
			return afterKey[1 : endIdx+1]
		}
	}

	commaIdx := strings.Index(afterKey, ",")
	braceIdx := strings.Index(afterKey, "}")
	endIdx := len(afterKey)
	if commaIdx != -1 && commaIdx < endIdx {
		endIdx = commaIdx
	}
	if braceIdx != -1 && braceIdx < endIdx {
		endIdx = braceIdx
	}

	return strings.TrimSpace(strings.Trim(afterKey[:endIdx], "\"'"))
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

func normalizePackageName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	return name
}

func FindSitePackages(sourceDir string) (string, error) {
	venvSitePkg := findVenvSitePackages(sourceDir)
	if venvSitePkg != "" {
		return venvSitePkg, nil
	}

	condaSitePkg := findCondaSitePackages()
	if condaSitePkg != "" {
		return condaSitePkg, nil
	}

	return findSystemSitePackages()
}

func findVenvSitePackages(sourceDir string) string {
	venvDirs := []string{
		filepath.Join(sourceDir, ".venv"),
		filepath.Join(sourceDir, "venv"),
		filepath.Join(sourceDir, "env"),
	}

	for _, venvDir := range venvDirs {
		sitePkg := filepath.Join(venvDir, "lib")
		if info, err := os.Stat(sitePkg); err == nil && info.IsDir() {
			entries, err := os.ReadDir(sitePkg)
			if err != nil {
				continue
			}
			for _, entry := range entries {
				if entry.IsDir() && strings.HasPrefix(entry.Name(), "python") {
					candidate := filepath.Join(sitePkg, entry.Name(), "site-packages")
					if dirExists(candidate) {
						return candidate
					}
				}
			}
		}
	}

	return ""
}

func findCondaSitePackages() string {
	condaPrefix := os.Getenv("CONDA_PREFIX")
	if condaPrefix != "" {
		sitePkg := filepath.Join(condaPrefix, "lib")
		if info, err := os.Stat(sitePkg); err == nil && info.IsDir() {
			entries, err := os.ReadDir(sitePkg)
			if err != nil {
				return ""
			}
			for _, entry := range entries {
				if entry.IsDir() && strings.HasPrefix(entry.Name(), "python") {
					candidate := filepath.Join(sitePkg, entry.Name(), "site-packages")
					if dirExists(candidate) {
						return candidate
					}
				}
			}
		}
	}
	return ""
}

func findSystemSitePackages() (string, error) {
	for _, cmdName := range []string{"python3", "python"} {
		cmd := exec.Command(cmdName, "-c", "import site; print(site.getsitepackages()[0])")
		output, err := cmd.Output()
		if err != nil {
			continue
		}
		dir := strings.TrimSpace(string(output))
		if dirExists(dir) {
			return dir, nil
		}
	}
	return "", fmt.Errorf("failed to find site-packages directory")
}

func FindPackageDir(sitePackagesDir string, pkgName string) string {
	normalized := normalizePackageName(pkgName)

	entries, err := os.ReadDir(sitePackagesDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		entryNormalized := normalizePackageName(entry.Name())

		if entryNormalized == normalized {
			return filepath.Join(sitePackagesDir, entry.Name())
		}

		if strings.HasPrefix(entryNormalized, normalized+"_") {
			return filepath.Join(sitePackagesDir, entry.Name())
		}

		if entryNormalized == normalized+"-dist" {
			return filepath.Join(sitePackagesDir, entry.Name())
		}
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		entryName := entry.Name()
		if strings.HasPrefix(entryName, normalized) && strings.Contains(entryName, "-") {
			parts := strings.SplitN(entryName, "-", 2)
			if normalizePackageName(parts[0]) == normalized {
				return filepath.Join(sitePackagesDir, entryName)
			}
		}
	}

	return ""
}

func FindPyFiles(dir string) []string {
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

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
