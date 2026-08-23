// Package webview2loader locates and stages the native WebView2 loader used
// by Velo's Windows webview implementation.
package webview2loader

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

const (
	DLLName          = "WebView2Loader.dll"
	nugetPackageName = "microsoft.web.webview2"
	loaderPathEnv    = "WEBVIEW2_LOADER_DLL"
	sdkDirectoryEnv  = "WEBVIEW2_SDK_DIR"
	nugetPackagesEnv = "NUGET_PACKAGES"
)

// Options describes deterministic lookup inputs. Call Find to use the host
// environment and conventional NuGet cache locations.
type Options struct {
	GOARCH            string
	SearchDirectories []string
	LoaderPath        string
	SDKDirectories    []string
	NuGetPackageRoots []string
}

// Find looks for a loader matching goarch. Explicit environment variables are
// preferred, followed by caller-provided/local directories and NuGet caches.
func Find(goarch string, searchDirectories ...string) (string, error) {
	searchDirectories = append([]string{}, searchDirectories...)
	if executable, err := os.Executable(); err == nil {
		searchDirectories = append(searchDirectories, filepath.Dir(executable))
	}
	if workingDirectory, err := os.Getwd(); err == nil {
		searchDirectories = append(searchDirectories, workingDirectory)
	}

	return FindWithOptions(Options{
		GOARCH:            goarch,
		SearchDirectories: searchDirectories,
		LoaderPath:        strings.TrimSpace(os.Getenv(loaderPathEnv)),
		SDKDirectories:    splitPathList(os.Getenv(sdkDirectoryEnv)),
		NuGetPackageRoots: defaultNuGetPackageRoots(),
	})
}

// FindWithOptions performs the lookup without reading process environment.
func FindWithOptions(options Options) (string, error) {
	architecture, ok := architectureDirectory(options.GOARCH)
	if !ok {
		return "", fmt.Errorf("unsupported WebView2 loader architecture %q", options.GOARCH)
	}

	searched := make([]string, 0)
	if options.LoaderPath != "" {
		searched = append(searched, options.LoaderPath)
		if path, ok := regularFile(options.LoaderPath); ok {
			return path, nil
		}
	}

	for _, directory := range uniquePaths(options.SearchDirectories) {
		candidate := filepath.Join(directory, DLLName)
		searched = append(searched, candidate)
		if path, ok := regularFile(candidate); ok {
			return path, nil
		}
	}

	for _, directory := range uniquePaths(options.SDKDirectories) {
		if path, ok := findInSDKDirectory(directory, architecture, &searched); ok {
			return path, nil
		}
	}

	for _, root := range uniquePaths(options.NuGetPackageRoots) {
		if path, ok := findInNuGetRoot(root, architecture, &searched); ok {
			return path, nil
		}
	}

	return "", fmt.Errorf(
		"%s for GOARCH=%s was not found; set %s or %s, place the DLL next to the application, or install Microsoft.Web.WebView2 in the NuGet cache (searched: %s)",
		DLLName,
		options.GOARCH,
		loaderPathEnv,
		sdkDirectoryEnv,
		strings.Join(searched, ", "),
	)
}

// CopyToDirectory copies source to directory using the loader's required file
// name. If source is already the destination, no write is performed.
func CopyToDirectory(source, directory string) (string, error) {
	resolvedSource, ok := regularFile(source)
	if !ok {
		return "", fmt.Errorf("WebView2 loader source is not a regular file: %s", source)
	}
	if err := os.MkdirAll(directory, 0755); err != nil {
		return "", fmt.Errorf("creating WebView2 loader directory: %w", err)
	}
	destination := filepath.Join(directory, DLLName)
	if samePath(resolvedSource, destination) {
		return destination, nil
	}
	data, err := os.ReadFile(resolvedSource)
	if err != nil {
		return "", fmt.Errorf("reading WebView2 loader: %w", err)
	}
	if err := os.WriteFile(destination, data, 0644); err != nil {
		return "", fmt.Errorf("writing WebView2 loader: %w", err)
	}
	return destination, nil
}

