package collector

type Dependency struct {
	Name    string
	Version string
}

type ResolvedField struct {
	Name     string
	Type     string
	Required bool
}

type ResolvedType struct {
	Name               string
	Fields             []ResolvedField
	TypeParameters     []string
	SuperClass         string
	SuperClassTypeArgs []string
	IsInterface        bool
	Interfaces         []string
}

type DependencyResolver interface {
	DetectDependencies(sourceDir string) ([]Dependency, error)

	ResolveType(typeName string) *ResolvedType
}
