package engine

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	formatter "github.com/tangcent/apilot/api-formatter"
	"github.com/tangcent/apilot/api-master/config"
)

func TestRunCLI_ListCollectors(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"apilot", "export", "--list-collectors"}

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
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"apilot", "export", "--list-formatters"}

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

func TestRun_HappyPath(t *testing.T) {
	tmpDir := t.TempDir()

	RegisterCollector(&mockCollector{
		name:             "test-collector",
		supportedLangs:   []string{"test"},
		collectEndpoints: []collector.ApiEndpoint{{Name: "test-endpoint"}},
	})

	RegisterFormatter(&mockFormatter{
		name:         "test-formatter",
		formatOutput: []byte("formatted output"),
	})

	cfg := Config{
		SourceDir:      tmpDir,
		CollectorName:  "test-collector",
		FormatterName:  "test-formatter",
		OutputPath:     "",
		PluginRegistry: "",
	}

	var output bytes.Buffer
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Run(cfg)

	w.Close()
	os.Stdout = originalStdout
	output.ReadFrom(r)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !bytes.Contains(output.Bytes(), []byte("formatted output")) {
		t.Errorf("Expected 'formatted output' in stdout, got: %s", output.String())
	}
}

func TestRun_AutoDetectFailure(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		SourceDir:      tmpDir,
		CollectorName:  "",
		FormatterName:  "test-formatter",
		OutputPath:     "",
		PluginRegistry: "",
	}

	err := Run(cfg)
	if err == nil {
		t.Error("Expected error for auto-detect failure, got nil")
	}
	if !strings.Contains(err.Error(), "auto-detect failed") {
		t.Errorf("Expected 'auto-detect failed' in error, got: %v", err)
	}
}

func TestRun_FormatterNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	RegisterCollector(&mockCollector{
		name:             "test-collector",
		supportedLangs:   []string{"test"},
		collectEndpoints: []collector.ApiEndpoint{},
	})

	cfg := Config{
		SourceDir:      tmpDir,
		CollectorName:  "test-collector",
		FormatterName:  "nonexistent-formatter",
		OutputPath:     "",
		PluginRegistry: "",
	}

	err := Run(cfg)
	if err == nil {
		t.Error("Expected error for missing formatter, got nil")
	}
	if !strings.Contains(err.Error(), "formatter") {
		t.Errorf("Expected 'formatter' in error, got: %v", err)
	}
}

func TestRun_WriteToFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := tmpDir + "/output.md"

	RegisterCollector(&mockCollector{
		name:             "test-collector",
		supportedLangs:   []string{"test"},
		collectEndpoints: []collector.ApiEndpoint{{Name: "test-endpoint"}},
	})

	RegisterFormatter(&mockFormatter{
		name:         "test-formatter",
		formatOutput: []byte("formatted output to file"),
	})

	cfg := Config{
		SourceDir:      tmpDir,
		CollectorName:  "test-collector",
		FormatterName:  "test-formatter",
		OutputPath:     outputFile,
		PluginRegistry: "",
	}

	err := Run(cfg)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !bytes.Contains(data, []byte("formatted output to file")) {
		t.Errorf("Expected 'formatted output to file' in file, got: %s", string(data))
	}
}

func TestRun_CollectorError(t *testing.T) {
	tmpDir := t.TempDir()

	RegisterCollector(&mockCollector{
		name:           "error-collector",
		supportedLangs: []string{"test"},
		collectError:   fmt.Errorf("collection failed"),
	})

	cfg := Config{
		SourceDir:      tmpDir,
		CollectorName:  "error-collector",
		FormatterName:  "test-formatter",
		OutputPath:     "",
		PluginRegistry: "",
	}

	err := Run(cfg)
	if err == nil {
		t.Error("Expected error from collector, got nil")
	}
	if !strings.Contains(err.Error(), "collection failed") {
		t.Errorf("Expected 'collection failed' in error, got: %v", err)
	}
}

