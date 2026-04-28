package nodecollector

import (
	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-node/npm"
)

type NodeDependencyResolver struct {
	delegate *npm.NpmTypeResolver
}

func NewNodeDependencyResolver(sourceDir string) *NodeDependencyResolver {
	return &NodeDependencyResolver{
		delegate: npm.NewNpmTypeResolver(sourceDir),
	}
}

func (r *NodeDependencyResolver) DetectDependencies(sourceDir string) ([]collector.Dependency, error) {
	return r.delegate.DetectDependencies(sourceDir)
}

func (r *NodeDependencyResolver) ResolveType(typeName string) *collector.ResolvedType {
	return r.delegate.ResolveType(typeName)
}
