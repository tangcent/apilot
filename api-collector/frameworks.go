package collector

// FrameworkReporter is an optional interface implemented by collectors that
// scan several frameworks and know which ones actually produced endpoints.
//
// A collector is named after a language ("java"), so the name alone does not
// say which framework a document was read from. Reporting the frameworks lets
// an export name its source instead of leaving the reader to guess (#170).
//
// Like UnresolvedReporter it is deliberately not part of Collector: subprocess
// plugins cannot report across the process boundary, so the engine must
// type-assert.
type FrameworkReporter interface {
	// Frameworks returns the frameworks the last Collect call matched, in the
	// order the collector declares them. Empty when the collector cannot tell
	// or matched none.
	Frameworks() []string
}