func TestRun_FormatterError(t *testing.T) {
	tmpDir := t.TempDir()

	RegisterCollector(&mockCollector{
		name:             "test-collector",
		supportedLangs:   []string{"test"},
		collectEndpoints: []collector.ApiEndpoint{},
	})

	RegisterFormatter(&mockFormatter{
		name:        "error-formatter",
		formatError: fmt.Errorf("formatting failed"),
	})

	cfg := Config{
		SourceDir:      tmpDir,
		CollectorName:  "test-collector",
		FormatterName:  "error-formatter",
		OutputPath:     "",
		PluginRegistry: "",
	}

	err := Run(cfg)
	if err == nil {
		t.Error("Expected error from formatter, got nil")
	}
	if !strings.Contains(err.Error(), "formatting failed") {
		t.Errorf("Expected 'formatting failed' in error, got: %v", err)
	}
}

func TestRun_SettingsInjectedIntoFormatOptions(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := tmpDir + "/config"
	os.Setenv("APILOT_CONFIG_DIR", configDir)
	defer os.Unsetenv("APILOT_CONFIG_DIR")

	config.SetSetting("test.key", "test-value")

	RegisterCollector(&mockCollector{
		name:             "test-collector",
		supportedLangs:   []string{"test"},
		collectEndpoints: []collector.ApiEndpoint{{Name: "test-endpoint"}},
	})

	RegisterFormatter(&mockFormatterWithSettings{
		name: "settings-formatter",
	})

	cfg := Config{
		SourceDir:      tmpDir,
		CollectorName:  "test-collector",
		FormatterName:  "settings-formatter",
		OutputPath:     "",
		PluginRegistry: "",
	}

	err := Run(cfg)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestRun_RequiredSettingMissing(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := tmpDir + "/config2"
	os.Setenv("APILOT_CONFIG_DIR", configDir)
	defer os.Unsetenv("APILOT_CONFIG_DIR")

	RegisterCollector(&mockCollector{
		name:             "test-collector",
		supportedLangs:   []string{"test"},
		collectEndpoints: []collector.ApiEndpoint{{Name: "test-endpoint"}},
	})

	RegisterFormatter(&mockFormatterWithRequiredSetting{
		name: "required-setting-formatter",
	})

	cfg := Config{
		SourceDir:      tmpDir,
		CollectorName:  "test-collector",
		FormatterName:  "required-setting-formatter",
		OutputPath:     "",
		PluginRegistry: "",
	}

	err := Run(cfg)
	if err == nil {
		t.Error("Expected error for missing required setting, got nil")
	}
	if !strings.Contains(err.Error(), "required.key") {
		t.Errorf("Expected 'required.key' in error, got: %v", err)
	}
}

func TestRun_RequiredSettingPresent(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := tmpDir + "/config3"
	os.Setenv("APILOT_CONFIG_DIR", configDir)
	defer os.Unsetenv("APILOT_CONFIG_DIR")

	config.SetSetting("required.key", "required-value")

	RegisterCollector(&mockCollector{
		name:             "test-collector",
		supportedLangs:   []string{"test"},
		collectEndpoints: []collector.ApiEndpoint{{Name: "test-endpoint"}},
	})

	RegisterFormatter(&mockFormatterWithRequiredSetting{
		name: "required-setting-formatter",
	})

	cfg := Config{
		SourceDir:      tmpDir,
		CollectorName:  "test-collector",
		FormatterName:  "required-setting-formatter",
		OutputPath:     "",
		PluginRegistry: "",
	}

	err := Run(cfg)
	if err != nil {
		t.Errorf("Expected no error when required setting is present, got: %v", err)
	}
}

func TestListFormatterSettings(t *testing.T) {
	saved := formatters
	formatters = map[string]formatter.Formatter{}
	defer func() { formatters = saved }()

	RegisterFormatter(&mockFormatterWithSettings{name: "fmt-with-settings"})
	RegisterFormatter(&mockFormatter{name: "fmt-without-settings"})

	settingDefs := ListFormatterSettings()
	if len(settingDefs) != 1 {
		t.Fatalf("Expected 1 setting, got %d", len(settingDefs))
	}
	if settingDefs[0].Key != "test.key" {
		t.Errorf("Expected key 'test.key', got %q", settingDefs[0].Key)
	}
}

func TestRunCLI_SettingsCommand(t *testing.T) {
	ResetRegistry()

	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"apilot", "settings"}

	var output bytes.Buffer
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	RunCLI()

	w.Close()
	os.Stdout = originalStdout
	output.ReadFrom(r)

	result := output.String()
	if !strings.Contains(result, "No settings required") {
		t.Errorf("Expected 'No settings required' in output, got: %s", result)
	}
}

