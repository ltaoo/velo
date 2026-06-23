package desktopapp

import (
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/ltaoo/velo"
)

var snippetLauncherShortcuts = []string{
	"ControlLeft+ShiftLeft+Space",
	"MetaLeft+ShiftLeft+Space",
}

type CodeSnippet struct {
	Aliases      []string `json:"aliases"`
	Code         string   `json:"code"`
	Command      string   `json:"command"`
	CommentID    string   `json:"commentId,omitempty"`
	CommentPath  string   `json:"commentPath,omitempty"`
	CreatedAt    string   `json:"createdAt"`
	EndLine      int      `json:"endLine"`
	ID           string   `json:"id"`
	Language     string   `json:"language"`
	Marked       bool     `json:"marked"`
	MemoID       string   `json:"memoId"`
	MemoPath     string   `json:"memoPath"`
	MemoTitle    string   `json:"memoTitle"`
	ProjectID    string   `json:"projectId,omitempty"`
	SourceMemoID string   `json:"sourceMemoId,omitempty"`
	SourceType   string   `json:"sourceType,omitempty"`
	SourceText   string   `json:"sourceText"`
	StartLine    int      `json:"startLine"`
	Title        string   `json:"title"`
	UpdatedAt    string   `json:"updatedAt"`
	Visibility   string   `json:"visibility"`
}

type StoredLink struct {
	CommentID    string `json:"commentId,omitempty"`
	CommentPath  string `json:"commentPath,omitempty"`
	CreatedAt    string `json:"createdAt"`
	ID           string `json:"id"`
	Label        string `json:"label"`
	Line         int    `json:"line"`
	MemoID       string `json:"memoId"`
	MemoPath     string `json:"memoPath"`
	MemoTitle    string `json:"memoTitle"`
	ProjectID    string `json:"projectId,omitempty"`
	SourceMemoID string `json:"sourceMemoId,omitempty"`
	SourceType   string `json:"sourceType,omitempty"`
	SourceText   string `json:"sourceText"`
	Syntax       string `json:"syntax"`
	UpdatedAt    string `json:"updatedAt"`
	URL          string `json:"url"`
	Visibility   string `json:"visibility"`
}

type snippetMarker struct {
	Aliases []string
	Title   string
}

type activeCodeBlock struct {
	FenceLine    string
	Language     string
	Lines        []string
	Marker       *snippetMarker
	OpeningFence memoCodeFence
	StartIndex   int
}

type scoredSnippet struct {
	item  CodeSnippet
	score int
}

type scoredLink struct {
	item  StoredLink
	score int
}

var (
	codeFenceLineRe     = regexp.MustCompile("^(`{3,}|~{3,})\\s*(.*)$")
	snippetMarkerTextRe = regexp.MustCompile(`(?i)(?:^|\s)(#?snippet|snip|code[-\s]?snippet|代码片段|片段)(?:\s*[:：-]\s*|\s+|$)(.*)$`)
	aliasPrefixRe       = regexp.MustCompile(`(?i)^(?:alias|aliases|aka|as|别名|缩写)\s*[:：=]\s*`)
	memoEmbedImageRe    = regexp.MustCompile(`!\[\[([^\]]+)\]\]`)
	memoEmbedRe         = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	markdownImageRe     = regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`)
	markdownLinkRe      = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	markdownHTTPLinkRe  = regexp.MustCompile(`(!?)\[([^\]]*)\]\(([^)]+)\)`)
	rawHTTPURLRe        = regexp.MustCompile(`(?i)\bhttps?://[^\s<>"` + "`" + `]+`)
)

func registerSnippetRoutes(b *velo.Box) {
	b.Get("/api/snippets/search", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		limit := snippetSearchLimit(c.Query("limit"))
		items, err := searchVaultSnippets(ctx, c.Query("q"), limit)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"items": items})
	})

	b.Get("/api/links/search", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		limit := snippetSearchLimit(c.Query("limit"))
		items, err := searchVaultLinks(ctx, c.Query("q"), limit)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"items": items})
	})

	b.Get("/api/snippet-launcher/open", func(c *velo.BoxContext) interface{} {
		openSnippetLauncher(b)
		return c.Ok(velo.H{"success": true})
	})
}

