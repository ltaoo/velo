package desktopapp

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"example/simple/internal/desktopapp/windowing"

	"github.com/ltaoo/velo"
	"github.com/ltaoo/velo/store"
	"github.com/rs/zerolog"
)

const openWindowsStoreKey = "desktop.openWindows"
const windowSessionsStoreKey = "desktop.windowSessions"

const (
	persistedWindowKindOpenWindow = "open_window"
	persistedWindowKindMemoWindow = "memo_window"
)

type PersistedOpenWindow struct {
	EntryPage string             `json:"entryPage"`
	Fixed     bool               `json:"fixed"`
	Frameless bool               `json:"frameless,omitempty"`
	Height    int                `json:"height"`
	Kind      string             `json:"kind"`
	MemoID    string             `json:"memoId,omitempty"`
	Name      string             `json:"name"`
	Pathname  string             `json:"pathname"`
	Payload   *MemoWindowPayload `json:"payload,omitempty"`
	State     json.RawMessage    `json:"state,omitempty"`
	Title     string             `json:"title"`
	UpdatedAt string             `json:"updatedAt,omitempty"`
	Width     int                `json:"width"`
	X         int                `json:"x"`
	Y         int                `json:"y"`
}

type persistedOpenWindowFile struct {
	Windows []PersistedOpenWindow `json:"windows"`
}

func rememberWindowSpec(store *store.Store, spec windowing.WindowSpec) error {
	if store == nil || !shouldPersistOpenWindowName(spec.Name) {
		return nil
	}
	return upsertPersistedOpenWindow(store, PersistedOpenWindow{
		EntryPage: spec.EntryPage,
		Height:    spec.Height,
		Kind:      persistedWindowKindOpenWindow,
		Name:      spec.Name,
		Pathname:  spec.Pathname,
		Title:     spec.Title,
		Width:     spec.Width,
	})
}

func rememberMemoWindow(store *store.Store, memoID string, payload MemoWindowPayload) error {
	if store == nil {
		return nil
	}
	memoID = strings.TrimSpace(memoID)
	if memoID == "" {
		return fmt.Errorf("memo id is required")
	}
	name := memoWindowName(memoID)
	if !shouldPersistOpenWindowName(name) {
		return nil
	}
	return upsertPersistedOpenWindow(store, PersistedOpenWindow{
		EntryPage: "memo-window.html",
		Fixed:     payload.Fixed,
		Frameless: true,
		Height:    560,
		Kind:      persistedWindowKindMemoWindow,
		MemoID:    memoID,
		Name:      name,
		Pathname:  memoWindowPathname(memoID, payload.Fixed),
		Payload:   &payload,
		Title:     "Memo",
		Width:     460,
	})
}

func restorePersistedOpenWindows(b *velo.Box, logger *zerolog.Logger) {
	if b == nil || b.Store == nil {
		return
	}
	onClose := forgetPersistedOpenWindowOnClose(b.Store, logger)
	file := loadPersistedOpenWindows(b.Store)
	for _, item := range file.Windows {
		if !shouldPersistOpenWindowName(item.Name) {
			continue
		}
		restoreWindowFrameFromSession(b.Store, item)

		switch item.Kind {
		case persistedWindowKindMemoWindow:
			payload, ok := restoredMemoWindowPayload(item, logger)
			if !ok {
				if err := forgetPersistedOpenWindow(b.Store, item.Name); err != nil && logger != nil {
					logger.Warn().Err(err).Str("window", item.Name).Msg("failed to forget stale memo window")
				}
				continue
			}
			memoWindowCache.Lock()
			memoWindowCache.items[item.MemoID] = payload
			memoWindowCache.Unlock()
			pathname := memoWindowPathname(item.MemoID, payload.Fixed)
			if strings.TrimSpace(item.Pathname) != "" {
				pathname = pathnameWithFixed(item.Pathname, payload.Fixed)
			}
			b.OpenWindow(&velo.VeloWebviewOpt{
				Name:       item.Name,
				Title:      firstNonEmpty(item.Title, "Memo"),
				Pathname:   pathname,
				Width:      positiveOr(item.Width, 460),
				Height:     positiveOr(item.Height, 560),
				Frameless:  item.Frameless,
				EntryPage:  firstNonEmpty(item.EntryPage, "memo-window.html"),
				FrontendFS: appAssets.FrontendFS,
				OnClose:    onClose,
			})
		default:
			pathname := pathnameWithFixed(item.Pathname, item.Fixed)
			b.OpenWindow(&velo.VeloWebviewOpt{
				Name:       item.Name,
				Title:      firstNonEmpty(item.Title, "App"),
				Pathname:   pathname,
				Width:      positiveOr(item.Width, 760),
				Height:     positiveOr(item.Height, 640),
				Frameless:  item.Frameless,
				EntryPage:  firstNonEmpty(item.EntryPage, "index.html"),
				FrontendFS: appAssets.FrontendFS,
				OnClose:    onClose,
			})
		}
	}
}