func TestRunCLI_SetAndGetCommand(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := tmpDir + "/config"
	os.Setenv("APILOT_CONFIG_DIR", configDir)
	defer os.Unsetenv("APILOT_CONFIG_DIR")

	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"apilot", "set", "test.cli.value", "test-cli-value"}

	var setOutput bytes.Buffer
	setStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	RunCLI()

	w.Close()
	os.Stdout = setStdout
	setOutput.ReadFrom(r)

	if !strings.Contains(setOutput.String(), "Set test.cli.value") {
		t.Errorf("Expected 'Set test.cli.value' in output, got: %s", setOutput.String())
	}

	os.Args = []string{"apilot", "get", "test.cli.value"}

	var getOutput bytes.Buffer
	getStdout := os.Stdout
	r2, w2, _ := os.Pipe()
	os.Stdout = w2

	RunCLI()

	w2.Close()
	os.Stdout = getStdout
	getOutput.ReadFrom(r2)

	if !strings.Contains(getOutput.String(), "test-cli-value") {
		t.Errorf("Expected 'test-cli-value' in output, got: %s", getOutput.String())
	}
}

func TestRunCLI_GetMasksSensitiveKeys(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := tmpDir + "/config"
	os.Setenv("APILOT_CONFIG_DIR", configDir)
	defer os.Unsetenv("APILOT_CONFIG_DIR")

	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"apilot", "set", "postman.api.key", "PMAK-12345678-abcdef"}
	RunCLI()

	os.Args = []string{"apilot", "get", "postman.api.key"}

	var getOutput bytes.Buffer
	getStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	RunCLI()

	w.Close()
	os.Stdout = getStdout
	getOutput.ReadFrom(r)

	output := getOutput.String()
	if strings.Contains(output, "PMAK-12345678-abcdef") {
		t.Errorf("Sensitive key should be masked, got plain value: %s", output)
	}
	if !strings.Contains(output, "****") {
		t.Errorf("Expected masked output with ****, got: %s", output)
	}
}

func TestRunCLI_HelpCommand(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"apilot", "--help"}

	var output bytes.Buffer
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	RunCLI()

	w.Close()
	os.Stdout = originalStdout
	output.ReadFrom(r)

	result := output.String()
	if !strings.Contains(result, "Flags:") {
		t.Errorf("Expected 'Flags:' in help output, got: %s", result)
	}
}

func TestDetectCollector_Java(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{"pom.xml", "pom.xml"},
		{"build.gradle", "build.gradle"},
		{"build.gradle.kts", "build.gradle.kts"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			indicatorFile := tmpDir + "/" + tt.filename
			if err := os.WriteFile(indicatorFile, []byte{}, 0644); err != nil {
				t.Fatalf("Failed to create indicator file: %v", err)
			}

			RegisterCollector(&mockCollector{name: "java"})

			result, err := detectCollector(tmpDir)
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if result != "java" {
				t.Errorf("Expected 'java', got %q", result)
			}
		})
	}
}

func TestDetectCollector_Go(t *testing.T) {
	tmpDir := t.TempDir()
	indicatorFile := tmpDir + "/go.mod"
	if err := os.WriteFile(indicatorFile, []byte("module example.com/test"), 0644); err != nil {
		t.Fatalf("Failed to create indicator file: %v", err)
	}

	RegisterCollector(&mockCollector{name: "go"})

	result, err := detectCollector(tmpDir)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result != "go" {
		t.Errorf("Expected 'go', got %q", result)
	}
}