func openSnippetLauncher(b *velo.Box) {
	b.OpenWindow(&velo.VeloWebviewOpt{
		Name:                 "snippet-launcher",
		Title:                "Command",
		Pathname:             "/snippet-launcher",
		Width:                720,
		Height:               60,
		Frameless:            true,
		HideTrafficLights:    true,
		NonActivating:        true,
		PreserveStateOnFocus: true,
		EntryPage:            "snippet-launcher.html",
		FrontendFS:           appAssets.FrontendFS,
	})
}

func snippetSearchLimit(value string) int {
	limit, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || limit <= 0 {
		return 12
	}
	if limit > 50 {
		return 50
	}
	return limit
}

func searchVaultSnippets(ctx *VaultContext, query string, limit int) ([]CodeSnippet, error) {
	memos, err := listVaultMemos(ctx)
	if err != nil {
		return nil, err
	}
	comments, err := listVaultMemoComments(ctx, "")
	if err != nil {
		return nil, err
	}
	memoByID := memoRecordsByID(memos)

	directive, term := parseSnippetSearchQuery(query)
	if !directive {
		return []CodeSnippet{}, nil
	}
	needle := strings.ToLower(strings.TrimSpace(term))

	scored := []scoredSnippet{}
	for _, memo := range memos {
		for _, item := range collectMemoCodeSnippets(memo) {
			score, ok := scoreSnippetSearch(item, needle)
			if !ok {
				continue
			}
			scored = append(scored, scoredSnippet{item: item, score: score})
		}
	}
	for _, comment := range comments {
		parent, ok := memoByID[comment.MemoID]
		if !ok {
			continue
		}
		for _, item := range collectMemoCommentCodeSnippets(comment, parent) {
			score, ok := scoreSnippetSearch(item, needle)
			if !ok {
				continue
			}
			scored = append(scored, scoredSnippet{item: item, score: score})
		}
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		left := snippetSortTime(scored[i].item)
		right := snippetSortTime(scored[j].item)
		if !left.Equal(right) {
			return left.After(right)
		}
		if scored[i].item.MemoID != scored[j].item.MemoID {
			return scored[i].item.MemoID > scored[j].item.MemoID
		}
		return scored[i].item.StartLine < scored[j].item.StartLine
	})

	if limit <= 0 || limit > len(scored) {
		limit = len(scored)
	}
	items := make([]CodeSnippet, 0, limit)
	for _, item := range scored[:limit] {
		items = append(items, item.item)
	}
	return items, nil
}

func searchVaultLinks(ctx *VaultContext, query string, limit int) ([]StoredLink, error) {
	memos, err := listVaultMemos(ctx)
	if err != nil {
		return nil, err
	}
	comments, err := listVaultMemoComments(ctx, "")
	if err != nil {
		return nil, err
	}
	memoByID := memoRecordsByID(memos)

	directive, term := parseLinkSearchQuery(query)
	if !directive {
		return []StoredLink{}, nil
	}
	needle := strings.ToLower(strings.TrimSpace(term))

	scored := []scoredLink{}
	for _, memo := range memos {
		for _, item := range collectMemoLinks(memo) {
			score, ok := scoreLinkSearch(item, needle)
			if !ok {
				continue
			}
			scored = append(scored, scoredLink{item: item, score: score})
		}
	}
	for _, comment := range comments {
		parent, ok := memoByID[comment.MemoID]
		if !ok {
			continue
		}
		for _, item := range collectMemoCommentLinks(comment, parent) {
			score, ok := scoreLinkSearch(item, needle)
			if !ok {
				continue
			}
			scored = append(scored, scoredLink{item: item, score: score})
		}
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		left := linkSortTime(scored[i].item)
		right := linkSortTime(scored[j].item)
		if !left.Equal(right) {
			return left.After(right)
		}
		if scored[i].item.MemoID != scored[j].item.MemoID {
			return scored[i].item.MemoID > scored[j].item.MemoID
		}
		return scored[i].item.Line < scored[j].item.Line
	})

	if limit <= 0 || limit > len(scored) {
		limit = len(scored)
	}
	items := make([]StoredLink, 0, limit)
	for _, item := range scored[:limit] {
		items = append(items, item.item)
	}
	return items, nil
}

func parseSnippetSearchQuery(query string) (bool, string) {
	raw := strings.TrimSpace(query)
	if raw == "" {
		return false, ""
	}
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return false, ""
	}
	if strings.EqualFold(fields[0], "snippet") || strings.EqualFold(fields[0], "snip") {
		return true, strings.TrimSpace(strings.TrimPrefix(raw, fields[0]))
	}
	return false, raw
}

