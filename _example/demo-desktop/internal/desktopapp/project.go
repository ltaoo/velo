package desktopapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type ProjectFile struct {
	ActiveProjectID string          `json:"activeProjectId"`
	Projects        []ProjectRecord `json:"projects"`
	SchemaVersion   int             `json:"schemaVersion"`
}

type ProjectRecord struct {
	Archived  bool   `json:"archived"`
	Color     string `json:"color"`
	CreatedAt string `json:"createdAt"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sortOrder"`
	UpdatedAt string `json:"updatedAt"`
}
type ProjectCreateRequest struct {
	Color string `json:"color"`
	Name  string `json:"name"`
}

type ProjectUpdateRequest struct {
	Archived  *bool   `json:"archived"`
	Color     *string `json:"color"`
	ID        string  `json:"id"`
	Name      *string `json:"name"`
	SortOrder *int    `json:"sortOrder"`
}

type ProjectActivateRequest struct {
	ProjectID string `json:"projectId"`
}

func loadVaultProjects(ctx *VaultContext) (ProjectFile, error) {
	path := filepath.Join(ctx.VeloDir, vaultProjectsFileName)
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return ProjectFile{SchemaVersion: vaultSchemaVersion, Projects: []ProjectRecord{}}, nil
	}
	if err != nil {
		return ProjectFile{}, fmt.Errorf("read projects: %w", err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return ProjectFile{SchemaVersion: vaultSchemaVersion, Projects: []ProjectRecord{}}, nil
	}
	var file ProjectFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return ProjectFile{}, fmt.Errorf("read projects: %w", err)
	}
	return normalizeProjectFile(file), nil
}

func normalizeProjectFile(file ProjectFile) ProjectFile {
	if file.SchemaVersion == 0 {
		file.SchemaVersion = vaultSchemaVersion
	}
	projects := make([]ProjectRecord, 0, len(file.Projects))
	seen := map[string]bool{}
	for _, project := range file.Projects {
		project.ID = sanitizeProjectID(project.ID)
		project.Name = strings.TrimSpace(project.Name)
		if project.ID == "" || project.Name == "" || seen[project.ID] {
			continue
		}
		project.Color = normalizeProjectColor(project.Color)
		seen[project.ID] = true
		projects = append(projects, project)
	}
	sort.SliceStable(projects, func(i, j int) bool {
		if projects[i].SortOrder == projects[j].SortOrder {
			return projects[i].CreatedAt < projects[j].CreatedAt
		}
		return projects[i].SortOrder < projects[j].SortOrder
	})
	file.Projects = projects
	file.ActiveProjectID = sanitizeProjectID(file.ActiveProjectID)
	if file.ActiveProjectID != "" && !projectFileHasID(file, file.ActiveProjectID) {
		file.ActiveProjectID = ""
	}
	return file
}

func saveVaultProjects(ctx *VaultContext, file ProjectFile) error {
	file = normalizeProjectFile(file)
	file.SchemaVersion = vaultSchemaVersion
	return writeJSONFileAtomic(filepath.Join(ctx.VeloDir, vaultProjectsFileName), file)
}

func listVaultProjects(ctx *VaultContext) (ProjectFile, error) {
	file, err := loadVaultProjects(ctx)
	if err != nil {
		return ProjectFile{}, err
	}
	return file, nil
}

func createVaultProject(ctx *VaultContext, req ProjectCreateRequest) (ProjectRecord, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return ProjectRecord{}, fmt.Errorf("project name is required")
	}
	file, err := loadVaultProjects(ctx)
	if err != nil {
		return ProjectRecord{}, err
	}
	sortOrder := len(file.Projects)
	for _, project := range file.Projects {
		if project.SortOrder >= sortOrder {
			sortOrder = project.SortOrder + 1
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	project := ProjectRecord{
		Archived:  false,
		Color:     normalizeProjectColor(req.Color),
		CreatedAt: now,
		ID:        newProjectID(),
		Name:      name,
		SortOrder: sortOrder,
		UpdatedAt: now,
	}
	file.Projects = append(file.Projects, project)
	if file.ActiveProjectID == "" {
		file.ActiveProjectID = project.ID
	}
	if err := saveVaultProjects(ctx, file); err != nil {
		return ProjectRecord{}, err
	}
	return project, nil
}

func updateVaultProject(ctx *VaultContext, req ProjectUpdateRequest) (ProjectRecord, error) {
	id := sanitizeProjectID(req.ID)
	if id == "" {
		return ProjectRecord{}, fmt.Errorf("project id is required")
	}
	file, err := loadVaultProjects(ctx)
	if err != nil {
		return ProjectRecord{}, err
	}
	for i, project := range file.Projects {
		if project.ID != id {
			continue
		}
		if req.Name != nil {
			name := strings.TrimSpace(*req.Name)
			if name == "" {
				return ProjectRecord{}, fmt.Errorf("project name is required")
			}
			project.Name = name
		}
		if req.Color != nil {
			project.Color = normalizeProjectColor(*req.Color)
		}
		if req.Archived != nil {
			project.Archived = *req.Archived
		}
		if req.SortOrder != nil {
			project.SortOrder = *req.SortOrder
		}
		project.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		file.Projects[i] = project
		if err := saveVaultProjects(ctx, file); err != nil {
			return ProjectRecord{}, err
		}
		return project, nil
	}
	return ProjectRecord{}, fmt.Errorf("project not found: %s", id)
}

func activateVaultProject(ctx *VaultContext, projectID string) (ProjectFile, error) {
	projectID = sanitizeProjectID(projectID)
	file, err := loadVaultProjects(ctx)
	if err != nil {
		return ProjectFile{}, err
	}
	if projectID != "" && !projectFileHasID(file, projectID) {
		return ProjectFile{}, fmt.Errorf("project not found: %s", projectID)
	}
	file.ActiveProjectID = projectID
	if err := saveVaultProjects(ctx, file); err != nil {
		return ProjectFile{}, err
	}
	return file, nil
}

func validateMemoProjectID(ctx *VaultContext, projectID string) (string, error) {
	projectID = sanitizeProjectID(projectID)
	if projectID == "" {
		return "", nil
	}
	file, err := loadVaultProjects(ctx)
	if err != nil {
		return "", err
	}
	if !projectFileHasID(file, projectID) {
		return "", fmt.Errorf("project not found: %s", projectID)
	}
	return projectID, nil
}

func projectFileHasID(file ProjectFile, id string) bool {
	for _, project := range file.Projects {
		if project.ID == id {
			return true
		}
	}
	return false
}
func newProjectID() string {
	return "project_" + randomVaultSuffix()
}

func sanitizeProjectID(value string) string {
	id := strings.TrimSpace(value)
	if id == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func normalizeProjectColor(value string) string {
	color := strings.TrimSpace(value)
	if color == "" {
		return "#2563eb"
	}
	if matched, _ := regexp.MatchString(`^#[0-9a-fA-F]{6}$`, color); matched {
		return strings.ToLower(color)
	}
	return "#2563eb"
}

func resolveOrCreateProjectByName(ctx *VaultContext, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil
	}
	file, err := loadVaultProjects(ctx)
	if err != nil {
		return "", err
	}
	for _, project := range file.Projects {
		if !project.Archived && project.Name == name {
			return project.ID, nil
		}
	}
	created, err := createVaultProject(ctx, ProjectCreateRequest{Name: name})
	if err != nil {
		return "", err
	}
	return created.ID, nil
}