func TestDetectCollector_Node(t *testing.T) {
	tmpDir := t.TempDir()
	indicatorFile := tmpDir + "/package.json"
	if err := os.WriteFile(indicatorFile, []byte(`{"name": "test"}`), 0644); err != nil {
		t.Fatalf("Failed to create indicator file: %v", err)
	}

	RegisterCollector(&mockCollector{name: "node"})

	result, err := detectCollector(tmpDir)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result != "node" {
		t.Errorf("Expected 'node', got %q", result)
	}
}

func TestDetectCollector_Python(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{"requirements.txt", "requirements.txt"},
		{"pyproject.toml", "pyproject.toml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			indicatorFile := tmpDir + "/" + tt.filename
			if err := os.WriteFile(indicatorFile, []byte{}, 0644); err != nil {
				t.Fatalf("Failed to create indicator file: %v", err)
			}

			RegisterCollector(&mockCollector{name: "python"})

			result, err := detectCollector(tmpDir)
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if result != "python" {
				t.Errorf("Expected 'python', got %q", result)
			}
		})
	}
}

func TestDetectCollector_NoIndicator(t *testing.T) {
	tmpDir := t.TempDir()

	result, err := detectCollector(tmpDir)
	if err == nil {
		t.Errorf("Expected error for no indicator, got result: %q", result)
	}
	if !strings.Contains(err.Error(), "could not auto-detect") {
		t.Errorf("Expected 'could not auto-detect' in error, got: %v", err)
	}
}

func TestDetectCollector_CollectorNotRegistered(t *testing.T) {
	tmpDir := t.TempDir()
	indicatorFile := tmpDir + "/go.mod"
	if err := os.WriteFile(indicatorFile, []byte("module example.com/test"), 0644); err != nil {
		t.Fatalf("Failed to create indicator file: %v", err)
	}

	saved := collectors
	collectors = map[string]collector.Collector{}
	defer func() { collectors = saved }()

	result, err := detectCollector(tmpDir)
	if err == nil {
		t.Errorf("Expected error when collector not registered, got result: %q", result)
	}
	if !strings.Contains(err.Error(), "could not auto-detect") {
		t.Errorf("Expected 'could not auto-detect' in error, got: %v", err)
	}
}