func parseLinkSearchQuery(query string) (bool, string) {
	raw := strings.TrimSpace(query)
	if raw == "" {
		return false, ""
	}
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return false, ""
	}
	switch strings.ToLower(fields[0]) {
	case "link", "links", "url", "链接":
		return true, strings.TrimSpace(strings.TrimPrefix(raw, fields[0]))
	default:
		return false, raw
	}
}

func scoreSnippetSearch(item CodeSnippet, needle string) (int, bool) {
	if needle == "" {
		if item.Marked {
			return 100, true
		}
		return 10, true
	}

	searchText := strings.ToLower(strings.Join([]string{
		item.Title,
		item.Command,
		strings.Join(item.Aliases, " "),
		item.Language,
		item.Code,
		item.SourceText,
		item.MemoTitle,
		item.MemoPath,
		item.CommentPath,
		item.SourceType,
		item.Visibility,
		item.ProjectID,
	}, " "))
	if !matchesSnippetQuery(searchText, needle) {
		return 0, false
	}

	score := 0
	command := strings.ToLower(item.Command)
	title := strings.ToLower(item.Title)
	language := strings.ToLower(item.Language)
	if item.Marked {
		score += 100
	}
	if command == needle {
		score += 90
	} else if command != "" && strings.HasPrefix(command, needle) {
		score += 60
	} else if command != "" && strings.Contains(command, needle) {
		score += 35
	}
	if title == needle {
		score += 70
	} else if title != "" && strings.Contains(title, needle) {
		score += 30
	}
	for _, alias := range item.Aliases {
		value := strings.ToLower(alias)
		if value == needle {
			score += 80
		} else if strings.HasPrefix(value, needle) {
			score += 50
		} else if strings.Contains(value, needle) {
			score += 25
		}
	}
	if language == needle {
		score += 20
	}
	if strings.Contains(strings.ToLower(item.Code), needle) {
		score += 10
	}
	if strings.Contains(strings.ToLower(item.SourceText), needle) || strings.Contains(strings.ToLower(item.MemoTitle), needle) {
		score += 6
	}
	return score, true
}

func scoreLinkSearch(item StoredLink, needle string) (int, bool) {
	if needle == "" {
		return 10, true
	}

	searchText := strings.ToLower(strings.Join([]string{
		item.Label,
		item.URL,
		item.MemoTitle,
		item.MemoPath,
		item.CommentPath,
		item.SourceType,
		item.ProjectID,
		item.Visibility,
	}, " "))
	if !matchesSnippetQuery(searchText, needle) {
		return 0, false
	}

	score := 0
	label := strings.ToLower(item.Label)
	linkURL := strings.ToLower(item.URL)
	if label == needle {
		score += 90
	} else if label != "" && strings.HasPrefix(label, needle) {
		score += 60
	} else if label != "" && strings.Contains(label, needle) {
		score += 35
	}
	if linkURL == needle {
		score += 80
	} else if strings.Contains(linkURL, needle) {
		score += 30
	}
	if strings.Contains(strings.ToLower(item.SourceText), needle) || strings.Contains(strings.ToLower(item.MemoTitle), needle) {
		score += 8
	}
	return score, true
}

func matchesSnippetQuery(haystack string, needle string) bool {
	needle = strings.TrimSpace(strings.ToLower(needle))
	if needle == "" {
		return true
	}
	if strings.Contains(haystack, needle) {
		return true
	}
	terms := strings.Fields(needle)
	if len(terms) == 0 {
		return true
	}
	for _, term := range terms {
		if !strings.Contains(haystack, term) {
			return false
		}
	}
	return true
}

func snippetSortTime(item CodeSnippet) time.Time {
	if item.UpdatedAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, item.UpdatedAt); err == nil {
			return t
		}
	}
	if item.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, item.CreatedAt); err == nil {
			return t
		}
	}
	return time.Time{}
}

func linkSortTime(item StoredLink) time.Time {
	if item.UpdatedAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, item.UpdatedAt); err == nil {
			return t
		}
	}
	if item.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, item.CreatedAt); err == nil {
			return t
		}
	}
	return time.Time{}
}

