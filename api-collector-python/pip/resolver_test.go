package pip

import (
	"os"
	"path/filepath"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

func TestDetectPipDependencies_RequirementsTxt(t *testing.T) {
	dir := t.TempDir()

	reqContent := "flask==2.3.0\n# comment\nrequests>=2.28.0\nnumpy\n-r other.txt\n"
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(reqContent), 0644); err != nil {
		t.Fatal(err)
	}

	deps, err := DetectPipDependencies(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(deps))
	}

	if deps[0].Name != "flask" || deps[0].Version != "2.3.0" {
		t.Errorf("deps[0] = {%s, %s}, want {flask, 2.3.0}", deps[0].Name, deps[0].Version)
	}
	if deps[1].Name != "requests" || deps[1].Version != "2.28.0" {
		t.Errorf("deps[1] = {%s, %s}, want {requests, 2.28.0}", deps[1].Name, deps[1].Version)
	}
	if deps[2].Name != "numpy" || deps[2].Version != "" {
		t.Errorf("deps[2] = {%s, %s}, want {numpy, }", deps[2].Name, deps[2].Version)
	}
}

func TestDetectPipDependencies_PyprojectToml(t *testing.T) {
	dir := t.TempDir()

	pyprojectContent := `[build-system]
requires = ["setuptools"]

[project]
name = "myapp"
dependencies = [
    "flask==2.3.0",
    "requests>=2.28.0",
    "numpy",
]

[project.optional-dependencies]
dev = [
    "pytest>=7.0",
]
`
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(pyprojectContent), 0644); err != nil {
		t.Fatal(err)
	}

	deps, err := DetectPipDependencies(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps) < 3 {
		t.Fatalf("expected at least 3 dependencies, got %d", len(deps))
	}

	names := make(map[string]string)
	for _, dep := range deps {
		names[dep.Name] = dep.Version
	}

	if v, ok := names["flask"]; !ok || v != "2.3.0" {
		t.Errorf("flask version = %q, want 2.3.0", v)
	}
	if v, ok := names["requests"]; !ok || v != "2.28.0" {
		t.Errorf("requests version = %q, want 2.28.0", v)
	}
	if _, ok := names["numpy"]; !ok {
		t.Error("expected numpy dependency")
	}
}

func TestDetectPipDependencies_Pipfile(t *testing.T) {
	dir := t.TempDir()

	pipfileContent := `[[source]]
url = "https://pypi.org/simple"
verify_ssl = true
name = "pypi"

[packages]
flask = "==2.3.0"
requests = ">=2.28.0"
numpy = "*"

[dev-packages]
pytest = ">=7.0"

[requires]
python_version = "3.11"
`
	if err := os.WriteFile(filepath.Join(dir, "Pipfile"), []byte(pipfileContent), 0644); err != nil {
		t.Fatal(err)
	}

	deps, err := DetectPipDependencies(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps) < 3 {
		t.Fatalf("expected at least 3 dependencies, got %d", len(deps))
	}

	names := make(map[string]string)
	for _, dep := range deps {
		names[dep.Name] = dep.Version
	}

	if v, ok := names["flask"]; !ok || v != "2.3.0" {
		t.Errorf("flask version = %q, want 2.3.0", v)
	}
	if v, ok := names["requests"]; !ok || v != "2.28.0" {
		t.Errorf("requests version = %q, want 2.28.0", v)
	}
	if _, ok := names["numpy"]; !ok {
		t.Error("expected numpy dependency")
	}
}

func TestDetectPipDependencies_NoFiles(t *testing.T) {
	dir := t.TempDir()

	deps, err := DetectPipDependencies(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deps != nil {
		t.Errorf("expected nil deps, got %v", deps)
	}
}

func TestDetectPipDependencies_RequirementsTxtPriority(t *testing.T) {
	dir := t.TempDir()

	reqContent := "flask==1.0.0\n"
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(reqContent), 0644); err != nil {
		t.Fatal(err)
	}

	pipfileContent := `[packages]
flask = "==2.0.0"
`
	if err := os.WriteFile(filepath.Join(dir, "Pipfile"), []byte(pipfileContent), 0644); err != nil {
		t.Fatal(err)
	}

	deps, err := DetectPipDependencies(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps))
	}
	if deps[0].Version != "1.0.0" {
		t.Errorf("expected requirements.txt version 1.0.0 to take priority, got %s", deps[0].Version)
	}
}

func TestParseRequirementLine(t *testing.T) {
	tests := []struct {
		input       string
		wantName    string
		wantVersion string
	}{
		{"flask==2.3.0", "flask", "2.3.0"},
		{"requests>=2.28.0", "requests", "2.28.0"},
		{"numpy<=1.25.0", "numpy", "1.25.0"},
		{"django~=4.2", "django", "4.2"},
		{"package!=1.0", "package", "1.0"},
		{"numpy", "numpy", ""},
		{"scipy>1.0", "scipy", "1.0"},
		{"pandas<2.0", "pandas", "2.0"},
		{"my-pkg==1.0,>=0.5", "my-pkg", "1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			name, version := parseRequirementLine(tt.input)
			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
			if version != tt.wantVersion {
				t.Errorf("version = %q, want %q", version, tt.wantVersion)
			}
		})
	}
}