func TestDetectCollector_FirstMatchWins(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(tmpDir+"/pom.xml", []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create pom.xml: %v", err)
	}
	if err := os.WriteFile(tmpDir+"/go.mod", []byte("module test"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	RegisterCollector(&mockCollector{name: "java"})
	RegisterCollector(&mockCollector{name: "go"})

	result, err := detectCollector(tmpDir)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if result != "java" {
		t.Errorf("Expected 'java' (first match), got %q", result)
	}
}

func TestMaskValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"short", "****"},
		{"PMAK-12345678-abcdef", "PMAK****cdef"},
		{"123456789", "1234****6789"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := maskValue(tt.input)
			if result != tt.expected {
				t.Errorf("maskValue(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFilterEndpointsByFile(t *testing.T) {
	endpoints := []collector.ApiEndpoint{
		{Name: "getUser", Folder: "UserController", Path: "/users/{id}", Method: "GET", Protocol: "http"},
		{Name: "createUser", Folder: "UserController", Path: "/users", Method: "POST", Protocol: "http"},
		{Name: "getOrder", Folder: "OrderController", Path: "/orders/{id}", Method: "GET", Protocol: "http"},
	}

	filtered := filterEndpointsByFile(endpoints, "/project/src/UserController.java")
	if len(filtered) != 2 {
		t.Fatalf("Expected 2 endpoints for UserController, got %d", len(filtered))
	}
	for _, ep := range filtered {
		if ep.Folder != "UserController" {
			t.Errorf("Expected Folder=UserController, got %q", ep.Folder)
		}
	}
}

func TestFilterEndpointsByFile_NoMatch(t *testing.T) {
	endpoints := []collector.ApiEndpoint{
		{Name: "getOrder", Folder: "OrderController", Path: "/orders/{id}", Method: "GET", Protocol: "http"},
	}

	filtered := filterEndpointsByFile(endpoints, "/project/src/UserController.java")
	if len(filtered) != 0 {
		t.Fatalf("Expected 0 endpoints, got %d", len(filtered))
	}
}

func TestFilterEndpointsByMethod(t *testing.T) {
	endpoints := []collector.ApiEndpoint{
		{Name: "getUser", Folder: "UserController", Path: "/users/{id}", Method: "GET", Protocol: "http"},
		{Name: "createUser", Folder: "UserController", Path: "/users", Method: "POST", Protocol: "http"},
		{Name: "getUser", Folder: "OrderController", Path: "/orders/{id}", Method: "GET", Protocol: "http"},
	}

	filtered := filterEndpointsByMethod(endpoints, "getUser")
	if len(filtered) != 2 {
		t.Fatalf("Expected 2 endpoints named getUser, got %d", len(filtered))
	}
	for _, ep := range filtered {
		if ep.Name != "getUser" {
			t.Errorf("Expected Name=getUser, got %q", ep.Name)
		}
	}
}

func TestFilterEndpointsByMethod_NoMatch(t *testing.T) {
	endpoints := []collector.ApiEndpoint{
		{Name: "getUser", Folder: "UserController", Path: "/users/{id}", Method: "GET", Protocol: "http"},
	}

	filtered := filterEndpointsByMethod(endpoints, "deleteUser")
	if len(filtered) != 0 {
		t.Fatalf("Expected 0 endpoints, got %d", len(filtered))
	}
}

func TestFindProjectRoot(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := tmpDir + "/src/main/java/com/example"
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	if err := os.WriteFile(tmpDir+"/pom.xml", []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create pom.xml: %v", err)
	}

	root, err := findProjectRoot(subDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if root != tmpDir {
		t.Errorf("Expected root %q, got %q", tmpDir, root)
	}
}

func TestFindProjectRoot_MultiModule(t *testing.T) {
	tmpDir := t.TempDir()
	moduleDir := filepath.Join(tmpDir, "user-service")
	subDir := filepath.Join(moduleDir, "src", "main", "java", "com", "example")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "pom.xml"), []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create parent pom.xml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "pom.xml"), []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create module pom.xml: %v", err)
	}

	root, err := findProjectRoot(subDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if root != moduleDir {
		t.Errorf("Expected nearest root %q, got %q", moduleDir, root)
	}
}

func TestFindProjectRoot_GradleMultiModule(t *testing.T) {
	tmpDir := t.TempDir()
	moduleDir := filepath.Join(tmpDir, "user-service")
	subDir := filepath.Join(moduleDir, "src", "main", "java", "com", "example")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "build.gradle"), []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create root build.gradle: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "build.gradle"), []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create module build.gradle: %v", err)
	}

	root, err := findProjectRoot(subDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if root != moduleDir {
		t.Errorf("Expected nearest root %q, got %q", moduleDir, root)
	}
}

func TestFindProjectRoot_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := findProjectRoot(tmpDir)
	if err == nil {
		t.Error("Expected error when no project root indicator found, got nil")
	}
}

func TestRun_FileLevelExport(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(tmpDir+"/pom.xml", []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create pom.xml: %v", err)
	}
	javaFile := tmpDir + "/UserController.java"
	if err := os.WriteFile(javaFile, []byte("class UserController {}"), 0644); err != nil {
		t.Fatalf("Failed to create java file: %v", err)
	}

	RegisterCollector(&mockCollector{
		name:           "java",
		supportedLangs: []string{"java"},
		collectEndpoints: []collector.ApiEndpoint{
			{Name: "getUser", Folder: "UserController", Path: "/users/{id}", Method: "GET", Protocol: "http"},
			{Name: "getOrder", Folder: "OrderController", Path: "/orders/{id}", Method: "GET", Protocol: "http"},
		},
	})

	RegisterFormatter(&mockFormatter{
		name:         "test-formatter",
		formatOutput: []byte("filtered output"),
	})

	cfg := Config{
		SourceDir:     javaFile,
		CollectorName: "java",
		FormatterName: "test-formatter",
	}

	var output bytes.Buffer
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Run(cfg)

	w.Close()
	os.Stdout = originalStdout
	output.ReadFrom(r)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !bytes.Contains(output.Bytes(), []byte("filtered output")) {
		t.Errorf("Expected 'filtered output' in stdout, got: %s", output.String())
	}
}

