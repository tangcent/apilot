// Package collector defines the stable interface contract for all API collector implementations.
// A Collector parses source code for a specific language/framework and produces []ApiEndpoint.
package collector

import "fmt"

// Collector is the interface every language/framework collector must implement.
type Collector interface {
	// Name returns the unique identifier for this collector (e.g. "java", "go").
	Name() string

	// SupportedLanguages returns the list of language identifiers this collector handles.
	SupportedLanguages() []string

	// Collect parses the source directory described by ctx and returns the discovered endpoints.
	Collect(ctx CollectContext) ([]ApiEndpoint, error)
}

// CollectContext carries the inputs needed by a Collector to perform collection.
type CollectContext struct {
	// SourceDir is the absolute path to the root of the source tree to parse.
	SourceDir string `json:"sourceDir"`

	// SourceFile is the absolute path to a specific source file when file-level
	// export of exactly one file is requested. When empty, all files under
	// SourceDir are scanned, unless SourceFiles selects several.
	//
	// It is left empty when several files are selected, so a collector that
	// reads only SourceFile never mistakes one file of many for the whole
	// selection; read the selection through SelectedFiles.
	SourceFile string `json:"sourceFile,omitempty"`

	// SourceFiles holds the absolute paths of every selected source file when
	// file-level export is requested, one or several. When empty, all files
	// under SourceDir are scanned. The files all belong to the project at
	// SourceDir.
	//
	// A collector may use the selection to limit the work it does: parse the
	// whole project, which supplies the types the selected files refer to, but
	// build endpoints only for the declarations of the selected files (see
	// FileSelection and FileSelector). A collector may also ignore it and rely
	// on post-collection filtering by the engine.
	SourceFiles []string `json:"sourceFiles,omitempty"`

	// Frameworks is an optional list of framework hints (e.g. ["spring-mvc", "feign"]).
	Frameworks []string `json:"frameworks,omitempty"`

	// NoDeps disables dependency-based type resolution for this collection.
	// Collectors skip external tooling and dependency artifacts entirely and
	// report DependencyResolutionDisabled.
	NoDeps bool `json:"noDeps,omitempty"`

	// Config holds collector-specific key-value configuration.
	Config map[string]string `json:"config,omitempty"`
}

// ConfigRequiredFields is the Config key that selects how a collector decides
// whether a request or response field is required. Its values are the
// RequiredFields constants; an absent key means RequiredFieldsDefault.
const ConfigRequiredFields = "requiredFields"

const (
	// RequiredFieldsDefault marks a field required unless it is annotated as
	// nullable (@Nullable, @Null).
	RequiredFieldsDefault = "default"
	// RequiredFieldsConstraints marks a field required only when a Bean
	// Validation constraint demands a value: @NotNull, @NotBlank or @NotEmpty.
	// This is how the constraints are enforced at runtime, and how the EasyYapi
	// IntelliJ plugin documents them.
	RequiredFieldsConstraints = "constraints"
)

// RequiredFields returns the requiredness mode the context selects, or an
// error naming the accepted values when it selects an unknown one.
func (c CollectContext) RequiredFields() (string, error) {
	switch mode := c.Config[ConfigRequiredFields]; mode {
	case "", RequiredFieldsDefault:
		return RequiredFieldsDefault, nil
	case RequiredFieldsConstraints:
		return RequiredFieldsConstraints, nil
	default:
		return "", fmt.Errorf("unknown required-fields mode %q: use %q or %q",
			mode, RequiredFieldsDefault, RequiredFieldsConstraints)
	}
}