func TestNormalizePackageName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Flask", "flask"},
		{"my-package", "my_package"},
		{"my.package", "my_package"},
		{"My-Package", "my_package"},
		{"requests", "requests"},
		{"  flask  ", "flask"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizePackageName(tt.input)
			if got != tt.want {
				t.Errorf("normalizePackageName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFindVenvSitePackages(t *testing.T) {
	dir := t.TempDir()

	venvDir := filepath.Join(dir, ".venv", "lib", "python3.11", "site-packages")
	if err := os.MkdirAll(venvDir, 0755); err != nil {
		t.Fatal(err)
	}

	result := findVenvSitePackages(dir)
	if result != venvDir {
		t.Errorf("findVenvSitePackages() = %q, want %q", result, venvDir)
	}
}

func TestFindVenvSitePackages_VenvDir(t *testing.T) {
	dir := t.TempDir()

	venvDir := filepath.Join(dir, "venv", "lib", "python3.12", "site-packages")
	if err := os.MkdirAll(venvDir, 0755); err != nil {
		t.Fatal(err)
	}

	result := findVenvSitePackages(dir)
	if result != venvDir {
		t.Errorf("findVenvSitePackages() = %q, want %q", result, venvDir)
	}
}

func TestFindVenvSitePackages_NoVenv(t *testing.T) {
	dir := t.TempDir()

	result := findVenvSitePackages(dir)
	if result != "" {
		t.Errorf("findVenvSitePackages() = %q, want empty string", result)
	}
}

func TestFindPackageDir(t *testing.T) {
	dir := t.TempDir()
	sitePackages := filepath.Join(dir, "site-packages")
	if err := os.MkdirAll(filepath.Join(sitePackages, "flask"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(sitePackages, "my_package"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(sitePackages, "requests-2.28.0.dist-info"), 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		pkgName string
		want    string
	}{
		{"flask", filepath.Join(sitePackages, "flask")},
		{"my-package", filepath.Join(sitePackages, "my_package")},
		{"nonexistent", ""},
	}

	for _, tt := range tests {
		t.Run(tt.pkgName, func(t *testing.T) {
			got := FindPackageDir(sitePackages, tt.pkgName)
			if got != tt.want {
				t.Errorf("FindPackageDir(%q) = %q, want %q", tt.pkgName, got, tt.want)
			}
		})
	}
}

func TestFindPyFiles(t *testing.T) {
	dir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(dir, "__pycache__"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "model.py"), []byte("class Foo: pass"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "__init__.py"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "__pycache__", "cached.pyc"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	files := FindPyFiles(dir)
	if len(files) != 1 {
		t.Fatalf("expected 1 .py file, got %d", len(files))
	}
	if filepath.Base(files[0]) != "model.py" {
		t.Errorf("expected model.py, got %s", files[0])
	}
}

func TestPipTypeResolver_Interface(t *testing.T) {
	resolver := NewPipTypeResolver("/tmp")
	var _ collector.DependencyResolver = resolver
}

func TestPipTypeResolver_DetectDependencies(t *testing.T) {
	dir := t.TempDir()

	reqContent := "flask==2.3.0\nrequests>=2.28.0\n"
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(reqContent), 0644); err != nil {
		t.Fatal(err)
	}

	resolver := NewPipTypeResolver(dir)
	deps, err := resolver.DetectDependencies(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(deps))
	}
}

func TestPipTypeResolver_ResolveType_Miss(t *testing.T) {
	dir := t.TempDir()

	resolver := NewPipTypeResolver(dir)
	rt := resolver.ResolveType("NonExistentModel")
	if rt != nil {
		t.Errorf("expected nil for non-existent type, got %v", rt)
	}
}

func TestPipTypeResolver_ResolveType_CacheMiss(t *testing.T) {
	dir := t.TempDir()

	resolver := NewPipTypeResolver(dir)

	rt1 := resolver.ResolveType("NonExistentModel")
	if rt1 != nil {
		t.Errorf("expected nil, got %v", rt1)
	}

	rt2 := resolver.ResolveType("NonExistentModel")
	if rt2 != nil {
		t.Errorf("expected nil on second call, got %v", rt2)
	}
}

func TestDetectFromRequirementsTxt_EmptyFile(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	deps, err := detectFromRequirementsTxt(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("expected 0 deps for empty file, got %d", len(deps))
	}
}

func TestDetectFromRequirementsTxt_CommentsAndFlags(t *testing.T) {
	dir := t.TempDir()

	content := "# This is a comment\n-r base.txt\n--index-url https://pypi.org/simple\nflask==2.3.0\n"
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	deps, err := detectFromRequirementsTxt(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps))
	}
	if deps[0].Name != "flask" {
		t.Errorf("expected flask, got %s", deps[0].Name)
	}
}

func TestDetectFromPipfile_DictVersion(t *testing.T) {
	dir := t.TempDir()

	content := `[packages]
mylib = {version = ">=1.0", index = "pypi"}
`
	if err := os.WriteFile(filepath.Join(dir, "Pipfile"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	deps, err := detectFromPipfile(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps))
	}
	if deps[0].Name != "mylib" {
		t.Errorf("expected mylib, got %s", deps[0].Name)
	}
	if deps[0].Version != "1.0" {
		t.Errorf("expected 1.0, got %s", deps[0].Version)
	}
}
