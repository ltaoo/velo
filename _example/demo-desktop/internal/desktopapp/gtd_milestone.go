package desktopapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const vaultGTDMilestonesFileName = "milestones.json"

const (
	gtdMilestoneStatusPlanned   = "planned"
	gtdMilestoneStatusActive    = "active"
	gtdMilestoneStatusCompleted = "completed"
	gtdMilestoneStatusCancelled = "cancelled"
)

type GTDMilestoneFile struct {
	Milestones    []GTDMilestoneRecord `json:"milestones"`
	SchemaVersion int                  `json:"schemaVersion"`
}

type GTDMilestoneRecord struct {
	CompletedAt   string   `json:"completedAt,omitempty"`
	CreatedAt     string   `json:"createdAt"`
	ID            string   `json:"id"`
	ItemIDs       []string `json:"itemIds"`
	ProjectIDs    []string `json:"projectIds"`
	ReviewMemoID  string   `json:"reviewMemoId,omitempty"`
	SchemaVersion int      `json:"schemaVersion"`
	Status        string   `json:"status"`
	TargetAt      string   `json:"targetAt,omitempty"`
	TaskIDs       []string `json:"taskIds"`
	Title         string   `json:"title"`
	UpdatedAt     string   `json:"updatedAt"`
}

type GTDMilestoneCreateRequest struct {
	ItemIDs      []string `json:"itemIds"`
	ProjectIDs   []string `json:"projectIds"`
	ReviewMemoID string   `json:"reviewMemoId"`
	Status       string   `json:"status"`
	TargetAt     string   `json:"targetAt"`
	TaskIDs      []string `json:"taskIds"`
	Title        string   `json:"title"`
}

type GTDMilestoneUpdateRequest struct {
	ID           string    `json:"id"`
	ItemIDs      *[]string `json:"itemIds"`
	ProjectIDs   *[]string `json:"projectIds"`
	ReviewMemoID *string   `json:"reviewMemoId"`
	Status       *string   `json:"status"`
	TargetAt     *string   `json:"targetAt"`
	TaskIDs      *[]string `json:"taskIds"`
	Title        *string   `json:"title"`
}

func gtdMilestonesPath(ctx *VaultContext) string {
	return filepath.Join(ctx.VeloDir, vaultGTDMilestonesFileName)
}

func listVaultGTDMilestones(ctx *VaultContext) (GTDMilestoneFile, error) {
	return loadGTDMilestones(ctx)
}

func loadGTDMilestones(ctx *VaultContext) (GTDMilestoneFile, error) {
	raw, err := os.ReadFile(gtdMilestonesPath(ctx))
	if os.IsNotExist(err) {
		return GTDMilestoneFile{SchemaVersion: vaultSchemaVersion, Milestones: []GTDMilestoneRecord{}}, nil
	}
	if err != nil {
		return GTDMilestoneFile{}, fmt.Errorf("read milestones: %w", err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return GTDMilestoneFile{SchemaVersion: vaultSchemaVersion, Milestones: []GTDMilestoneRecord{}}, nil
	}
	var file GTDMilestoneFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return GTDMilestoneFile{}, fmt.Errorf("read milestones: %w", err)
	}
	return normalizeGTDMilestoneFile(file), nil
}

func saveGTDMilestones(ctx *VaultContext, file GTDMilestoneFile) error {
	file = normalizeGTDMilestoneFile(file)
	file.SchemaVersion = vaultSchemaVersion
	return writeJSONFileAtomic(gtdMilestonesPath(ctx), file)
}