func memoRecordsByID(memos []MemoRecord) map[string]MemoRecord {
	byID := map[string]MemoRecord{}
	for _, memo := range memos {
		if strings.TrimSpace(memo.ID) == "" {
			continue
		}
		byID[memo.ID] = memo
	}
	return byID
}

func collectMemoCodeSnippets(memo MemoRecord) []CodeSnippet {
	lines := memoContentLines(memo.Content)
	items := []CodeSnippet{}
	var active *activeCodeBlock

	for index, line := range lines {
		fence, hasFence := parseMemoCodeFenceLine(line)
		if !hasFence {
			if active != nil {
				active.Lines = append(active.Lines, line)
			}
			continue
		}

		if active != nil {
			if memoCodeFenceCloses(fence, active.OpeningFence) {
				items = append(items, codeSnippetView(memo, lines, *active, index))
				active = nil
			} else {
				active.Lines = append(active.Lines, line)
			}
			continue
		}

		language, marker := parseCodeFence(line)
		if marker == nil {
			marker = snippetMarkerFromPreviousLines(lines, index)
		}
		active = &activeCodeBlock{
			FenceLine:    line,
			Language:     language,
			Lines:        []string{},
			Marker:       marker,
			OpeningFence: fence,
			StartIndex:   index,
		}
	}

	if active != nil {
		items = append(items, codeSnippetView(memo, lines, *active, len(lines)-1))
	}
	return items
}

func collectMemoCommentCodeSnippets(comment MemoCommentRecord, parent MemoRecord) []CodeSnippet {
	source := memoRecordForCommentSource(comment, parent)
	items := collectMemoCodeSnippets(source)
	for index := range items {
		items[index] = codeSnippetWithCommentSource(items[index], comment, parent)
	}
	return items
}

func collectMemoLinks(memo MemoRecord) []StoredLink {
	lines := memoContentLines(memo.Content)
	items := []StoredLink{}
	var activeFence *memoCodeFence

	for index, line := range lines {
		fence, hasFence := parseMemoCodeFenceLine(line)
		if activeFence != nil {
			if hasFence && memoCodeFenceCloses(fence, *activeFence) {
				activeFence = nil
			}
			continue
		}
		if hasFence {
			activeFence = &fence
			continue
		}
		items = append(items, collectMemoLineLinks(memo, lines, maskMemoInlineCode(line), index)...)
	}
	return items
}

func collectMemoCommentLinks(comment MemoCommentRecord, parent MemoRecord) []StoredLink {
	source := memoRecordForCommentSource(comment, parent)
	items := collectMemoLinks(source)
	for index := range items {
		items[index] = storedLinkWithCommentSource(items[index], comment, parent)
	}
	return items
}

func memoRecordForCommentSource(comment MemoCommentRecord, parent MemoRecord) MemoRecord {
	return MemoRecord{
		Content:    comment.Content,
		CreatedAt:  comment.CreatedAt,
		ID:         comment.ID,
		Path:       comment.Path,
		ProjectID:  parent.ProjectID,
		References: comment.References,
		Tags:       comment.Tags,
		UpdatedAt:  comment.UpdatedAt,
		Visibility: parent.Visibility,
	}
}

func codeSnippetWithCommentSource(item CodeSnippet, comment MemoCommentRecord, parent MemoRecord) CodeSnippet {
	item.CommentID = comment.ID
	item.CommentPath = comment.Path
	item.MemoID = parent.ID
	item.MemoPath = parent.Path
	item.MemoTitle = memoTitleText(parent) + " / 评论"
	item.ProjectID = parent.ProjectID
	item.SourceMemoID = parent.ID
	item.SourceType = "comment"
	item.Visibility = parent.Visibility
	return item
}

func storedLinkWithCommentSource(item StoredLink, comment MemoCommentRecord, parent MemoRecord) StoredLink {
	item.CommentID = comment.ID
	item.CommentPath = comment.Path
	item.MemoID = parent.ID
	item.MemoPath = parent.Path
	item.MemoTitle = memoTitleText(parent) + " / 评论"
	item.ProjectID = parent.ProjectID
	item.SourceMemoID = parent.ID
	item.SourceType = "comment"
	item.Visibility = parent.Visibility
	return item
}

