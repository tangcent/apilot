package gocollector

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	collector "github.com/tangcent/apilot/api-collector"
)

type GoDependencyResolver struct {
	mu        sync.Mutex
	sourceDir string
	cache     map[string]*collector.ResolvedType
	misses    map[string]bool
	moduleDir string
}

func NewGoDependencyResolver(sourceDir string) *GoDependencyResolver {
	return &GoDependencyResolver{
		sourceDir: sourceDir,
		cache:     make(map[string]*collector.ResolvedType),
		misses:    make(map[string]bool),
	}
}

func (r *GoDependencyResolver) DetectDependencies(sourceDir string) ([]collector.Dependency, error) {
	return detectGoDependencies(sourceDir)
}

func (r *GoDependencyResolver) ResolveType(typeName string) *collector.ResolvedType {
	r.mu.Lock()
	defer r.mu.Unlock()

	if cached, ok := r.cache[typeName]; ok {
		return cached
	}

	if r.misses[typeName] {
		return nil
	}

	if r.moduleDir == "" {
		dir, err := resolveModuleCacheDir(r.sourceDir)
		if err != nil {
			r.misses[typeName] = true
			return nil
		}
		r.moduleDir = dir
	}

	rt := r.resolveFromModuleCache(typeName)
	if rt != nil {
		r.cache[typeName] = rt
		return rt
	}

	r.misses[typeName] = true
	return nil
}

func (r *GoDependencyResolver) resolveFromModuleCache(typeName string) *collector.ResolvedType {
	pkgPath, typeName := splitGoType(typeName)
	if pkgPath == "" {
		return nil
	}

	pkgDir := filepath.Join(r.moduleDir, filepath.FromSlash(pkgPath))
	if info, err := os.Stat(pkgDir); err != nil || !info.IsDir() {
		versions, err := findModuleVersions(r.moduleDir, pkgPath)
		if err != nil || len(versions) == 0 {
			return nil
		}
		pkgDir = versions[0]
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, pkgDir, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			rt := extractResolvedTypeFromAST(file, typeName)
			if rt != nil {
				return rt
			}
		}
	}

	return nil
}

func extractResolvedTypeFromAST(file *ast.File, typeName string) *collector.ResolvedType {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != typeName {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				return &collector.ResolvedType{
					Name: typeName,
				}
			}

			rt := &collector.ResolvedType{
				Name: typeName,
			}

			if structType.Fields != nil {
				for _, field := range structType.Fields.List {
					if len(field.Names) == 0 {
						embeddedType := extractGoTypeNameFromExpr(field.Type)
						rt.Fields = append(rt.Fields, collector.ResolvedField{
							Name:     embeddedType,
							Type:     embeddedType,
							Required: true,
						})
						continue
					}

					for _, name := range field.Names {
						sf := collector.ResolvedField{
							Name: name.Name,
							Type: extractGoTypeNameFromExpr(field.Type),
						}

						if field.Tag != nil {
							tag := strings.Trim(field.Tag.Value, "`")
							structTag := reflect.StructTag(tag)
							if jsonTag, ok := structTag.Lookup("json"); ok {
								parts := strings.SplitN(jsonTag, ",", 2)
								if parts[0] != "-" && parts[0] != "" {
									sf.Name = parts[0]
								}
								if len(parts) > 1 && strings.Contains(parts[1], "omitempty") {
									sf.Required = false
								} else {
									sf.Required = true
								}
							}
							if bindingTag, ok := structTag.Lookup("binding"); ok {
								if strings.Contains(bindingTag, "required") {
									sf.Required = true
								}
							}
						} else {
							sf.Required = true
						}

						rt.Fields = append(rt.Fields, sf)
					}
				}
			}

			return rt
		}
	}

	return nil
}

func extractGoTypeNameFromExpr(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return extractGoTypeNameFromExpr(e.X)
	case *ast.ArrayType:
		return "[]" + extractGoTypeNameFromExpr(e.Elt)
	case *ast.MapType:
		return "map[" + extractGoTypeNameFromExpr(e.Key) + "]" + extractGoTypeNameFromExpr(e.Value)
	case *ast.SelectorExpr:
		if x, ok := e.X.(*ast.Ident); ok {
			return x.Name + "." + e.Sel.Name
		}
		return e.Sel.Name
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	}
	return ""
}

func splitGoType(typeName string) (pkgPath, name string) {
	if strings.Contains(typeName, ".") {
		parts := strings.Split(typeName, ".")
		name = parts[len(parts)-1]
		pkgPath = strings.Join(parts[:len(parts)-1], "/")
		return
	}
	return "", typeName
}

func resolveModuleCacheDir(sourceDir string) (string, error) {
	cmd := exec.Command("go", "env", "GOMODCACHE")
	cmd.Dir = sourceDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get GOMODCACHE: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func findModuleVersions(moduleCache string, pkgPath string) ([]string, error) {
	searchDir := filepath.Join(moduleCache, filepath.FromSlash(pkgPath))
	var versions []string

	entries, err := os.ReadDir(searchDir)
	if err != nil {
		parentDir := filepath.Dir(searchDir)
		entries, err = os.ReadDir(parentDir)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), filepath.Base(searchDir)+"@") {
				versionDir := filepath.Join(parentDir, entry.Name())
				subEntries, err := os.ReadDir(versionDir)
				if err != nil {
					continue
				}
				for _, sub := range subEntries {
					if sub.IsDir() {
						versions = append(versions, filepath.Join(versionDir, sub.Name()))
					}
				}
			}
		}
	} else {
		for _, entry := range entries {
			if entry.IsDir() && strings.Contains(entry.Name(), "@") {
				versions = append(versions, filepath.Join(searchDir, entry.Name()))
			}
		}
	}

	return versions, nil
}

type goModule struct {
	Path      string `json:"Path"`
	Version   string `json:"Version"`
	Indirect  bool   `json:"Indirect"`
}

func detectGoDependencies(sourceDir string) ([]collector.Dependency, error) {
	cmd := exec.Command("go", "list", "-m", "-json", "all")
	cmd.Dir = sourceDir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list go modules: %w", err)
	}

	var deps []collector.Dependency
	decoder := json.NewDecoder(strings.NewReader(string(output)))
	for decoder.More() {
		var mod goModule
		if err := decoder.Decode(&mod); err != nil {
			break
		}
		deps = append(deps, collector.Dependency{
			Name:    mod.Path,
			Version: mod.Version,
		})
	}

	return deps, nil
}
