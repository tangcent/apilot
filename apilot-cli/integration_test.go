package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	var err error
	binaryPath, err = buildBinary()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build binary: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(binaryPath)

	os.Exit(m.Run())
}

func buildBinary() (string, error) {
	_, filename, _, _ := runtime.Caller(0)
	cliDir := filepath.Dir(filename)

	tmpFile, err := os.CreateTemp("", "apilot-test-*")
	if err != nil {
		return "", err
	}
	binaryPath := tmpFile.Name()
	tmpFile.Close()

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = cliDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("build failed: %v\n%s", err, output)
	}

	return binaryPath, nil
}

func runCLI(args ...string) (string, error) {
	cmd := exec.Command(binaryPath, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func runCLIWithExitCode(args ...string) (string, int) {
	cmd := exec.Command(binaryPath, args...)
	output, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}
	return string(output), exitCode
}

func TestHelpOutput(t *testing.T) {
	output, err := runCLI("--help")
	if err != nil {
		t.Fatalf("--help should not error: %v", err)
	}

	if !strings.Contains(output, "Usage: apilot <source-path> [flags]") {
		t.Error("help output should contain usage line")
	}

	if !strings.Contains(output, "--collector") {
		t.Error("help output should contain --collector flag")
	}

	if !strings.Contains(output, "--formatter") {
		t.Error("help output should contain --formatter flag")
	}

	if !strings.Contains(output, "Registered collectors:") {
		t.Error("help output should list registered collectors")
	}

	if !strings.Contains(output, "Registered formatters:") {
		t.Error("help output should list registered formatters")
	}
}

func TestListCollectors(t *testing.T) {
	output, err := runCLI("--list-collectors")
	if err != nil {
		t.Fatalf("--list-collectors should not error: %v", err)
	}

	if !strings.Contains(output, "go:") {
		t.Error("should list go collector")
	}
	if !strings.Contains(output, "java:") {
		t.Error("should list java collector")
	}
	if !strings.Contains(output, "node:") {
		t.Error("should list node collector")
	}
	if !strings.Contains(output, "python:") {
		t.Error("should list python collector")
	}
}

func TestListFormatters(t *testing.T) {
	output, err := runCLI("--list-formatters")
	if err != nil {
		t.Fatalf("--list-formatters should not error: %v", err)
	}

	if !strings.Contains(output, "markdown") {
		t.Error("should list markdown formatter")
	}
	if !strings.Contains(output, "curl") {
		t.Error("should list curl formatter")
	}
	if !strings.Contains(output, "postman") {
		t.Error("should list postman formatter")
	}
}

func TestGoProjectWithMarkdownFormatter(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	testdataDir := filepath.Join(filepath.Dir(filename), "testdata", "goproject")

	output, exitCode := runCLIWithExitCode(testdataDir, "--collector", "go", "--formatter", "markdown")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, output: %s", exitCode, output)
	}
}

func TestGoProjectWithCurlFormatter(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	testdataDir := filepath.Join(filepath.Dir(filename), "testdata", "goproject")

	output, exitCode := runCLIWithExitCode(testdataDir, "--collector", "go", "--formatter", "curl")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, output: %s", exitCode, output)
	}
}

func TestGoProjectWithPostmanFormatter(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	testdataDir := filepath.Join(filepath.Dir(filename), "testdata", "goproject")

	output, exitCode := runCLIWithExitCode(testdataDir, "--collector", "go", "--formatter", "postman")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, output: %s", exitCode, output)
	}
}

func TestMissingSourcePath(t *testing.T) {
	output, exitCode := runCLIWithExitCode()
	if exitCode != 1 {
		t.Fatalf("expected exit code 1 for missing source path, got %d, output: %s", exitCode, output)
	}

	if !strings.Contains(output, "source path required") {
		t.Error("output should contain 'source path required' error")
	}
}

func TestNonExistentSourcePath(t *testing.T) {
	output, exitCode := runCLIWithExitCode("/nonexistent/path")
	if exitCode != 1 {
		t.Fatalf("expected exit code 1 for non-existent path, got %d, output: %s", exitCode, output)
	}

	if !strings.Contains(output, "error") {
		t.Error("output should contain error message")
	}
}

func TestAutoDetectCollector(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	testdataDir := filepath.Join(filepath.Dir(filename), "testdata", "goproject")

	output, exitCode := runCLIWithExitCode(testdataDir)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0 for auto-detect, got %d, output: %s", exitCode, output)
	}
}

func TestOutputToFile(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	testdataDir := filepath.Join(filepath.Dir(filename), "testdata", "goproject")

	tmpFile, err := os.CreateTemp("", "apilot-output-*.md")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	output, exitCode := runCLIWithExitCode(testdataDir, "--collector", "go", "--output", tmpPath)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, output: %s", exitCode, output)
	}

	_, err = os.Stat(tmpPath)
	if err != nil {
		t.Fatalf("output file should exist: %v", err)
	}
}

func TestVersionFlag(t *testing.T) {
	output, exitCode := runCLIWithExitCode("--version")
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, output: %s", exitCode, output)
	}

	if !strings.Contains(output, "apilot") {
		t.Error("version output should contain 'apilot'")
	}
}
