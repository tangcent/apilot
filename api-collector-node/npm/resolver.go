package npm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-node/express"
)

type NpmTypeResolver struct {
	mu         sync.Mutex
	sourceDir  string
	registry   *express.TSTypeRegistry
	cache      map[string]*collector.ResolvedType
	misses     map[string]bool
	deps       []string
	loadedPkgs map[string]bool
	depsParsed bool
}

func NewNpmTypeResolver(sourceDir string) *NpmTypeResolver {
	return &NpmTypeResolver{
		sourceDir:  sourceDir,
		registry:   express.NewTSTypeRegistry(),
		cache:      make(map[string]*collector.ResolvedType),
		misses:     make(map[string]bool),
		loadedPkgs: make(map[string]bool),
	}
}

func (r *NpmTypeResolver) DetectDependencies(sourceDir string) ([]collector.Dependency, error) {
	return DetectNpmDependencies(sourceDir)
}

func (r *NpmTypeResolver) ResolveType(typeName string) *collector.ResolvedType {
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

	rt := r.resolveFromRegistry(typeName)
	if rt != nil {
		r.cache[typeName] = rt
		return rt
	}

	r.misses[typeName] = true
	return nil
}

func (r *NpmTypeResolver) parseDependencies() {
	pkgPath := filepath.Join(r.sourceDir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return
	}

	seen := make(map[string]bool)
	addDep := func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		r.deps = append(r.deps, name)
	}

	for name := range pkg.Dependencies {
		addDep(name)
	}
	for name := range pkg.DevDependencies {
		addDep(name)
	}
	for name := range pkg.PeerDependencies {
		addDep(name)
	}
}

func (r *NpmTypeResolver) loadAllDependencyTypes() {
	for _, dep := range r.deps {
		if r.loadedPkgs[dep] {
			continue
		}
		r.loadedPkgs[dep] = true
		r.loadPackageTypes(dep)
	}
}

func (r *NpmTypeResolver) loadPackageTypes(pkgName string) {
	dtsFiles := FindDTSFilesForPackage(r.sourceDir, pkgName)
	for _, f := range dtsFiles {
		reg, err := express.ExtractTypesFromFile(f)
		if err != nil {
			continue
		}
		r.registry.Merge(reg)
	}
}

func (r *NpmTypeResolver) resolveFromRegistry(typeName string) *collector.ResolvedType {
	baseName := typeName
	if idx := strings.Index(typeName, "<"); idx != -1 {
		baseName = typeName[:idx]
	}

	if iface, found := r.registry.Interfaces[baseName]; found {
		return tsInterfaceToResolvedType(iface)
	}

	if alias, found := r.registry.TypeAliases[baseName]; found {
		return tsTypeAliasToResolvedType(alias)
	}

	if enum, found := r.registry.Enums[baseName]; found {
		return tsEnumToResolvedType(enum)
	}

	return nil
}

func FindDTSFilesForPackage(sourceDir string, pkgName string) []string {
	var files []string
	nodeModulesDir := filepath.Join(sourceDir, "node_modules")

	pkgDir := filepath.Join(nodeModulesDir, pkgName)
	if !dirExists(pkgDir) {
		return findAtTypesDTS(sourceDir, pkgName)
	}

	typesFile := readTypesFieldFromPackageJSON(filepath.Join(pkgDir, "package.json"))
	if typesFile != "" {
		typesPath := filepath.Join(pkgDir, typesFile)
		if fileExists(typesPath) {
			files = append(files, typesPath)
		}
	}

	inlineDTS := filepath.Join(pkgDir, "index.d.ts")
	if fileExists(inlineDTS) {
		files = append(files, inlineDTS)
	}

	atTypesFiles := findAtTypesDTS(sourceDir, pkgName)
	files = append(files, atTypesFiles...)

	return files
}

