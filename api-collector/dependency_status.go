package collector

// DependencyResolutionState classifies the outcome of dependency-based type
// resolution during a Collect call.
type DependencyResolutionState string

const (
	// DependencyResolutionActive means the dependency resolver ran and could
	// expand types beyond the project's own sources.
	DependencyResolutionActive DependencyResolutionState = "active"

	// DependencyResolutionToolMissing means resolution was skipped because a
	// prerequisite is absent: an external tool is not on PATH, or the artifacts
	// a resolver reads from have not been fetched. Detail names it and says
	// how to restore the capability.
	DependencyResolutionToolMissing DependencyResolutionState = "tool-missing"

	// DependencyResolutionNoDeps means there was nothing to resolve: the
	// project carries no dependency manifest or declares no dependencies.
	DependencyResolutionNoDeps DependencyResolutionState = "no-deps"

	// DependencyResolutionDisabled means the caller turned dependency
	// resolution off (e.g. the --no-deps flag).
	DependencyResolutionDisabled DependencyResolutionState = "disabled"
)

// DependencyResolution reports how dependency-based type resolution fared
// during the last Collect call.
//
// Dependency types live outside the project's own sources — dependency JARs,
// node_modules, site-packages, the Go module cache — and reaching them depends
// on tooling or artifacts that are often absent. When resolution is skipped
// silently, an export full of opaque scalar types looks like a resolution
// success; this report tells the two apart.
type DependencyResolution struct {
	// State classifies the outcome.
	State DependencyResolutionState

	// Detail explains a non-active state: it names the missing tool or
	// artifacts with guidance for restoring them. Empty for Active, which
	// needs no explanation.
	Detail string

	// ResolvedTypes counts the distinct endpoint types expanded from
	// dependencies during the last Collect call. Meaningful only for Active;
	// zero there means the endpoints never referenced a dependency type.
	ResolvedTypes int
}

// DependencyResolutionReporter is an optional interface implemented by
// collectors whose type resolution can reach beyond the project's own sources.
//
// Like UnresolvedReporter, it is deliberately not part of Collector: subprocess
// plugins cannot report across the process boundary, so the engine must
// type-assert.
type DependencyResolutionReporter interface {
	// DependencyResolution reports the outcome of dependency-based type
	// resolution during the last Collect call.
	DependencyResolution() DependencyResolution
}