func restoredMemoWindowPayload(item PersistedOpenWindow, logger *zerolog.Logger) (MemoWindowPayload, bool) {
	memoID := strings.TrimSpace(item.MemoID)
	if memoID == "" {
		memoID = memoIDFromMemoWindowName(item.Name)
	}
	if memoID == "" {
		return MemoWindowPayload{}, false
	}
	if payload, err := latestMemoWindowPayload(memoID, item.Fixed); err == nil {
		return payload, true
	} else if logger != nil {
		logger.Warn().Err(err).Str("memoId", memoID).Msg("failed to load latest memo for restored window")
	}
	if item.Payload == nil || len(item.Payload.Memo) == 0 || !json.Valid(item.Payload.Memo) {
		return MemoWindowPayload{}, false
	}
	payload := *item.Payload
	payload.Fixed = item.Fixed
	payload.Memos = memoWindowMemosPayload(payload.Memo, payload.Memos)
	return payload, true
}

func latestMemoWindowPayload(memoID string, fixed bool) (MemoWindowPayload, error) {
	ctx, err := requireActiveVault()
	if err != nil {
		return MemoWindowPayload{}, err
	}
	memos, err := listVaultMemos(ctx)
	if err != nil {
		return MemoWindowPayload{}, err
	}
	for _, memo := range memos {
		if memo.ID != memoID {
			continue
		}
		memoRaw, err := json.Marshal(memo)
		if err != nil {
			return MemoWindowPayload{}, err
		}
		memosRaw, err := json.Marshal(memos)
		if err != nil {
			return MemoWindowPayload{}, err
		}
		return MemoWindowPayload{Fixed: fixed, Memo: memoRaw, Memos: memosRaw}, nil
	}
	return MemoWindowPayload{}, fmt.Errorf("memo %q not found", memoID)
}

func loadPersistedMemoWindowPayload(store *store.Store, memoID string) (MemoWindowPayload, bool) {
	memoID = strings.TrimSpace(memoID)
	if store == nil || memoID == "" {
		return MemoWindowPayload{}, false
	}
	file := loadPersistedOpenWindows(store)
	for _, item := range file.Windows {
		if item.Kind != persistedWindowKindMemoWindow || item.MemoID != memoID {
			continue
		}
		payload, ok := restoredMemoWindowPayload(item, nil)
		return payload, ok
	}
	return MemoWindowPayload{}, false
}

func savePersistedWindowSession(store *store.Store, item PersistedOpenWindow) error {
	item.Name = strings.TrimSpace(item.Name)
	if store == nil || !shouldPersistOpenWindowName(item.Name) {
		return nil
	}
	if item.Kind == "" {
		item.Kind = persistedWindowKindOpenWindow
	}
	item.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)

	file := loadPersistedOpenWindows(store)
	replaced := false
	for i := range file.Windows {
		if file.Windows[i].Name != item.Name {
			continue
		}
		item = mergePersistedWindowSession(file.Windows[i], item)
		item.Pathname = pathnameWithFixed(item.Pathname, item.Fixed)
		file.Windows[i] = item
		replaced = true
		break
	}
	if !replaced {
		item.Pathname = pathnameWithFixed(item.Pathname, item.Fixed)
		file.Windows = append(file.Windows, item)
	}
	return savePersistedOpenWindows(store, file)
}

