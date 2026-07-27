package fileserver

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileInfo represents a file or directory entry.
type FileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	ModTime string `json:"modTime"`
	IsDir   bool   `json:"isDir"`
	Ext     string `json:"ext"`
}

// SortBy names a sort field.
type SortBy string

const (
	SortByName    SortBy = "name"
	SortBySize    SortBy = "size"
	SortByModTime SortBy = "modTime"
	SortByExt     SortBy = "ext"
)

// SortOption controls result ordering.
type SortOption struct {
	By        SortBy // sort field, defaults to SortByName
	Ascending bool   // true = ascending, false = descending
}

// FetchFilesOption configures the file listing behaviour.
type FetchFilesOption struct {
	// Dir is the directory to list. Defaults to user home directory.
	Dir string
	// Search enables fuzzy matching on file names (case-insensitive).
	Search string
	// Ignore is a list of glob patterns for names to skip (via filepath.Match).
	// Hidden files (starting with ".") are always ignored in recursive mode.
	Ignore []string
	// Accept is a list of extension or glob patterns that files must match.
	// When empty everything is accepted. e.g. [".go", ".md", "*.txt"]
	Accept []string
	// Recursive enables recursive descent into subdirectories.
	Recursive bool
	// Depth limits recursion (0 means no limit, used only when Recursive is true).
	Depth int
	// Sort controls result ordering. nil means dirs-first, name ascending.
	Sort *SortOption
	// Offset and Limit control pagination. Limit <= 0 means no limit.
	Offset, Limit int
}

// FetchFiles lists files under opt.Dir, applying ignore/accept filters,
// optional fuzzy search, and pagination. It returns the page slice and the
// total number of matched entries before pagination.
func FetchFiles(opt FetchFilesOption) ([]FileInfo, int, error) {
	if opt.Dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, 0, err
		}
		opt.Dir = home
	}
	all, err := collectFiles(opt.Dir, opt.Dir, opt, 0)
	if err != nil {
		return nil, 0, err
	}
	total := len(all)
	page := paginate(all, opt.Offset, opt.Limit)
	return page, total, nil
}

// FetchCommonDirs returns commonly-used directory paths (Home, Desktop,
// Documents, Downloads, /Applications).
func FetchCommonDirs() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		home,
		filepath.Join(home, "Desktop"),
		filepath.Join(home, "Documents"),
		filepath.Join(home, "Downloads"),
		"/Applications",
	}
}

// FuzzyMatch checks whether query matches name (case-insensitive). It returns
// matchSequential when every character of query appears in name in order,
// matchSubstring when query is a contiguous substring, and matchNone
// otherwise.
func FuzzyMatch(query, name string) (MatchKind, bool) {
	ql := strings.ToLower(query)
	nl := strings.ToLower(name)

	qi := 0
	for ni := 0; ni < len(nl) && qi < len(ql); ni++ {
		if nl[ni] == ql[qi] {
			qi++
		}
	}
	if qi == len(ql) {
		return MatchSequential, true
	}
	if strings.Contains(nl, ql) {
		return MatchSubstring, true
	}
	return MatchNone, false
}

// MatchKind describes the quality of a fuzzy match.
type MatchKind int

const (
	MatchNone       MatchKind = 0
	MatchSubstring  MatchKind = 1
	MatchSequential MatchKind = 2
)

// ---------------------------------------------------------------------------
// internal helpers
// ---------------------------------------------------------------------------