func architectureDirectory(goarch string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(goarch)) {
	case "amd64", "x64":
		return "x64", true
	case "386", "x86":
		return "x86", true
	case "arm64":
		return "arm64", true
	default:
		return "", false
	}
}

func defaultNuGetPackageRoots() []string {
	roots := make([]string, 0, 2)
	for _, root := range splitPathList(os.Getenv(nugetPackagesEnv)) {
		roots = append(roots, normalizeNuGetPackageRoot(root))
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		roots = append(roots, filepath.Join(home, ".nuget", "packages", nugetPackageName))
	}
	return uniquePaths(roots)
}

func normalizeNuGetPackageRoot(root string) string {
	root = filepath.Clean(root)
	if strings.EqualFold(filepath.Base(root), nugetPackageName) {
		return root
	}
	return filepath.Join(root, nugetPackageName)
}

func findInSDKDirectory(root, architecture string, searched *[]string) (string, bool) {
	for _, candidate := range []string{
		filepath.Join(root, DLLName),
		filepath.Join(root, "build", "native", architecture, DLLName),
		filepath.Join(root, "runtimes", "win-"+architecture, "native", DLLName),
		filepath.Join(root, "native", architecture, DLLName),
		filepath.Join(root, architecture, DLLName),
	} {
		*searched = append(*searched, candidate)
		if path, ok := regularFile(candidate); ok {
			return path, true
		}
	}
	return "", false
}

func findInNuGetRoot(root, architecture string, searched *[]string) (string, bool) {
	root = normalizeNuGetPackageRoot(root)
	if path, ok := findInSDKDirectory(root, architecture, searched); ok {
		return path, true
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", false
	}
	versions := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			versions = append(versions, entry.Name())
		}
	}
	sort.SliceStable(versions, func(i, j int) bool {
		return compareVersionNames(versions[i], versions[j]) > 0
	})
	for _, version := range versions {
		if path, ok := findInSDKDirectory(filepath.Join(root, version), architecture, searched); ok {
			return path, true
		}
	}
	return "", false
}

func compareVersionNames(left, right string) int {
	leftCore, leftSuffix := splitVersion(left)
	rightCore, rightSuffix := splitVersion(right)
	count := len(leftCore)
	if len(rightCore) > count {
		count = len(rightCore)
	}
	for i := 0; i < count; i++ {
		leftPart := versionPart(leftCore, i)
		rightPart := versionPart(rightCore, i)
		if leftPart < rightPart {
			return -1
		}
		if leftPart > rightPart {
			return 1
		}
	}
	if leftSuffix == rightSuffix {
		return strings.Compare(left, right)
	}
	if leftSuffix == "" {
		return 1
	}
	if rightSuffix == "" {
		return -1
	}
	return strings.Compare(leftSuffix, rightSuffix)
}

func splitVersion(version string) ([]int, string) {
	core, suffix, _ := strings.Cut(strings.TrimSpace(version), "-")
	parts := strings.Split(core, ".")
	numbers := make([]int, len(parts))
	for i, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil {
			return nil, version
		}
		numbers[i] = number
	}
	return numbers, suffix
}

func versionPart(parts []int, index int) int {
	if index >= len(parts) {
		return 0
	}
	return parts[index]
}

func regularFile(path string) (string, bool) {
	if strings.TrimSpace(path) == "" {
		return "", false
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	info, err := os.Stat(absolute)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		return "", false
	}
	return absolute, true
}

func splitPathList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return filepath.SplitList(value)
}

func uniquePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		cleaned := filepath.Clean(path)
		key := cleaned
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, cleaned)
	}
	return result
}

func samePath(left, right string) bool {
	left, leftErr := filepath.Abs(left)
	right, rightErr := filepath.Abs(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
	}
	return filepath.Clean(left) == filepath.Clean(right)
}
