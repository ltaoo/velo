package webview2loader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindWithOptionsPrefersExplicitLoader(t *testing.T) {
	root := t.TempDir()
	explicit := writeLoader(t, filepath.Join(root, "explicit", DLLName), "explicit")
	writeLoader(t, filepath.Join(root, "local", DLLName), "local")

	got, err := FindWithOptions(Options{
		GOARCH:            "amd64",
		LoaderPath:        explicit,
		SearchDirectories: []string{filepath.Join(root, "local")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != explicit {
		t.Fatalf("FindWithOptions() = %q, want %q", got, explicit)
	}
}

func TestFindWithOptionsUsesLocalLoader(t *testing.T) {
	root := t.TempDir()
	want := writeLoader(t, filepath.Join(root, DLLName), "local")

	got, err := FindWithOptions(Options{GOARCH: "amd64", SearchDirectories: []string{root}})
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("FindWithOptions() = %q, want %q", got, want)
	}
}

func TestFindWithOptionsUsesNewestNuGetPackage(t *testing.T) {
	packages := filepath.Join(t.TempDir(), "packages", nugetPackageName)
	writeLoader(t, filepath.Join(packages, "1.0.999.0", "build", "native", "x64", DLLName), "old")
	want := writeLoader(t, filepath.Join(packages, "1.0.1000.0", "build", "native", "x64", DLLName), "new")

	got, err := FindWithOptions(Options{GOARCH: "amd64", NuGetPackageRoots: []string{packages}})
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("FindWithOptions() = %q, want %q", got, want)
	}
}

func TestFindWithOptionsUsesRuntimeNuGetLayout(t *testing.T) {
	packages := filepath.Join(t.TempDir(), "packages", nugetPackageName)
	want := writeLoader(t, filepath.Join(packages, "1.0.2792.45", "runtimes", "win-arm64", "native", DLLName), "arm64")

	got, err := FindWithOptions(Options{GOARCH: "arm64", NuGetPackageRoots: []string{packages}})
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("FindWithOptions() = %q, want %q", got, want)
	}
}

func TestFindWithOptionsReportsActionableError(t *testing.T) {
	_, err := FindWithOptions(Options{GOARCH: "amd64", SearchDirectories: []string{t.TempDir()}})
	if err == nil {
		t.Fatal("FindWithOptions() error = nil")
	}
	for _, expected := range []string{DLLName, loaderPathEnv, sdkDirectoryEnv} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("error %q does not contain %q", err, expected)
		}
	}
}

func TestCopyToDirectory(t *testing.T) {
	source := writeLoader(t, filepath.Join(t.TempDir(), DLLName), "loader-data")
	destinationDirectory := t.TempDir()

	destination, err := CopyToDirectory(source, destinationDirectory)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "loader-data" {
		t.Fatalf("copied data = %q", data)
	}
}

func writeLoader(t *testing.T, path, contents string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return absolute
}