func collectMemoLineLinks(memo MemoRecord, lines []string, line string, lineIndex int) []StoredLink {
	items := []StoredLink{}
	markdownRanges := [][2]int{}

	for _, match := range markdownHTTPLinkRe.FindAllStringSubmatchIndex(line, -1) {
		if len(match) < 8 {
			continue
		}
		markdownRanges = append(markdownRanges, [2]int{match[0], match[1]})
		marker := line[match[2]:match[3]]
		if marker == "!" {
			continue
		}
		label := strings.TrimSpace(line[match[4]:match[5]])
		target, ok := normalizeStoredHTTPURL(line[match[6]:match[7]])
		if !ok {
			continue
		}
		items = append(items, storedLinkView(memo, lines, lineIndex, len(items), "markdown", label, target))
	}

	for _, match := range rawHTTPURLRe.FindAllStringIndex(line, -1) {
		if len(match) != 2 || memoByteRangeContains(markdownRanges, match[0]) {
			continue
		}
		target, ok := normalizeStoredHTTPURL(cleanRawStoredURL(line[match[0]:match[1]]))
		if !ok {
			continue
		}
		items = append(items, storedLinkView(memo, lines, lineIndex, len(items), "raw", "", target))
	}

	return items
}

func storedLinkView(memo MemoRecord, lines []string, lineIndex int, index int, syntax string, label string, target string) StoredLink {
	title := compactSnippetText(label, 120)
	if title == "" {
		title = linkDisplayName(target)
	}
	return StoredLink{
		CreatedAt:    memo.CreatedAt,
		ID:           memo.ID + ":" + strconv.Itoa(lineIndex) + ":" + strconv.Itoa(index) + ":link",
		Label:        title,
		Line:         lineIndex + 1,
		MemoID:       memo.ID,
		MemoPath:     memo.Path,
		MemoTitle:    memoTitleText(memo),
		ProjectID:    memo.ProjectID,
		SourceMemoID: memo.ID,
		SourceType:   "memo",
		SourceText:   sourceTextFromMemoLines(lines, lineIndex, "仅包含链接的 memo"),
		Syntax:       syntax,
		UpdatedAt:    memo.UpdatedAt,
		URL:          target,
		Visibility:   memo.Visibility,
	}
}

func codeSnippetView(memo MemoRecord, lines []string, block activeCodeBlock, endIndex int) CodeSnippet {
	contentMarker := snippetMarkerFromFirstCodeLine(block.Lines)
	marker := block.Marker
	codeLines := block.Lines
	if marker == nil && contentMarker != nil {
		marker = contentMarker
		if len(codeLines) > 0 {
			codeLines = codeLines[1:]
		}
	}

	code := strings.Join(codeLines, "\n")
	language := strings.TrimSpace(block.Language)
	title := ""
	aliases := []string{}
	if marker != nil {
		title = marker.Title
		aliases = marker.Aliases
	}
	command := snippetCommand(title, aliases, language)
	if title == "" {
		if language != "" {
			title = language + " 代码片段"
		} else {
			title = "代码片段"
		}
	}
	return CodeSnippet{
		Aliases:      aliases,
		Code:         code,
		Command:      command,
		CreatedAt:    memo.CreatedAt,
		EndLine:      endIndex + 1,
		ID:           memo.ID + ":" + strconv.Itoa(block.StartIndex) + ":" + strconv.Itoa(endIndex) + ":code",
		Language:     language,
		Marked:       marker != nil,
		MemoID:       memo.ID,
		MemoPath:     memo.Path,
		MemoTitle:    memoTitleText(memo),
		ProjectID:    memo.ProjectID,
		SourceMemoID: memo.ID,
		SourceType:   "memo",
		SourceText:   sourceTextFromMemoLines(lines, block.StartIndex, "仅包含代码块的 memo"),
		StartLine:    block.StartIndex + 1,
		Title:        title,
		UpdatedAt:    memo.UpdatedAt,
		Visibility:   memo.Visibility,
	}
}

func parseCodeFence(line string) (string, *snippetMarker) {
	fence, ok := parseMemoCodeFenceLine(line)
	if !ok {
		return "", nil
	}
	return codeBlockLanguageFromInfo(fence.Info), snippetMarkerFromText(fence.Info, false)
}