func TestRun_MethodFilter(t *testing.T) {
	tmpDir := t.TempDir()

	RegisterCollector(&mockCollector{
		name:             "test-collector",
		supportedLangs:   []string{"test"},
		collectEndpoints: []collector.ApiEndpoint{{Name: "getUser", Folder: "UserController"}, {Name: "createUser", Folder: "UserController"}},
	})

	var capturedEndpoints []collector.ApiEndpoint
	RegisterFormatter(&captureFormatter{
		name: "capture-formatter",
		capture: &capturedEndpoints,
	})

	cfg := Config{
		SourceDir:     tmpDir,
		CollectorName: "test-collector",
		FormatterName: "capture-formatter",
		MethodFilter:  "getUser",
	}

	err := Run(cfg)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if len(capturedEndpoints) != 1 {
		t.Fatalf("Expected 1 endpoint after method filter, got %d", len(capturedEndpoints))
	}
	if capturedEndpoints[0].Name != "getUser" {
		t.Errorf("Expected Name=getUser, got %q", capturedEndpoints[0].Name)
	}
}

func TestRun_FileAndMethodFilter(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(tmpDir+"/pom.xml", []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create pom.xml: %v", err)
	}
	javaFile := tmpDir + "/UserController.java"
	if err := os.WriteFile(javaFile, []byte("class UserController {}"), 0644); err != nil {
		t.Fatalf("Failed to create java file: %v", err)
	}

	RegisterCollector(&mockCollector{
		name:           "java",
		supportedLangs: []string{"java"},
		collectEndpoints: []collector.ApiEndpoint{
			{Name: "getUser", Folder: "UserController", Path: "/users/{id}", Method: "GET", Protocol: "http"},
			{Name: "createUser", Folder: "UserController", Path: "/users", Method: "POST", Protocol: "http"},
			{Name: "getOrder", Folder: "OrderController", Path: "/orders/{id}", Method: "GET", Protocol: "http"},
		},
	})

	var capturedEndpoints []collector.ApiEndpoint
	RegisterFormatter(&captureFormatter{
		name:    "capture-formatter",
		capture: &capturedEndpoints,
	})

	cfg := Config{
		SourceDir:     javaFile,
		CollectorName: "java",
		FormatterName: "capture-formatter",
		MethodFilter:  "getUser",
	}

	err := Run(cfg)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if len(capturedEndpoints) != 1 {
		t.Fatalf("Expected 1 endpoint after file+method filter, got %d", len(capturedEndpoints))
	}
	if capturedEndpoints[0].Name != "getUser" {
		t.Errorf("Expected Name=getUser, got %q", capturedEndpoints[0].Name)
	}
	if capturedEndpoints[0].Folder != "UserController" {
		t.Errorf("Expected Folder=UserController, got %q", capturedEndpoints[0].Folder)
	}
}

func TestRun_ProjectRootOverride(t *testing.T) {
	projectRoot := t.TempDir()
	moduleDir := projectRoot + "/user-service"
	subDir := moduleDir + "/src/main/java/com/example"
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	if err := os.WriteFile(projectRoot+"/pom.xml", []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create root pom.xml: %v", err)
	}
	if err := os.WriteFile(moduleDir+"/pom.xml", []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create module pom.xml: %v", err)
	}
	javaFile := moduleDir + "/UserController.java"
	if err := os.WriteFile(javaFile, []byte("class UserController {}"), 0644); err != nil {
		t.Fatalf("Failed to create java file: %v", err)
	}

	var capturedEndpoints []collector.ApiEndpoint
	RegisterCollector(&mockCollector{
		name:           "java",
		supportedLangs: []string{"java"},
		collectEndpoints: []collector.ApiEndpoint{
			{Name: "getUser", Folder: "UserController", Path: "/users/{id}", Method: "GET", Protocol: "http"},
		},
	})
	RegisterFormatter(&captureFormatter{
		name:    "capture-formatter",
		capture: &capturedEndpoints,
	})

	cfg := Config{
		SourceDir:     javaFile,
		CollectorName: "java",
		FormatterName: "capture-formatter",
		ProjectRoot:   projectRoot,
	}

	err := Run(cfg)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if len(capturedEndpoints) != 1 {
		t.Fatalf("Expected 1 endpoint, got %d", len(capturedEndpoints))
	}
}

