package collector

// WarningsReporter is an optional interface implemented by collectors that can
// finish a collection while knowing part of the export is thinner than the
// source. A response body the collector had to leave opaque is the example
// this was added for: the document is produced, but a whole section is marked
// rather than resolved.
//
// Without it the only evidence is the document itself, and an opaque section
// looks like an API that returns nothing.
//
// Like UnresolvedReporter and DependencyResolutionReporter it is deliberately
// not part of Collector: subprocess plugins cannot report across the process
// boundary, so the engine must type-assert.
type WarningsReporter interface {
	// Warnings returns the diagnostics raised during the last Collect call,
	// one message per gap. Each message should name the endpoint it concerns.
	Warnings() []string
}