func collectFiles(baseDir, displayBase string, opt FetchFilesOption, currentDepth int) ([]FileInfo, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	var sequential []FileInfo
	var substring []FileInfo

	hasSearch := opt.Search != ""

	for _, e := range entries {
		name := e.Name()

		// --- ignore check ---
		if shouldIgnore(name, e.IsDir(), opt.Ignore, opt.Recursive) {
			continue
		}

		// --- accept check ---
		if !e.IsDir() && !acceptMatch(name, opt.Accept) {
			continue
		}

		// --- fuzzy search ---
		mk := MatchSequential
		if hasSearch {
			var ok bool
			mk, ok = FuzzyMatch(opt.Search, name)
			if !ok {
				// recurse into unmatched directories if recursive
				if e.IsDir() && opt.Recursive && withinDepth(currentDepth, opt.Depth) {
					childFiles, _ := collectFiles(
						filepath.Join(baseDir, name),
						filepath.Join(displayBase, name),
						opt, currentDepth+1,
					)
					for _, cf := range childFiles {
						cfMk, _ := FuzzyMatch(opt.Search, cf.Name)
						if cfMk == MatchSequential {
							sequential = append(sequential, cf)
						} else {
							substring = append(substring, cf)
						}
					}
				}
				continue
			}
		}

		info, err := e.Info()
		if err != nil {
			continue
		}
		ext := ""
		if !e.IsDir() {
			ext = strings.ToLower(filepath.Ext(name))
		}
		fi := FileInfo{
			Name:    name,
			Path:    filepath.Join(displayBase, name),
			Size:    info.Size(),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
			IsDir:   e.IsDir(),
			Ext:     ext,
		}
		if mk >= MatchSequential {
			sequential = append(sequential, fi)
		} else {
			substring = append(substring, fi)
		}

		// recursion for matched directories
		if e.IsDir() && opt.Recursive && withinDepth(currentDepth, opt.Depth) {
			childFiles, _ := collectFiles(
				filepath.Join(baseDir, name),
				filepath.Join(displayBase, name),
				opt, currentDepth+1,
			)
			for _, cf := range childFiles {
				if hasSearch {
					cfMk, _ := FuzzyMatch(opt.Search, cf.Name)
					if cfMk == MatchSequential {
						sequential = append(sequential, cf)
					} else {
						substring = append(substring, cf)
					}
				} else {
					sequential = append(sequential, cf)
				}
			}
		}
	}

	sortResults(sequential, opt)
	sortResults(substring, opt)

	out := make([]FileInfo, 0, len(sequential)+len(substring))
	out = append(out, sequential...)
	out = append(out, substring...)
	return out, nil
}

func shouldIgnore(name string, isDir bool, ignorePatterns []string, recursive bool) bool {
	// always skip hidden entries in recursive mode
	if recursive && strings.HasPrefix(name, ".") {
		return true
	}
	for _, pat := range ignorePatterns {
		if matched, _ := filepath.Match(pat, name); matched {
			return true
		}
	}
	return false
}

func acceptMatch(name string, accept []string) bool {
	if len(accept) == 0 {
		return true
	}
	ext := filepath.Ext(name)
	for _, a := range accept {
		if a == ext {
			return true
		}
		if matched, _ := filepath.Match(a, name); matched {
			return true
		}
	}
	return false
}

func withinDepth(current, max int) bool {
	if max <= 0 {
		return true // no limit
	}
	return current < max
}

func paginate(files []FileInfo, offset, limit int) []FileInfo {
	if offset >= len(files) {
		return nil
	}
	end := offset + limit
	if limit <= 0 || end > len(files) {
		end = len(files)
	}
	return files[offset:end]
}

func sortResults(files []FileInfo, opt FetchFilesOption) {
	by := SortByName
	asc := true

	if opt.Sort != nil {
		if opt.Sort.By != "" {
			by = opt.Sort.By
		}
		asc = opt.Sort.Ascending
	}
	sort.Slice(files, func(i, j int) bool {
		// directories always grouped first
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		ai, aj := files[i], files[j]
		switch by {
		case SortBySize:
			if asc {
				return ai.Size < aj.Size
			}
			return ai.Size > aj.Size
		case SortByModTime:
			if asc {
				return ai.ModTime < aj.ModTime
			}
			return ai.ModTime > aj.ModTime
		case SortByExt:
			if ai.Ext != aj.Ext {
				if asc {
					return ai.Ext < aj.Ext
				}
				return ai.Ext > aj.Ext
			}
			fallthrough
		default: // SortByName
			if asc {
				return strings.ToLower(ai.Name) < strings.ToLower(aj.Name)
			}
			return strings.ToLower(ai.Name) > strings.ToLower(aj.Name)
		}
	})
}
