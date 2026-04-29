package pycollector

import (
	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-python/pip"
)

type PythonDependencyResolver struct {
	delegate *pip.PipTypeResolver
}

func NewPythonDependencyResolver(sourceDir string) *PythonDependencyResolver {
	return &PythonDependencyResolver{
		delegate: pip.NewPipTypeResolver(sourceDir),
	}
}

func (r *PythonDependencyResolver) DetectDependencies(sourceDir string) ([]collector.Dependency, error) {
	return r.delegate.DetectDependencies(sourceDir)
}

func (r *PythonDependencyResolver) ResolveType(typeName string) *collector.ResolvedType {
	return r.delegate.ResolveType(typeName)
}
