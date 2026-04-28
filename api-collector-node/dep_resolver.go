package nodecollector

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

type NodeDependencyResolver struct {
	mu       sync.Mutex
	sourceDir string
	registry *express.TSTypeRegistry
	cache    map[string]*collector.ResolvedType
	misses   map[string]bool
	loaded   bool
}

func NewNodeDependencyResolver(sourceDir string) *NodeDependencyResolver {
	return &NodeDependencyResolver{
		sourceDir: sourceDir,
		registry:  express.NewTSTypeRegistry(),
		cache:     make(map[string]*collector.ResolvedType),
		misses:    make(map[string]bool),
	}
}

func (r *NodeDependencyResolver) DetectDependencies(sourceDir string) ([]collector.Dependency, error) {
	return detectNodeDependencies(sourceDir)
}

func (r *NodeDependencyResolver) ResolveType(typeName string) *collector.ResolvedType {
	r.mu.Lock()
	defer r.mu.Unlock()

	if cached, ok := r.cache[typeName]; ok {
		return cached
	}

	if r.misses[typeName] {
		return nil
	}

	if !r.loaded {
		r.loadFromNodeModules()
		r.loaded = true
	}

	rt := r.resolveFromRegistry(typeName)
	if rt != nil {
		r.cache[typeName] = rt
		return rt
	}

	r.misses[typeName] = true
	return nil
}

func (r *NodeDependencyResolver) loadFromNodeModules() {
	nodeModulesDir := filepath.Join(r.sourceDir, "node_modules")
	info, err := os.Stat(nodeModulesDir)
	if err != nil || !info.IsDir() {
		return
	}

	entries, err := os.ReadDir(nodeModulesDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pkgDir := filepath.Join(nodeModulesDir, entry.Name())
		if strings.HasPrefix(entry.Name(), "@") {
			scopedEntries, err := os.ReadDir(pkgDir)
			if err != nil {
				continue
			}
			for _, scoped := range scopedEntries {
				if scoped.IsDir() {
					r.loadPackageTypes(filepath.Join(pkgDir, scoped.Name()))
				}
			}
		} else {
			r.loadPackageTypes(pkgDir)
		}
	}
}

func (r *NodeDependencyResolver) loadPackageTypes(pkgDir string) {
	dtsFiles := findDTSFiles(pkgDir)
	if len(dtsFiles) == 0 {
		return
	}

	for _, f := range dtsFiles {
		reg, err := express.ExtractTypesFromFile(f)
		if err != nil {
			continue
		}
		r.registry.Merge(reg)
	}
}

func (r *NodeDependencyResolver) resolveFromRegistry(typeName string) *collector.ResolvedType {
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

	return nil
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

func findDTSFiles(dir string) []string {
	var files []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == "node_modules" || name == ".git" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".d.ts") && !strings.HasSuffix(path, ".d.ts.map") {
			files = append(files, path)
		}
		return nil
	})
	return files
}

type packageJSON struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Dependencies map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	PeerDependencies map[string]string `json:"peerDependencies"`
}

func detectNodeDependencies(sourceDir string) ([]collector.Dependency, error) {
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
