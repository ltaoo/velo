package desktopapp

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type editorSpec struct {
	name        string
	priority    int
	appPath     string
	cmdBuilder  func(executable string, file string, line string, col string) *exec.Cmd
	fallbackURL func(file string, line string, col string) string
}

func splitEditorLocation(value string) (string, string, string) {
	file := strings.TrimSpace(value)
	line := "1"
	col := "1"
	if file == "" {
		return file, line, col
	}

	if parsed, err := url.Parse(file); err == nil && parsed.RawQuery != "" {
		query := parsed.Query()
		if value := query.Get("line"); value != "" {
			line = editorPositionValue(value, line)
			query.Del("line")
		}
		if value := firstNonEmpty(query.Get("col"), query.Get("column")); value != "" {
			col = editorPositionValue(value, col)
			query.Del("col")
			query.Del("column")
		}
		parsed.RawQuery = query.Encode()
		file = parsed.String()
	}

	if path, suffixLine, suffixCol, ok := splitEditorPositionSuffix(file); ok {
		file = path
		line = editorPositionValue(suffixLine, line)
		col = editorPositionValue(suffixCol, col)
	}

	return file, line, col
}

func editorPositionValue(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if isPositiveInteger(value) {
		return value
	}
	if isPositiveInteger(fallback) {
		return fallback
	}
	return "1"
}

func splitEditorPositionSuffix(value string) (string, string, string, bool) {
	value = strings.TrimSpace(value)
	if value == "" || isEditorAssetReference(value) || isEditorOSSAssetURL(value) {
		return "", "", "", false
	}
	if hasNonLocalEditorScheme(value) {
		return "", "", "", false
	}

	lastColon := strings.LastIndex(value, ":")
	if lastColon <= 0 || lastColon == len(value)-1 {
		return "", "", "", false
	}
	lastPart := value[lastColon+1:]
	if !isPositiveInteger(lastPart) {
		return "", "", "", false
	}

	before := value[:lastColon]
	line := lastPart
	col := "1"
	path := before
	secondColon := strings.LastIndex(before, ":")
	if secondColon > 0 && secondColon < len(before)-1 && isPositiveInteger(before[secondColon+1:]) {
		path = before[:secondColon]
		line = before[secondColon+1:]
		col = lastPart
	}
	if strings.TrimSpace(path) == "" {
		return "", "", "", false
	}
	return path, line, col, true
}

func hasNonLocalEditorScheme(value string) bool {
	if isWindowsDrivePath(value) {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" {
		return false
	}
	scheme := strings.ToLower(parsed.Scheme)
	return scheme != "file" && scheme != "local"
}

func isWindowsDrivePath(value string) bool {
	if len(value) < 3 || value[1] != ':' {
		return false
	}
	drive := value[0]
	return ((drive >= 'a' && drive <= 'z') || (drive >= 'A' && drive <= 'Z')) && (value[2] == '\\' || value[2] == '/')
}

func isPositiveInteger(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return strings.TrimLeft(value, "0") != ""
}

func resolveEditorFileTarget(value string, rawSettings json.RawMessage, storePath string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("file is required")
	}

	if storageID, key, ok, err := parseEditorAssetReference(value); ok || err != nil {
		if err != nil {
			return "", err
		}
		return resolveEditorAssetPath(storageID, key, rawSettings, storePath)
	}

	if storageID, key, ok, err := parseEditorOSSAssetURL(value); ok || err != nil {
		if err != nil {
			return "", err
		}
		return resolveEditorAssetPath(storageID, key, rawSettings, storePath)
	}

	if path, ok, err := editorLocalURLPath(value); ok || err != nil {
		if err != nil {
			return "", err
		}
		value = path
	}

	return expandLocalPath(value), nil
}

func resolveEditorAssetPath(storageID string, key string, rawSettings json.RawMessage, storePath string) (string, error) {
	cleanKey := cleanOSSObjectPath(key)
	if cleanKey == "" {
		return "", fmt.Errorf("file path is required")
	}
	cfg, err := storedOSSConfig(rawSettings, storageID, storePath)
	if err != nil {
		return "", err
	}
	if !isLocalOSSConfig(cfg) {
		return "", fmt.Errorf("asset is not in local storage")
	}
	return localOSSObjectDiskPath(cfg, cleanKey)
}