func codeBlockLanguageFromInfo(info string) string {
	clean := strings.TrimSpace(strings.NewReplacer("{", " ", "}", " ").Replace(info))
	if clean == "" {
		return ""
	}
	tokens := strings.Fields(clean)
	if len(tokens) == 0 {
		return ""
	}
	first := tokens[0]
	if isSnippetMarkerToken(first) || isSnippetAttributeToken(first) {
		return ""
	}
	first = strings.TrimRightFunc(first, func(r rune) bool {
		return strings.ContainsRune(",:：;，；|", r)
	})
	return strings.TrimSpace(first)
}

func snippetMarkerFromPreviousLines(lines []string, lineIndex int) *snippetMarker {
	for index := lineIndex - 1; index >= 0; index-- {
		line := strings.TrimSpace(lines[index])
		if line == "" {
			continue
		}
		return snippetMarkerFromText(line, false)
	}
	return nil
}

func snippetMarkerFromFirstCodeLine(lines []string) *snippetMarker {
	if len(lines) == 0 {
		return nil
	}
	return snippetMarkerFromText(lines[0], true)
}

func snippetMarkerFromText(value string, allowCommentPrefix bool) *snippetMarker {
	text := normalizeSnippetMarkerText(value, allowCommentPrefix)
	if text == "" {
		return nil
	}
	match := snippetMarkerTextRe.FindStringSubmatch(text)
	if len(match) == 0 {
		return nil
	}
	return parseSnippetMarkerMeta(match[2])
}

func normalizeSnippetMarkerText(value string, allowCommentPrefix bool) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return ""
	}
	text = strings.TrimPrefix(text, "<!--")
	text = strings.TrimSuffix(text, "-->")
	text = strings.TrimPrefix(text, "/*")
	text = strings.TrimSuffix(text, "*/")
	text = strings.NewReplacer("{", " ", "}", " ").Replace(text)
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, ">")
	text = strings.TrimSpace(text)
	text = trimMarkdownListPrefix(text)
	text = trimMarkdownHeadingPrefix(text)
	if allowCommentPrefix {
		text = strings.TrimSpace(text)
		for _, prefix := range []string{"//", "#", "--", ";"} {
			if strings.HasPrefix(text, prefix) {
				text = strings.TrimSpace(strings.TrimPrefix(text, prefix))
				break
			}
		}
	}
	return strings.TrimSpace(text)
}

func parseSnippetMarkerMeta(value string) *snippetMarker {
	raw := strings.TrimSpace(value)
	parts := splitSnippetMeta(raw)
	title := ""
	if len(parts) > 0 {
		title = compactSnippetText(parts[0], 120)
	}
	aliases := []string{}
	if len(parts) > 1 {
		for _, part := range parts[1:] {
			aliases = append(aliases, splitAliasText(part)...)
		}
	}
	return &snippetMarker{
		Aliases: uniqueSnippetStrings(aliases),
		Title:   title,
	}
}

func splitSnippetMeta(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == '|' || r == ';' || r == '；' || r == ',' || r == '，'
	})
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		part := strings.TrimSpace(aliasPrefixRe.ReplaceAllString(field, ""))
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func splitAliasText(value string) []string {
	text := strings.TrimSpace(value)
	if text == "" {
		return nil
	}
	if hasWhitespace(text) && !isSimpleAliasText(text) {
		return []string{text}
	}
	return strings.Fields(text)
}

func hasWhitespace(value string) bool {
	for _, r := range value {
		if unicode.IsSpace(r) {
			return true
		}
	}
	return false
}

func isSimpleAliasText(value string) bool {
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' || unicode.IsSpace(r) {
			continue
		}
		return false
	}
	return true
}

func snippetCommand(title string, aliases []string, language string) string {
	for _, alias := range aliases {
		if strings.TrimSpace(alias) != "" {
			return strings.TrimSpace(alias)
		}
	}
	if title != "" {
		return slugSnippetCommand(title)
	}
	return slugSnippetCommand(language)
}