func TestRun_ProjectRootOverrideWithDir(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(projectRoot+"/pom.xml", []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create pom.xml: %v", err)
	}

	subDir := projectRoot + "/subdir"
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	var capturedEndpoints []collector.ApiEndpoint
	RegisterCollector(&mockCollector{
		name:           "java",
		supportedLangs: []string{"java"},
		collectEndpoints: []collector.ApiEndpoint{
			{Name: "getUser", Folder: "UserController", Path: "/users/{id}", Method: "GET", Protocol: "http"},
		},
	})
	RegisterFormatter(&captureFormatter{
		name:    "capture-formatter",
		capture: &capturedEndpoints,
	})

	cfg := Config{
		SourceDir:     subDir,
		CollectorName: "java",
		FormatterName: "capture-formatter",
		ProjectRoot:   projectRoot,
	}

	err := Run(cfg)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

type captureFormatter struct {
	name    string
	capture *[]collector.ApiEndpoint
}

func (m *captureFormatter) Name() string { return m.name }

func (m *captureFormatter) Format(endpoints []collector.ApiEndpoint, opts formatter.FormatOptions) ([]byte, error) {
	*m.capture = endpoints
	return []byte("ok"), nil
}

type mockFormatter struct {
	name         string
	formatOutput []byte
	formatError  error
}

func (m *mockFormatter) Name() string {
	return m.name
}

func (m *mockFormatter) Format(endpoints []collector.ApiEndpoint, opts formatter.FormatOptions) ([]byte, error) {
	return m.formatOutput, m.formatError
}

type mockFormatterWithSettings struct {
	name string
}

func (m *mockFormatterWithSettings) Name() string {
	return m.name
}

func (m *mockFormatterWithSettings) Format(endpoints []collector.ApiEndpoint, opts formatter.FormatOptions) ([]byte, error) {
	return []byte("ok"), nil
}

func (m *mockFormatterWithSettings) RequiredSettings() []formatter.SettingDef {
	return []formatter.SettingDef{
		{
			Key:         "test.key",
			Description: "A test setting",
			Required:    false,
		},
	}
}

type mockFormatterWithRequiredSetting struct {
	name string
}

func (m *mockFormatterWithRequiredSetting) Name() string {
	return m.name
}

func (m *mockFormatterWithRequiredSetting) Format(endpoints []collector.ApiEndpoint, opts formatter.FormatOptions) ([]byte, error) {
	return []byte("ok"), nil
}

func (m *mockFormatterWithRequiredSetting) RequiredSettings() []formatter.SettingDef {
	return []formatter.SettingDef{
		{
			Key:         "required.key",
			Description: "A required setting",
			Required:    true,
		},
	}
}

type mockCollector struct {
	name             string
	supportedLangs   []string
	collectEndpoints []collector.ApiEndpoint
	collectError     error
}

func (m *mockCollector) Name() string {
	return m.name
}

func (m *mockCollector) SupportedLanguages() []string {
	if m.supportedLangs != nil {
		return m.supportedLangs
	}
	return []string{m.name}
}

func (m *mockCollector) Collect(ctx collector.CollectContext) ([]collector.ApiEndpoint, error) {
	return m.collectEndpoints, m.collectError
}

func TestPrintCollectors_NoCollectors(t *testing.T) {
	saved := collectors
	collectors = map[string]collector.Collector{}
	defer func() { collectors = saved }()

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
	saved := formatters
	formatters = map[string]formatter.Formatter{}
	defer func() { formatters = saved }()

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