func parseEditorAssetReference(value string) (string, string, bool, error) {
	if !isEditorAssetReference(value) {
		return "", "", false, nil
	}
	rest := strings.TrimPrefix(strings.TrimSpace(value), "@assets/")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return "", "", true, fmt.Errorf("invalid asset reference")
	}
	key, err := url.PathUnescape(parts[1])
	if err != nil {
		return "", "", true, err
	}
	return parts[0], key, true, nil
}

func isEditorAssetReference(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), "@assets/")
}

func parseEditorOSSAssetURL(value string) (string, string, bool, error) {
	if !isEditorOSSAssetURL(value) {
		return "", "", false, nil
	}
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", "", true, err
	}
	query := parsed.Query()
	storageID := firstNonEmpty(query.Get("storageId"), query.Get("storageID"), query.Get("id"))
	key := firstNonEmpty(query.Get("path"), query.Get("key"))
	if strings.TrimSpace(key) == "" {
		return "", "", true, fmt.Errorf("file path is required")
	}
	return storageID, key, true, nil
}

func isEditorOSSAssetURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	return err == nil && parsed.Path == "/api/oss/assets"
}

func editorLocalURLPath(value string) (string, bool, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", false, err
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "file" && scheme != "local" {
		return "", false, nil
	}

	path := parsed.Path
	if parsed.Host != "" && !strings.EqualFold(parsed.Host, "localhost") {
		if runtime.GOOS == "windows" {
			path = parsed.Host + parsed.Path
		} else {
			path = string(os.PathSeparator) + parsed.Host + parsed.Path
		}
	}
	if path == "" {
		path = parsed.Opaque
	}
	if path == "" {
		return "", true, fmt.Errorf("file path is required")
	}
	decoded, err := url.PathUnescape(path)
	if err != nil {
		return "", true, err
	}
	return decoded, true, nil
}

func openFileInEditor(file string, line string, col string, preferredEditor string) error {
	absoluteFile := expandLocalPath(file)
	if !filepath.IsAbs(absoluteFile) && !isWindowsDrivePath(absoluteFile) {
		cwd, err := os.Getwd()
		if err == nil {
			absoluteFile = filepath.Join(cwd, absoluteFile)
		}
	}
	info, err := os.Stat(absoluteFile)
	if err != nil {
		return fmt.Errorf("file not found: %s", absoluteFile)
	}
	if info.IsDir() {
		return fmt.Errorf("folder cannot be opened in editor: %s", absoluteFile)
	}

	spec, executable, err := chooseEditor(preferredEditor)
	if err != nil {
		return err
	}

	line = editorPositionValue(line, "1")
	col = editorPositionValue(col, "1")
	cmd := spec.cmdBuilder(executable, absoluteFile, line, col)
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		if spec.fallbackURL == nil || runtime.GOOS != "darwin" {
			return fmt.Errorf("failed to launch editor: %w", err)
		}
		fallback := exec.Command("open", spec.fallbackURL(absoluteFile, line, col))
		fallback.Env = os.Environ()
		if fallbackErr := fallback.Start(); fallbackErr != nil {
			return fmt.Errorf("failed to launch editor via URL scheme: %w", fallbackErr)
		}
	}
	return nil
}

