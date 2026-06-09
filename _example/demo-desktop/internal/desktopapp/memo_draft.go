package desktopapp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const vaultMemoDraftsFileName = "memo-drafts.json"

type MemoDraftFile struct {
	SchemaVersion int               `json:"schemaVersion"`
	Drafts        []MemoDraftRecord `json:"drafts"`
}

type MemoDraftRecord struct {
	BaseUpdatedAt string `json:"baseUpdatedAt,omitempty"`
	Content       string `json:"content"`
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	MemoID        string `json:"memoId,omitempty"`
	ProjectID     string `json:"projectId,omitempty"`
	UpdatedAt     string `json:"updatedAt"`
	Visibility    string `json:"visibility"`
}

type MemoDraftUpsertRequest struct {
	BaseUpdatedAt string `json:"baseUpdatedAt,omitempty"`
	Content       string `json:"content"`
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	MemoID        string `json:"memoId,omitempty"`
	ProjectID     string `json:"projectId,omitempty"`
	Visibility    string `json:"visibility"`
}

type MemoDraftDeleteRequest struct {
	ID string `json:"id"`
}

func memoDraftsPath(ctx *VaultContext) string {
	return filepath.Join(ctx.VeloDir, vaultMemoDraftsFileName)
}

func listVaultMemoDrafts(ctx *VaultContext) ([]MemoDraftRecord, error) {
	file, err := loadMemoDraftFile(ctx)
	if err != nil {
		return nil, err
	}
	return file.Drafts, nil
}

func upsertVaultMemoDraft(ctx *VaultContext, req MemoDraftUpsertRequest) (MemoDraftRecord, error) {
	draft, err := normalizeMemoDraftRequest(ctx, req)
	if err != nil {
		return MemoDraftRecord{}, err
	}

	file, err := loadMemoDraftFile(ctx)
	if err != nil {
		return MemoDraftRecord{}, err
	}

	replaced := false
	for index, item := range file.Drafts {
		if item.ID != draft.ID {
			continue
		}
		file.Drafts[index] = draft
		replaced = true
		break
	}
	if !replaced {
		file.Drafts = append(file.Drafts, draft)
	}
	sortMemoDrafts(file.Drafts)
	if err := writeMemoDraftFile(ctx, file); err != nil {
		return MemoDraftRecord{}, err
	}
	return draft, nil
}

func deleteVaultMemoDraft(ctx *VaultContext, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("draft id is required")
	}

	file, err := loadMemoDraftFile(ctx)
	if err != nil {
		return err
	}

	next := make([]MemoDraftRecord, 0, len(file.Drafts))
	for _, draft := range file.Drafts {
		if draft.ID != id {
			next = append(next, draft)
		}
	}
	if len(next) == len(file.Drafts) {
		return nil
	}
	file.Drafts = next
	return writeMemoDraftFile(ctx, file)
}

func loadMemoDraftFile(ctx *VaultContext) (MemoDraftFile, error) {
	path := memoDraftsPath(ctx)
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return MemoDraftFile{SchemaVersion: vaultSchemaVersion, Drafts: []MemoDraftRecord{}}, nil
	}
	if err != nil {
		return MemoDraftFile{}, fmt.Errorf("read memo drafts: %w", err)
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return MemoDraftFile{SchemaVersion: vaultSchemaVersion, Drafts: []MemoDraftRecord{}}, nil
	}

	var file MemoDraftFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return MemoDraftFile{}, fmt.Errorf("read memo drafts: %w", err)
	}
	file.SchemaVersion = vaultSchemaVersion
	file.Drafts = normalizeMemoDrafts(file.Drafts)
	return file, nil
}

func writeMemoDraftFile(ctx *VaultContext, file MemoDraftFile) error {
	file.SchemaVersion = vaultSchemaVersion
	file.Drafts = normalizeMemoDrafts(file.Drafts)
	return writeJSONFileAtomic(memoDraftsPath(ctx), file)
}

func normalizeMemoDraftRequest(ctx *VaultContext, req MemoDraftUpsertRequest) (MemoDraftRecord, error) {
	id := strings.TrimSpace(req.ID)
	if id == "" {
		return MemoDraftRecord{}, fmt.Errorf("draft id is required")
	}

	kind := normalizeMemoDraftKind(req.Kind)
	if kind == "" {
		return MemoDraftRecord{}, fmt.Errorf("draft kind is required")
	}

	memoID := strings.TrimSpace(req.MemoID)
	if kind == "memo-edit" && memoID == "" {
		return MemoDraftRecord{}, fmt.Errorf("draft memo id is required")
	}

	projectID, err := validateMemoProjectID(ctx, req.ProjectID)
	if err != nil {
		return MemoDraftRecord{}, err
	}

	return MemoDraftRecord{
		BaseUpdatedAt: strings.TrimSpace(req.BaseUpdatedAt),
		Content:       normalizeMemoContent(req.Content),
		ID:            id,
		Kind:          kind,
		MemoID:        memoID,
		ProjectID:     projectID,
		UpdatedAt:     time.Now().UTC().Format(time.RFC3339Nano),
		Visibility:    normalizeMemoVisibility(req.Visibility),
	}, nil
}

func normalizeMemoDrafts(drafts []MemoDraftRecord) []MemoDraftRecord {
	next := make([]MemoDraftRecord, 0, len(drafts))
	seen := make(map[string]bool)
	for _, draft := range drafts {
		draft.ID = strings.TrimSpace(draft.ID)
		draft.Kind = normalizeMemoDraftKind(draft.Kind)
		if draft.ID == "" || draft.Kind == "" || seen[draft.ID] {
			continue
		}
		draft.MemoID = strings.TrimSpace(draft.MemoID)
		if draft.Kind == "memo-edit" && draft.MemoID == "" {
			continue
		}
		draft.BaseUpdatedAt = strings.TrimSpace(draft.BaseUpdatedAt)
		draft.Content = normalizeMemoContent(draft.Content)
		draft.ProjectID = strings.TrimSpace(draft.ProjectID)
		draft.Visibility = normalizeMemoVisibility(draft.Visibility)
		draft.UpdatedAt = strings.TrimSpace(draft.UpdatedAt)
		if draft.UpdatedAt == "" {
			draft.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		}
		seen[draft.ID] = true
		next = append(next, draft)
	}
	sortMemoDrafts(next)
	return next
}

func normalizeMemoDraftKind(kind string) string {
	switch strings.TrimSpace(kind) {
	case "composer", "memo-edit":
		return strings.TrimSpace(kind)
	default:
		return ""
	}
}

func sortMemoDrafts(drafts []MemoDraftRecord) {
	sort.SliceStable(drafts, func(i, j int) bool {
		left, leftErr := time.Parse(time.RFC3339Nano, drafts[i].UpdatedAt)
		right, rightErr := time.Parse(time.RFC3339Nano, drafts[j].UpdatedAt)
		if leftErr == nil && rightErr == nil && !left.Equal(right) {
			return left.After(right)
		}
		return drafts[i].ID < drafts[j].ID
	})
}
