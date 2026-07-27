package fileserver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createTempDirWithFiles(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(dir, "notes.md"), []byte("world"), 0644)
	os.WriteFile(filepath.Join(dir, "script.go"), []byte("package main"), 0644)

	sub := filepath.Join(dir, "subdir")
	os.Mkdir(sub, 0755)
	os.WriteFile(filepath.Join(sub, "deep_file.json"), []byte("{}"), 0644)

	return dir
}

// ---- FetchFiles ----

func TestFetchFiles_FlatListing(t *testing.T) {
	dir := createTempDirWithFiles(t)

	files, total, err := FetchFiles(FetchFilesOption{Dir: dir})
	assert.NoError(t, err)
	assert.Equal(t, 4, total) // 3 files + 1 subdir
	assert.Len(t, files, 4)

	names := make(map[string]bool)
	for _, f := range files {
		names[f.Name] = true
	}
	assert.True(t, names["readme.txt"])
	assert.True(t, names["notes.md"])
	assert.True(t, names["script.go"])
	assert.True(t, names["subdir"])
}

func TestFetchFiles_Recursive(t *testing.T) {
	dir := createTempDirWithFiles(t)

	files, total, err := FetchFiles(FetchFilesOption{Dir: dir, Recursive: true})
	assert.NoError(t, err)
	assert.Equal(t, 5, total) // 3 root files + 1 subdir + 1 deep file
	assert.Len(t, files, 5)
}

func TestFetchFiles_Search(t *testing.T) {
	dir := createTempDirWithFiles(t)

	files, total, err := FetchFiles(FetchFilesOption{Dir: dir, Search: "readme"})
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "readme.txt", files[0].Name)
}

func TestFetchFiles_SearchRecursive(t *testing.T) {
	dir := createTempDirWithFiles(t)

	files, total, err := FetchFiles(FetchFilesOption{Dir: dir, Search: "deep", Recursive: true})
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "deep_file.json", files[0].Name)
}

func TestFetchFiles_Accept(t *testing.T) {
	dir := createTempDirWithFiles(t)

	files, total, err := FetchFiles(FetchFilesOption{Dir: dir, Accept: []string{".go"}})
	assert.NoError(t, err)
	// dirs pass through Accept filter; 2 = script.go + subdir
	assert.Equal(t, 2, total)
	assert.Equal(t, "script.go", files[1].Name)
}

func TestFetchFiles_AcceptMulti(t *testing.T) {
	dir := createTempDirWithFiles(t)

	_, total, err := FetchFiles(FetchFilesOption{Dir: dir, Accept: []string{".go", ".md"}})
	assert.NoError(t, err)
	// dirs pass through; 3 = subdir + notes.md + script.go
	assert.Equal(t, 3, total)
}

func TestFetchFiles_Ignore(t *testing.T) {
	dir := createTempDirWithFiles(t)

	files, total, err := FetchFiles(FetchFilesOption{Dir: dir, Ignore: []string{"*.txt"}})
	assert.NoError(t, err)
	assert.Equal(t, 3, total) // .md, .go, subdir

	for _, f := range files {
		assert.NotEqual(t, "readme.txt", f.Name)
	}
}

func TestFetchFiles_Pagination(t *testing.T) {
	dir := createTempDirWithFiles(t)

	page, total, err := FetchFiles(FetchFilesOption{Dir: dir, Offset: 0, Limit: 2})
	assert.NoError(t, err)
	assert.Equal(t, 4, total)
	assert.Len(t, page, 2)
}

func TestFetchFiles_PaginationOffset(t *testing.T) {
	dir := createTempDirWithFiles(t)

	page, total, err := FetchFiles(FetchFilesOption{Dir: dir, Offset: 2, Limit: 10})
	assert.NoError(t, err)
	assert.Equal(t, 4, total)
	assert.Len(t, page, 2)
}

func TestFetchFiles_DepthLimit(t *testing.T) {
	dir := createTempDirWithFiles(t)

	_, total, err := FetchFiles(FetchFilesOption{Dir: dir, Recursive: true, Depth: 1})
	assert.NoError(t, err)
	// Depth=1 recurses 1 level down: root (4) + subdir contents (1) = 5
	assert.Equal(t, 5, total)
}