func findAtTypesDTS(sourceDir string, pkgName string) []string {
	var files []string
	nodeModulesDir := filepath.Join(sourceDir, "node_modules")

	unscopedName := pkgName
	if strings.HasPrefix(pkgName, "@") {
		parts := strings.SplitN(pkgName[1:], "/", 2)
		if len(parts) == 2 {
			unscopedName = parts[1]
		}
	}

	atTypesDir := filepath.Join(nodeModulesDir, "@types", unscopedName)
	if dirExists(atTypesDir) {
		typesFile := readTypesFieldFromPackageJSON(filepath.Join(atTypesDir, "package.json"))
		if typesFile != "" {
			typesPath := filepath.Join(atTypesDir, typesFile)
			if fileExists(typesPath) {
				files = append(files, typesPath)
			}
		}

		atTypesDTS := filepath.Join(atTypesDir, "index.d.ts")
		if fileExists(atTypesDTS) {
			files = append(files, atTypesDTS)
		}
	}

	if strings.HasPrefix(pkgName, "@") {
		scopedAtTypesDir := filepath.Join(nodeModulesDir, "@types", pkgName[1:])
		if dirExists(scopedAtTypesDir) && scopedAtTypesDir != atTypesDir {
			typesFile := readTypesFieldFromPackageJSON(filepath.Join(scopedAtTypesDir, "package.json"))
			if typesFile != "" {
				typesPath := filepath.Join(scopedAtTypesDir, typesFile)
				if fileExists(typesPath) {
					files = append(files, typesPath)
				}
			}

			scopedAtTypesDTS := filepath.Join(scopedAtTypesDir, "index.d.ts")
			if fileExists(scopedAtTypesDTS) {
				files = append(files, scopedAtTypesDTS)
			}
		}
	}

	return files
}

func readTypesFieldFromPackageJSON(pkgJSONPath string) string {
	data, err := os.ReadFile(pkgJSONPath)
	if err != nil {
		return ""
	}

	var pkg struct {
		Types   string `json:"types"`
		Typings string `json:"typings"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return ""
	}

	if pkg.Types != "" {
		return pkg.Types
	}

	return pkg.Typings
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func tsInterfaceToResolvedType(iface *express.TSInterface) *collector.ResolvedType {
	rt := &collector.ResolvedType{
		Name:           iface.Name,
		TypeParameters: iface.TypeParameters,
		IsInterface:    true,
	}

	for _, f := range iface.Fields {
		rt.Fields = append(rt.Fields, collector.ResolvedField{
			Name:     f.Name,
			Type:     f.Type,
			Required: f.Required,
		})
	}

	return rt
}

func tsTypeAliasToResolvedType(alias *express.TSTypeAlias) *collector.ResolvedType {
	return &collector.ResolvedType{
		Name:           alias.Name,
		TypeParameters: alias.TypeParameters,
		IsInterface:    false,
	}
}

func tsEnumToResolvedType(enum *express.TSEnum) *collector.ResolvedType {
	return &collector.ResolvedType{
		Name:        enum.Name,
		IsInterface: false,
	}
}

type packageJSON struct {
	Dependencies     map[string]string `json:"dependencies"`
	DevDependencies  map[string]string `json:"devDependencies"`
	PeerDependencies map[string]string `json:"peerDependencies"`
}

func DetectNpmDependencies(sourceDir string) ([]collector.Dependency, error) {
	pkgPath := filepath.Join(sourceDir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil, nil
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}

	seen := make(map[string]bool)
	var deps []collector.Dependency

	addDep := func(name, version string) {
		if seen[name] {
			return
		}
		seen[name] = true
		cleanVersion := strings.TrimPrefix(version, "^")
		cleanVersion = strings.TrimPrefix(cleanVersion, "~")
		cleanVersion = strings.TrimPrefix(cleanVersion, ">=")
		cleanVersion = strings.TrimPrefix(cleanVersion, "<=")
		deps = append(deps, collector.Dependency{
			Name:    name,
			Version: cleanVersion,
		})
	}

	for name, version := range pkg.Dependencies {
		addDep(name, version)
	}
	for name, version := range pkg.DevDependencies {
		addDep(name, version)
	}
	for name, version := range pkg.PeerDependencies {
		addDep(name, version)
	}

	return deps, nil
}

func ResolveTypeDeclarations(sourceDir string, missingTypes []string) (*express.TSTypeRegistry, error) {
	resolver := NewNpmTypeResolver(sourceDir)
	registry := express.NewTSTypeRegistry()

	for _, typeName := range missingTypes {
		rt := resolver.ResolveType(typeName)
		if rt == nil {
			continue
		}

		if rt.IsInterface {
			iface := &express.TSInterface{
				Name:           rt.Name,
				TypeParameters: rt.TypeParameters,
			}
			for _, f := range rt.Fields {
				iface.Fields = append(iface.Fields, express.TSField{
					Name:     f.Name,
					Type:     f.Type,
					Required: f.Required,
				})
			}
			registry.Interfaces[rt.Name] = iface
		} else {
			alias := &express.TSTypeAlias{
				Name:           rt.Name,
				TypeParameters: rt.TypeParameters,
			}
			registry.TypeAliases[rt.Name] = alias
		}
	}

	return registry, nil
}