func createVaultGTDMilestone(ctx *VaultContext, req GTDMilestoneCreateRequest) (GTDMilestoneRecord, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return GTDMilestoneRecord{}, fmt.Errorf("milestone title is required")
	}
	file, err := loadGTDMilestones(ctx)
	if err != nil {
		return GTDMilestoneRecord{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	milestone := GTDMilestoneRecord{
		CreatedAt:     now,
		ID:            newGTDMilestoneID(),
		ItemIDs:       normalizeGTDItemIDs(req.ItemIDs),
		ProjectIDs:    normalizeProjectIDs(req.ProjectIDs),
		ReviewMemoID:  strings.TrimSpace(req.ReviewMemoID),
		SchemaVersion: vaultSchemaVersion,
		Status:        normalizeGTDMilestoneStatus(req.Status),
		TargetAt:      normalizeTaskTime(req.TargetAt),
		TaskIDs:       normalizeTaskIDs(req.TaskIDs),
		Title:         title,
		UpdatedAt:     now,
	}
	if milestone.Status == gtdMilestoneStatusCompleted {
		milestone.CompletedAt = now
	}
	file.Milestones = append(file.Milestones, milestone)
	if err := saveGTDMilestones(ctx, file); err != nil {
		return GTDMilestoneRecord{}, err
	}
	return milestone, nil
}

func updateVaultGTDMilestone(ctx *VaultContext, req GTDMilestoneUpdateRequest) (GTDMilestoneRecord, error) {
	id := sanitizeGTDMilestoneID(req.ID)
	if id == "" {
		return GTDMilestoneRecord{}, fmt.Errorf("milestone id is required")
	}
	file, err := loadGTDMilestones(ctx)
	if err != nil {
		return GTDMilestoneRecord{}, err
	}
	for i, milestone := range file.Milestones {
		if milestone.ID != id {
			continue
		}
		if req.Title != nil {
			title := strings.TrimSpace(*req.Title)
			if title == "" {
				return GTDMilestoneRecord{}, fmt.Errorf("milestone title is required")
			}
			milestone.Title = title
		}
		if req.Status != nil {
			next := normalizeGTDMilestoneStatus(*req.Status)
			if milestone.Status != next {
				milestone.Status = next
				if next == gtdMilestoneStatusCompleted && milestone.CompletedAt == "" {
					milestone.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
				}
				if next != gtdMilestoneStatusCompleted {
					milestone.CompletedAt = ""
				}
			}
		}
		if req.TargetAt != nil {
			milestone.TargetAt = normalizeTaskTime(*req.TargetAt)
		}
		if req.ItemIDs != nil {
			milestone.ItemIDs = normalizeGTDItemIDs(*req.ItemIDs)
		}
		if req.ProjectIDs != nil {
			milestone.ProjectIDs = normalizeProjectIDs(*req.ProjectIDs)
		}
		if req.TaskIDs != nil {
			milestone.TaskIDs = normalizeTaskIDs(*req.TaskIDs)
		}
		if req.ReviewMemoID != nil {
			milestone.ReviewMemoID = strings.TrimSpace(*req.ReviewMemoID)
		}
		milestone.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		file.Milestones[i] = milestone
		if err := saveGTDMilestones(ctx, file); err != nil {
			return GTDMilestoneRecord{}, err
		}
		return milestone, nil
	}
	return GTDMilestoneRecord{}, fmt.Errorf("milestone not found: %s", id)
}

func deleteVaultGTDMilestone(ctx *VaultContext, id string) error {
	id = sanitizeGTDMilestoneID(id)
	if id == "" {
		return fmt.Errorf("milestone id is required")
	}
	file, err := loadGTDMilestones(ctx)
	if err != nil {
		return err
	}
	next := make([]GTDMilestoneRecord, 0, len(file.Milestones))
	found := false
	for _, milestone := range file.Milestones {
		if milestone.ID == id {
			found = true
			continue
		}
		next = append(next, milestone)
	}
	if !found {
		return fmt.Errorf("milestone not found: %s", id)
	}
	file.Milestones = next
	return saveGTDMilestones(ctx, file)
}

func normalizeGTDMilestoneFile(file GTDMilestoneFile) GTDMilestoneFile {
	if file.SchemaVersion == 0 {
		file.SchemaVersion = vaultSchemaVersion
	}
	seen := map[string]bool{}
	milestones := make([]GTDMilestoneRecord, 0, len(file.Milestones))
	for _, milestone := range file.Milestones {
		milestone = normalizeGTDMilestoneRecord(milestone)
		if milestone.ID == "" || milestone.Title == "" || seen[milestone.ID] {
			continue
		}
		seen[milestone.ID] = true
		milestones = append(milestones, milestone)
	}
	sort.SliceStable(milestones, func(i, j int) bool {
		leftTarget := parseMemoTime(milestones[i].TargetAt)
		rightTarget := parseMemoTime(milestones[j].TargetAt)
		if !leftTarget.IsZero() && !rightTarget.IsZero() && !leftTarget.Equal(rightTarget) {
			return leftTarget.Before(rightTarget)
		}
		left := parseMemoTime(firstNonEmpty(milestones[i].UpdatedAt, milestones[i].CreatedAt))
		right := parseMemoTime(firstNonEmpty(milestones[j].UpdatedAt, milestones[j].CreatedAt))
		if left.Equal(right) {
			return milestones[i].ID > milestones[j].ID
		}
		return left.After(right)
	})
	file.Milestones = milestones
	return file
}

func normalizeGTDMilestoneRecord(milestone GTDMilestoneRecord) GTDMilestoneRecord {
	milestone.ID = sanitizeGTDMilestoneID(milestone.ID)
	milestone.Title = strings.TrimSpace(milestone.Title)
	milestone.Status = normalizeGTDMilestoneStatus(milestone.Status)
	milestone.TargetAt = normalizeTaskTime(milestone.TargetAt)
	milestone.ItemIDs = normalizeGTDItemIDs(milestone.ItemIDs)
	milestone.ProjectIDs = normalizeProjectIDs(milestone.ProjectIDs)
	milestone.TaskIDs = normalizeTaskIDs(milestone.TaskIDs)
	milestone.ReviewMemoID = strings.TrimSpace(milestone.ReviewMemoID)
	if milestone.SchemaVersion == 0 {
		milestone.SchemaVersion = vaultSchemaVersion
	}
	return milestone
}

func normalizeGTDMilestoneStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case gtdMilestoneStatusActive:
		return gtdMilestoneStatusActive
	case gtdMilestoneStatusCompleted:
		return gtdMilestoneStatusCompleted
	case gtdMilestoneStatusCancelled:
		return gtdMilestoneStatusCancelled
	default:
		return gtdMilestoneStatusPlanned
	}
}

func newGTDMilestoneID() string {
	return "milestone_" + time.Now().UTC().Format("20060102T150405") + "_" + randomVaultSuffix()
}

func sanitizeGTDMilestoneID(value string) string {
	return sanitizeProjectID(value)
}

func normalizeGTDItemIDs(values []string) []string {
	ids := make([]string, 0, len(values))
	for _, value := range values {
		id := sanitizeGTDItemID(value)
		if id != "" {
			ids = append(ids, id)
		}
	}
	return uniqueStrings(ids)
}

func normalizeProjectIDs(values []string) []string {
	ids := make([]string, 0, len(values))
	for _, value := range values {
		id := sanitizeProjectID(value)
		if id != "" {
			ids = append(ids, id)
		}
	}
	return uniqueStrings(ids)
}
