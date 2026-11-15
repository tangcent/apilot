package engine

import (
	"encoding/json"
	"path/filepath"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	formatter "github.com/tangcent/apilot/api-formatter"
	model "github.com/tangcent/apilot/api-model"
)

// paramsCapturingFormatter records the params it was asked to format with.
type paramsCapturingFormatter struct {
	name   string
	params map[string]any
}

func (f *paramsCapturingFormatter) Name() string { return f.name }
func (f *paramsCapturingFormatter) Format(_ []model.ApiEndpoint, opts formatter.FormatOptions) ([]byte, error) {
	_ = json.Unmarshal(opts.Params, &f.params)
	return []byte("ok"), nil
}

func TestRun_FormatterReceivesSourceDir(t *testing.T) {
	RegisterCollector(&mockCollector{
		name:             "source-dir-collector",
		collectEndpoints: []collector.ApiEndpoint{{Name: "ping", Method: "GET", Path: "/ping"}},
	})
	f := &paramsCapturingFormatter{name: "source-dir-formatter"}
	RegisterFormatter(f)

	dir := t.TempDir()
	err := Run(Config{
		SourceDir:     dir,
		CollectorName: "source-dir-collector",
		FormatterName: "source-dir-formatter",
		OutputPath:    filepath.Join(t.TempDir(), "out.txt"),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	want, _ := filepath.Abs(dir)
	if f.params["sourceDir"] != want {
		t.Fatalf("sourceDir param = %v, want %q", f.params["sourceDir"], want)
	}
}
