package plugin

// dynlib.go — shared library (.so/.dylib/.dll) plugin loader.
//
// NOTE: This feature requires CGO (via dlopen) and is deferred to v2.
// In v1, all external plugins use the subprocess protocol (subprocess.go).
//
// When implemented, this file will use plugin.Open (Go standard library)
// or unsafe CGO dlopen to load a shared library and resolve the
// "NewCollector" or "NewFormatter" symbol by name.