func mergePersistedWindowSession(existing PersistedOpenWindow, next PersistedOpenWindow) PersistedOpenWindow {
	if next.EntryPage == "" {
		next.EntryPage = existing.EntryPage
	}
	if next.Kind == "" {
		next.Kind = existing.Kind
	}
	if next.MemoID == "" {
		next.MemoID = existing.MemoID
	}
	if next.Pathname == "" {
		next.Pathname = existing.Pathname
	}
	if next.Payload == nil {
		next.Payload = existing.Payload
	}
	if len(next.State) == 0 {
		next.State = existing.State
	}
	if next.Title == "" {
		next.Title = existing.Title
	}
	if next.Width <= 0 {
		next.Width = existing.Width
	}
	if next.Height <= 0 {
		next.Height = existing.Height
	}
	if !next.Frameless {
		next.Frameless = existing.Frameless
	}
	return next
}

func updatePersistedOpenWindowFixed(store *store.Store, name string, fixed bool) error {
	name = strings.TrimSpace(name)
	if store == nil || name == "" {
		return nil
	}
	file := loadPersistedOpenWindows(store)
	changed := false
	for i := range file.Windows {
		if file.Windows[i].Name != name {
			continue
		}
		file.Windows[i].Fixed = fixed
		file.Windows[i].Pathname = pathnameWithFixed(file.Windows[i].Pathname, fixed)
		file.Windows[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		if file.Windows[i].Payload != nil {
			file.Windows[i].Payload.Fixed = fixed
		}
		if file.Windows[i].Kind == persistedWindowKindMemoWindow && file.Windows[i].MemoID != "" {
			memoWindowCache.Lock()
			payload, ok := memoWindowCache.items[file.Windows[i].MemoID]
			if ok {
				payload.Fixed = fixed
				memoWindowCache.items[file.Windows[i].MemoID] = payload
			}
			memoWindowCache.Unlock()
		}
		changed = true
		break
	}
	if !changed {
		return nil
	}
	return savePersistedOpenWindows(store, file)
}

func updatePersistedOpenWindowFrame(store *store.Store, name string, x int, y int, width int, height int, fixed *bool) error {
	name = strings.TrimSpace(name)
	if store == nil || name == "" {
		return nil
	}
	file := loadPersistedOpenWindows(store)
	changed := false
	for i := range file.Windows {
		if file.Windows[i].Name != name {
			continue
		}
		file.Windows[i].X = x
		file.Windows[i].Y = y
		file.Windows[i].Width = width
		file.Windows[i].Height = height
		if fixed != nil {
			file.Windows[i].Fixed = *fixed
			file.Windows[i].Pathname = pathnameWithFixed(file.Windows[i].Pathname, *fixed)
			if file.Windows[i].Payload != nil {
				file.Windows[i].Payload.Fixed = *fixed
			}
		}
		file.Windows[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		changed = true
		break
	}
	if !changed {
		return nil
	}
	return savePersistedOpenWindows(store, file)
}

func forgetPersistedOpenWindow(store *store.Store, name string) error {
	name = strings.TrimSpace(name)
	if store == nil || name == "" {
		return nil
	}
	file := loadPersistedOpenWindows(store)
	next := make([]PersistedOpenWindow, 0, len(file.Windows))
	changed := false
	for _, item := range file.Windows {
		if item.Name == name {
			changed = true
			continue
		}
		next = append(next, item)
	}
	if !changed {
		return nil
	}
	file.Windows = next
	return savePersistedOpenWindows(store, file)
}

func forgetPersistedOpenWindowOnClose(store *store.Store, logger *zerolog.Logger) func(string) {
	return func(name string) {
		name = strings.TrimSpace(name)
		if !shouldPersistOpenWindowName(name) {
			return
		}
		if err := forgetPersistedOpenWindow(store, name); err != nil && logger != nil {
			logger.Warn().Err(err).Str("window", name).Msg("failed to forget closed window session")
		}
	}
}

func upsertPersistedOpenWindow(store *store.Store, item PersistedOpenWindow) error {
	item.Name = strings.TrimSpace(item.Name)
	if store == nil || !shouldPersistOpenWindowName(item.Name) {
		return nil
	}
	if item.Kind == "" {
		item.Kind = persistedWindowKindOpenWindow
	}
	item.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	item.Pathname = pathnameWithFixed(item.Pathname, item.Fixed)

	file := loadPersistedOpenWindows(store)
	replaced := false
	for i := range file.Windows {
		if file.Windows[i].Name != item.Name {
			continue
		}
		existing := file.Windows[i]
		if item.Fixed == false {
			item.Fixed = existing.Fixed
			item.Pathname = pathnameWithFixed(item.Pathname, item.Fixed)
		}
		if len(item.State) == 0 {
			item.State = existing.State
		}
		if item.X == 0 && item.Y == 0 {
			item.X = existing.X
			item.Y = existing.Y
		}
		if existing.Width > 0 {
			item.Width = existing.Width
		}
		if existing.Height > 0 {
			item.Height = existing.Height
		}
		file.Windows[i] = item
		replaced = true
		break
	}
	if !replaced {
		file.Windows = append(file.Windows, item)
	}
	return savePersistedOpenWindows(store, file)
}

func loadPersistedOpenWindows(store *store.Store) persistedOpenWindowFile {
	if store == nil {
		return persistedOpenWindowFile{}
	}
	raw := store.Get(windowSessionsStoreKey)
	if raw == nil {
		raw = store.Get(openWindowsStoreKey)
	}
	if raw == nil {
		return persistedOpenWindowFile{}
	}
	var file persistedOpenWindowFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return persistedOpenWindowFile{}
	}
	next := make([]PersistedOpenWindow, 0, len(file.Windows))
	seen := make(map[string]bool)
	for _, item := range file.Windows {
		item.Name = strings.TrimSpace(item.Name)
		if !shouldPersistOpenWindowName(item.Name) || seen[item.Name] {
			continue
		}
		seen[item.Name] = true
		next = append(next, item)
	}
	file.Windows = next
	return file
}

func savePersistedOpenWindows(store *store.Store, file persistedOpenWindowFile) error {
	if store == nil {
		return nil
	}
	raw, err := json.Marshal(file)
	if err != nil {
		return err
	}
	return store.Set(windowSessionsStoreKey, json.RawMessage(raw))
}

func shouldPersistOpenWindowName(name string) bool {
	switch strings.TrimSpace(name) {
	case "", "default", "desktop", "vault-picker", "snippet-launcher":
		return false
	default:
		return true
	}
}

func persistedWindowFixed(store *store.Store, name string) bool {
	name = strings.TrimSpace(name)
	if store == nil || name == "" {
		return false
	}
	file := loadPersistedOpenWindows(store)
	for _, item := range file.Windows {
		if item.Name == name {
			return item.Fixed
		}
	}
	return false
}

func persistedWindowSession(store *store.Store, name string) (PersistedOpenWindow, bool) {
	name = strings.TrimSpace(name)
	if store == nil || name == "" {
		return PersistedOpenWindow{}, false
	}
	file := loadPersistedOpenWindows(store)
	for _, item := range file.Windows {
		if item.Name == name {
			return item, true
		}
	}
	return PersistedOpenWindow{}, false
}

func restoreWindowFrameFromSession(st *store.Store, item PersistedOpenWindow) {
	if st == nil || item.Name == "" || item.Width <= 0 || item.Height <= 0 {
		return
	}
	_ = st.SaveWindow(item.Name, &store.WindowState{
		X:      item.X,
		Y:      item.Y,
		Width:  item.Width,
		Height: item.Height,
	})
}

func memoWindowPathname(memoID string, fixed bool) string {
	params := url.Values{}
	params.Set("id", strings.TrimSpace(memoID))
	if fixed {
		params.Set("fixed", "1")
	}
	return "/memo-window?" + params.Encode()
}

func pathnameWithFixed(pathname string, fixed bool) string {
	pathname = strings.TrimSpace(pathname)
	if pathname == "" {
		pathname = "/settings"
	}
	parsed, err := url.Parse(pathname)
	if err != nil {
		return pathname
	}
	query := parsed.Query()
	if fixed {
		query.Set("fixed", "1")
	} else {
		query.Del("fixed")
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func memoIDFromMemoWindowName(name string) string {
	const prefix = "memo-window-"
	name = strings.TrimSpace(name)
	if !strings.HasPrefix(name, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(name, prefix))
}

func positiveOr(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