func TestFetchFiles_NonExistentDir(t *testing.T) {
	_, _, err := FetchFiles(FetchFilesOption{Dir: "/nonexistent/path"})
	assert.Error(t, err)
}

// ---- Sort ----

func TestFetchFiles_SortByNameDesc(t *testing.T) {
	dir := createTempDirWithFiles(t)

	files, _, err := FetchFiles(FetchFilesOption{
		Dir: dir,
		Sort: &SortOption{By: SortByName, Ascending: false},
	})
	assert.NoError(t, err)
	// dirs first, then files descending
	assert.Equal(t, "subdir", files[0].Name)
	assert.Equal(t, "script.go", files[1].Name)
	assert.Equal(t, "readme.txt", files[2].Name)
	assert.Equal(t, "notes.md", files[3].Name)
}

func TestFetchFiles_SortBySizeAsc(t *testing.T) {
	dir := createTempDirWithFiles(t)

	files, _, err := FetchFiles(FetchFilesOption{
		Dir:  dir,
		Sort: &SortOption{By: SortBySize, Ascending: true},
	})
	assert.NoError(t, err)
	assert.Equal(t, "subdir", files[0].Name) // dir always first
	assert.Len(t, files, 4)
}

func TestFetchFiles_SortBySizeDesc(t *testing.T) {
	dir := createTempDirWithFiles(t)

	files, _, err := FetchFiles(FetchFilesOption{
		Dir:  dir,
		Sort: &SortOption{By: SortBySize, Ascending: false},
	})
	assert.NoError(t, err)
	assert.Equal(t, "subdir", files[0].Name) // dir always first
	assert.Equal(t, "script.go", files[1].Name)
}

func TestFetchFiles_SortByExt(t *testing.T) {
	dir := createTempDirWithFiles(t)

	files, _, err := FetchFiles(FetchFilesOption{
		Dir:  dir,
		Sort: &SortOption{By: SortByExt, Ascending: true},
	})
	assert.NoError(t, err)
	assert.Equal(t, "subdir", files[0].Name) // dir first (no ext)
	// files sorted by ext asc: .go, .md, .txt
	assert.Equal(t, ".go", files[1].Ext)
	assert.Equal(t, ".md", files[2].Ext)
	assert.Equal(t, ".txt", files[3].Ext)
}

func TestFetchFiles_SortByNameAsc(t *testing.T) {
	dir := createTempDirWithFiles(t)

	files, _, err := FetchFiles(FetchFilesOption{
		Dir:  dir,
		Sort: &SortOption{By: SortByName, Ascending: true},
	})
	assert.NoError(t, err)
	assert.Equal(t, "subdir", files[0].Name)
	assert.Equal(t, "notes.md", files[1].Name)
	assert.Equal(t, "readme.txt", files[2].Name)
	assert.Equal(t, "script.go", files[3].Name)
}

// ---- FuzzyMatch ----

func TestFuzzyMatch_Sequential(t *testing.T) {
	kind, matched := FuzzyMatch("rdm", "readme.txt")
	assert.True(t, matched)
	assert.Equal(t, MatchSequential, kind)
}

func TestFuzzyMatch_Substring(t *testing.T) {
	kind, matched := FuzzyMatch("read", "readme.txt")
	assert.True(t, matched)
	assert.Equal(t, MatchSequential, kind)
}

func TestFuzzyMatch_CaseInsensitive(t *testing.T) {
	kind, matched := FuzzyMatch("README", "readme.txt")
	assert.True(t, matched)
	assert.Equal(t, MatchSequential, kind)
}

func TestFuzzyMatch_NoMatch(t *testing.T) {
	_, matched := FuzzyMatch("xyz", "readme.txt")
	assert.False(t, matched)
}

func TestFuzzyMatch_EmptyQuery(t *testing.T) {
	kind, matched := FuzzyMatch("", "readme.txt")
	assert.True(t, matched)
	assert.Equal(t, MatchSequential, kind)
}

// ---- FetchCommonDirs ----

func TestFetchCommonDirs(t *testing.T) {
	dirs := FetchCommonDirs()
	assert.GreaterOrEqual(t, len(dirs), 1)

	home, err := os.UserHomeDir()
	if err == nil {
		assert.Contains(t, dirs, home)
	}
}