func slugSnippetCommand(value string) string {
	text := strings.TrimSpace(strings.ToLower(value))
	if text == "" {
		return ""
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if r == '_' || r == '-' || r == '.' {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if unicode.IsSpace(r) && !lastDash {
			builder.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func normalizeStoredHTTPURL(value string) (string, bool) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return "", false
	}
	for _, r := range raw {
		if r <= 0x20 || r == 0x7f {
			return "", false
		}
	}
	parsed, err := url.Parse(raw)
	if err != nil || !parsed.IsAbs() || strings.TrimSpace(parsed.Host) == "" {
		return "", false
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", false
	}
	parsed.Scheme = scheme
	return parsed.String(), true
}

func cleanRawStoredURL(value string) string {
	url := strings.TrimSpace(value)
	for strings.TrimRight(url, "),.;:!?，。；：！？") != url {
		url = strings.TrimRight(url, "),.;:!?，。；：！？")
	}
	return url
}

func linkDisplayName(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return compactSnippetText(value, 120)
	}
	label := parsed.Host
	if parsed.Path != "" && parsed.Path != "/" {
		label += parsed.Path
	}
	if label == "" {
		label = value
	}
	return compactSnippetText(label, 120)
}

func isSnippetMarkerToken(value string) bool {
	clean := strings.Trim(strings.ToLower(value), "{}:：,，;；|")
	switch clean {
	case "snippet", "#snippet", "snip", "codesnippet", "code-snippet", "code_snippet", "代码片段", "片段":
		return true
	default:
		return false
	}
}

func isSnippetAttributeToken(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(lower, "title=") ||
		strings.HasPrefix(lower, "alias=") ||
		strings.HasPrefix(lower, "aliases=") ||
		strings.HasPrefix(lower, "aka=") ||
		strings.HasPrefix(lower, "as=") ||
		strings.HasPrefix(lower, "别名=") ||
		strings.HasPrefix(lower, "缩写=")
}

func memoContentLines(content string) []string {
	return strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
}

func memoTitleText(memo MemoRecord) string {
	lines := memoContentLines(memo.Content)
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		title := cleanSnippetMemoLine(line)
		if title != "" {
			return compactSnippetText(title, 48)
		}
	}
	if strings.TrimSpace(memo.ID) != "" {
		return memo.ID
	}
	return "Untitled memo"
}

func sourceTextFromMemoLines(lines []string, lineIndex int, fallback string) string {
	for index := lineIndex - 1; index >= 0; index-- {
		clean := cleanSnippetMemoLine(lines[index])
		if clean != "" {
			return compactSnippetText(clean, 84)
		}
	}
	for _, line := range lines {
		clean := cleanSnippetMemoLine(line)
		if clean != "" {
			return compactSnippetText(clean, 84)
		}
	}
	return fallback
}

func cleanSnippetMemoLine(line string) string {
	text := strings.TrimSpace(line)
	text = trimMarkdownHeadingPrefix(text)
	text = strings.TrimPrefix(text, ">")
	text = strings.TrimSpace(text)
	text = trimMarkdownListPrefix(text)
	text = memoEmbedImageRe.ReplaceAllString(text, "$1")
	text = memoEmbedRe.ReplaceAllString(text, "$1")
	text = markdownImageRe.ReplaceAllString(text, "$1")
	text = markdownLinkRe.ReplaceAllString(text, "$1")
	text = strings.Map(func(r rune) rune {
		if strings.ContainsRune("`*_~", r) {
			return ' '
		}
		return r
	}, text)
	return strings.Join(strings.Fields(text), " ")
}

func trimMarkdownHeadingPrefix(text string) string {
	text = strings.TrimSpace(text)
	count := 0
	for count < len(text) && text[count] == '#' {
		count++
	}
	if count > 0 && count <= 6 && (count == len(text) || unicode.IsSpace(rune(text[count]))) {
		return strings.TrimSpace(text[count:])
	}
	return text
}

func trimMarkdownListPrefix(text string) string {
	text = strings.TrimSpace(text)
	if len(text) >= 2 && (text[0] == '-' || text[0] == '*' || text[0] == '+') && unicode.IsSpace(rune(text[1])) {
		return strings.TrimSpace(text[2:])
	}
	dot := strings.Index(text, ".")
	if dot > 0 {
		allDigits := true
		for _, r := range text[:dot] {
			if !unicode.IsDigit(r) {
				allDigits = false
				break
			}
		}
		if allDigits && dot+1 < len(text) && unicode.IsSpace(rune(text[dot+1])) {
			return strings.TrimSpace(text[dot+2:])
		}
	}
	return text
}

func compactSnippetText(value string, limit int) string {
	text := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if limit <= 0 || len([]rune(text)) <= limit {
		return text
	}
	runes := []rune(text)
	return string(runes[:limit-1]) + "..."
}

func uniqueSnippetStrings(items []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, item := range items {
		value := strings.TrimSpace(item)
		key := strings.ToLower(value)
		if value == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, value)
	}
	return out
}
