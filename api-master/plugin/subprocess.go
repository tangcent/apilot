package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/tangcent/apilot/api-collector/collector"
	"github.com/tangcent/apilot/api-formater/formater"
)

// subprocessCollector wraps an external binary as a collector.Collector.
type subprocessCollector struct {
	manifest PluginManifest
}

func newSubprocessCollector(m PluginManifest) (collector.Collector, error) {
	if err := checkExecutable(m); err != nil {
		return nil, err
	}
	return &subprocessCollector{manifest: m}, nil
}

func (s *subprocessCollector) Name() string { return s.manifest.Name }

func (s *subprocessCollector) SupportedLanguages() []string {
	// TODO: query the subprocess with a --supported-languages flag
	return nil
}

func (s *subprocessCollector) Collect(ctx collector.CollectContext) ([]collector.ApiEndpoint, error) {
	input, err := json.Marshal(ctx)
	if err != nil {
		return nil, err
	}
	out, err := runSubprocess(s.manifest, input)
	if err != nil {
		return nil, err
	}
	return collector.UnmarshalEndpoints(out)
}

// subprocessFormatter wraps an external binary as a formater.Formatter.
type subprocessFormatter struct {
	manifest PluginManifest
}

func newSubprocessFormatter(m PluginManifest) (formater.Formatter, error) {
	if err := checkExecutable(m); err != nil {
		return nil, err
	}
	return &subprocessFormatter{manifest: m}, nil
}

func (s *subprocessFormatter) Name() string { return s.manifest.Name }

func (s *subprocessFormatter) SupportedFormats() []string {
	// TODO: query the subprocess with a --supported-formats flag
	return nil
}

func (s *subprocessFormatter) Format(endpoints []collector.ApiEndpoint, opts formater.FormatOptions) ([]byte, error) {
	envelope := struct {
		Endpoints []collector.ApiEndpoint `json:"endpoints"`
		Options   formater.FormatOptions  `json:"options"`
	}{Endpoints: endpoints, Options: opts}

	input, err := json.Marshal(envelope)
	if err != nil {
		return nil, err
	}
	return runSubprocess(s.manifest, input)
}

func runSubprocess(m PluginManifest, stdin []byte) ([]byte, error) {
	args := append([]string{}, m.Args...)
	cmd := exec.Command(m.Command, args...)
	cmd.Stdin = bytes.NewReader(stdin)
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("subprocess %q failed: %w", m.Command, err)
	}
	return out, nil
}

func checkExecutable(m PluginManifest) error {
	cmd := m.Command
	if cmd == "" {
		cmd = m.Path
	}
	if cmd == "" {
		return fmt.Errorf("plugin %q has no command or path", m.Name)
	}
	if _, err := exec.LookPath(cmd); err != nil {
		if _, err2 := os.Stat(cmd); err2 != nil {
			return fmt.Errorf("plugin binary %q not found or not executable", cmd)
		}
	}
	return nil
}
