package engine

import (
	"bytes"
	"flag"
	"os"
	"strings"
	"testing"
)

func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.CommandLine.SetOutput(&bytes.Buffer{})
}

func TestRunCLI_ListCollectors(t *testing.T) {
	resetFlags()
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"api-master", "--list-collectors"}

	var output bytes.Buffer
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	RunCLI()

	w.Close()
	os.Stdout = originalStdout
	output.ReadFrom(r)

	result := output.String()
	if !strings.Contains(result, "No collectors registered") {
		t.Errorf("Expected 'No collectors registered' in output, got: %s", result)
	}
}

func TestRunCLI_ListFormatters(t *testing.T) {
	resetFlags()
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"api-master", "--list-formatters"}

	var output bytes.Buffer
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	RunCLI()

	w.Close()
	os.Stdout = originalStdout
	output.ReadFrom(r)

	result := output.String()
	if !strings.Contains(result, "No formatters registered") {
		t.Errorf("Expected 'No formatters registered' in output, got: %s", result)
	}
}

func TestPrintCollectors_NoCollectors(t *testing.T) {
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printCollectors()

	w.Close()
	os.Stdout = originalStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	result := buf.String()

	if !strings.Contains(result, "No collectors registered") {
		t.Errorf("Expected 'No collectors registered' in output, got: %s", result)
	}
}

func TestPrintFormatters_NoFormatters(t *testing.T) {
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printFormatters()

	w.Close()
	os.Stdout = originalStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	result := buf.String()

	if !strings.Contains(result, "No formatters registered") {
		t.Errorf("Expected 'No formatters registered' in output, got: %s", result)
	}
}