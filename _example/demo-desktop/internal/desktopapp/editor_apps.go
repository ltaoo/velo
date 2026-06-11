package desktopapp

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const editorAppIDPrefix = "app:"
const editorNoneAppID = "none"
const editorSystemAppID = "system"

type EditorAppInfo struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Path         string `json:"path,omitempty"`
	Kind         string `json:"kind"`
	Available    bool   `json:"available"`
	SupportsLine bool   `json:"supportsLine"`
}

func listEditorApplications(query string) []EditorAppInfo {
	query = strings.ToLower(strings.TrimSpace(query))
	seen := make(map[string]bool)
	apps := make([]EditorAppInfo, 0, 32)

	add := func(app EditorAppInfo) {
		app.ID = strings.TrimSpace(app.ID)
		app.Name = strings.TrimSpace(app.Name)
		app.Path = strings.TrimSpace(app.Path)
		if app.ID == "" || app.Name == "" {
			return
		}
		if query != "" && !strings.Contains(strings.ToLower(app.Name), query) && !strings.Contains(strings.ToLower(app.Path), query) && !strings.Contains(strings.ToLower(app.ID), query) {
			return
		}
		key := app.ID
		if key == "" {
			key = app.Path
		}
		if seen[key] {
			return
		}
		seen[key] = true
		apps = append(apps, app)
	}

	for _, app := range knownEditorApplications() {
		add(app)
	}
	add(noopApplication())
	add(systemDefaultApplication())
	for _, app := range discoverLocalApplications() {
		add(app)
	}

	sort.SliceStable(apps, func(i, j int) bool {
		if apps[i].Kind != apps[j].Kind {
			return apps[i].Kind == "editor"
		}
		left := strings.ToLower(apps[i].Name)
		right := strings.ToLower(apps[j].Name)
		if left != right {
			return left < right
		}
		return apps[i].Path < apps[j].Path
	})
	if len(apps) > 120 {
		return apps[:120]
	}
	return apps
}

func noopApplication() EditorAppInfo {
	return EditorAppInfo{
		ID:        editorNoneAppID,
		Name:      "不打开",
		Kind:      "none",
		Available: true,
	}
}

func systemDefaultApplication() EditorAppInfo {
	return EditorAppInfo{
		ID:        editorSystemAppID,
		Name:      "系统默认应用",
		Kind:      "system",
		Available: true,
	}
}

func knownEditorApplications() []EditorAppInfo {
	editors := editorSpecs()
	apps := make([]EditorAppInfo, 0, len(editors))
	for _, spec := range editors {
		executable := editorExecutable(spec)
		if executable == "" {
			continue
		}
		apps = append(apps, EditorAppInfo{
			ID:           spec.name,
			Name:         editorSpecDisplayName(spec),
			Path:         executable,
			Kind:         "editor",
			Available:    true,
			SupportsLine: true,
		})
	}
	return apps
}

func discoverLocalApplications() []EditorAppInfo {
	if runtime.GOOS != "darwin" {
		return nil
	}

	roots := []string{"/Applications"}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		roots = append(roots, filepath.Join(home, "Applications"))
	}

	apps := make([]EditorAppInfo, 0, 64)
	seenPath := make(map[string]bool)
	for _, root := range roots {
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil || !entry.IsDir() {
				return nil
			}
			if path != root && tooDeepApplicationPath(root, path) {
				return filepath.SkipDir
			}
			if !strings.HasSuffix(entry.Name(), ".app") {
				return nil
			}
			if seenPath[path] {
				return filepath.SkipDir
			}
			seenPath[path] = true
			apps = append(apps, EditorAppInfo{
				ID:        editorAppIDForPath(path),
				Name:      editorAppNameFromPath(path),
				Path:      path,
				Kind:      "app",
				Available: true,
			})
			return filepath.SkipDir
		})
	}
	return apps
}

func tooDeepApplicationPath(root string, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return false
	}
	return strings.Count(rel, string(os.PathSeparator)) > 2
}

func editorAppIDForPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return editorAppIDPrefix + path
}

func isCustomEditorAppID(id string) bool {
	return strings.HasPrefix(strings.TrimSpace(id), editorAppIDPrefix)
}

func editorDisplayName(id string) string {
	if isCustomEditorAppID(id) {
		return ""
	}
	id = normalizeEditorName(id)
	if id == "" {
		return ""
	}
	if id == editorNoneAppID {
		return "不打开"
	}
	if id == editorSystemAppID {
		return "系统默认应用"
	}
	for _, spec := range editorSpecs() {
		if spec.name == id {
			return editorSpecDisplayName(spec)
		}
	}
	return ""
}

func editorSpecDisplayName(spec editorSpec) string {
	if spec.displayName != "" {
		return spec.displayName
	}
	return spec.name
}

func editorAppNameFromPath(path string) string {
	name := filepath.Base(strings.TrimSpace(path))
	name = strings.TrimSuffix(name, ".app")
	if name == "" {
		return path
	}
	return name
}