func chooseEditor(preferredEditor string) (*editorSpec, string, error) {
	editors := editorSpecs()
	preferredEditor = normalizeEditorName(preferredEditor)
	if preferredEditor != "" {
		for i := range editors {
			if editors[i].name != preferredEditor {
				continue
			}
			if executable := editorExecutable(editors[i]); executable != "" {
				return &editors[i], executable, nil
			}
			break
		}
	}

	for _, envName := range []string{os.Getenv("EDITOR"), os.Getenv("GIT_EDITOR")} {
		fields := strings.Fields(envName)
		if len(fields) == 0 {
			continue
		}
		base := normalizeEditorName(filepath.Base(fields[0]))
		if base == "" {
			continue
		}
		for i := range editors {
			if editors[i].name != base {
				continue
			}
			if executable := editorExecutable(editors[i]); executable != "" {
				return &editors[i], executable, nil
			}
		}
	}

	var chosen *editorSpec
	chosenExecutable := ""
	for i := range editors {
		executable := editorExecutable(editors[i])
		if executable == "" {
			continue
		}
		if chosen == nil || editors[i].priority > chosen.priority {
			chosen = &editors[i]
			chosenExecutable = executable
		}
	}
	if chosen == nil {
		return nil, "", fmt.Errorf("no editor found")
	}
	return chosen, chosenExecutable, nil
}

func editorSpecs() []editorSpec {
	return []editorSpec{
		{
			name:     "trae",
			priority: 10,
			appPath:  "/Applications/Trae.app/Contents/MacOS/trae",
			cmdBuilder: func(executable string, file string, line string, col string) *exec.Cmd {
				return exec.Command(executable, file)
			},
			fallbackURL: func(file string, line string, col string) string {
				return fmt.Sprintf("trae://file/%s:%s:%s", file, line, col)
			},
		},
		{
			name:     "cursor",
			priority: 10,
			appPath:  "/Applications/Cursor.app/Contents/MacOS/cursor",
			cmdBuilder: func(executable string, file string, line string, col string) *exec.Cmd {
				return exec.Command(executable, "--goto", fmt.Sprintf("%s:%s:%s", file, line, col))
			},
			fallbackURL: func(file string, line string, col string) string {
				return fmt.Sprintf("cursor://file/%s:%s:%s", file, line, col)
			},
		},
		{
			name:     "code",
			priority: 10,
			appPath:  "/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code",
			cmdBuilder: func(executable string, file string, line string, col string) *exec.Cmd {
				return exec.Command(executable, "-g", fmt.Sprintf("%s:%s:%s", file, line, col))
			},
		},
		{
			name:     "code-insiders",
			priority: 10,
			appPath:  "/Applications/Visual Studio Code - Insiders.app/Contents/Resources/app/bin/code-insiders",
			cmdBuilder: func(executable string, file string, line string, col string) *exec.Cmd {
				return exec.Command(executable, "-g", fmt.Sprintf("%s:%s:%s", file, line, col))
			},
		},
		{
			name:     "webstorm",
			priority: 5,
			appPath:  "/Applications/WebStorm.app/Contents/MacOS/webstorm",
			cmdBuilder: func(executable string, file string, line string, col string) *exec.Cmd {
				return exec.Command(executable, "--line", line, file)
			},
		},
		{
			name:     "idea",
			priority: 5,
			appPath:  "/Applications/IntelliJ IDEA.app/Contents/MacOS/idea",
			cmdBuilder: func(executable string, file string, line string, col string) *exec.Cmd {
				return exec.Command(executable, "--line", line, file)
			},
		},
		{
			name:     "vim",
			priority: 3,
			cmdBuilder: func(executable string, file string, line string, col string) *exec.Cmd {
				return exec.Command(executable, "+"+line, file)
			},
		},
		{
			name:     "nvim",
			priority: 3,
			cmdBuilder: func(executable string, file string, line string, col string) *exec.Cmd {
				return exec.Command(executable, "+"+line, file)
			},
		},
		{
			name:     "emacs",
			priority: 2,
			cmdBuilder: func(executable string, file string, line string, col string) *exec.Cmd {
				return exec.Command(executable, "+"+line, file)
			},
		},
	}
}

func editorExecutable(spec editorSpec) string {
	if path, err := exec.LookPath(spec.name); err == nil {
		return path
	}
	if runtime.GOOS == "darwin" && spec.appPath != "" {
		if _, err := os.Stat(spec.appPath); err == nil {
			return spec.appPath
		}
	}
	return ""
}

func normalizeEditorName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "vscode", "vs-code", "visual-studio-code":
		return "code"
	default:
		return value
	}
}
