package desktopapp

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

const (
	defaultExternalAPIAddr = "127.0.0.1:18088"
	externalAPIMaxBodySize = 1 << 20
)

type externalAPIServerConfig struct {
	Addr    string
	Enabled bool
	Token   string
}

type externalAPIResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type externalAPIHandler struct{}

type externalAPIConfigFile struct {
	ExternalAPI struct {
		Address string `json:"address"`
		Addr    string `json:"addr"`
		Enabled *bool  `json:"enabled"`
		Host    string `json:"host"`
		Port    int    `json:"port"`
		Token   string `json:"token"`
	} `json:"external_api"`
}

type externalGTDCompleteRequest struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func loadExternalAPIServerConfig() externalAPIServerConfig {
	cfg := externalAPIServerConfig{
		Addr:    defaultExternalAPIAddr,
		Enabled: true,
	}

	if len(appAssets.AppConfigData) > 0 {
		var file externalAPIConfigFile
		if err := json.Unmarshal(appAssets.AppConfigData, &file); err == nil {
			api := file.ExternalAPI
			if api.Enabled != nil {
				cfg.Enabled = *api.Enabled
			}
			if addr := strings.TrimSpace(firstNonEmpty(api.Addr, api.Address)); addr != "" {
				cfg.Addr = addr
			} else if api.Port > 0 {
				host := strings.TrimSpace(api.Host)
				if host == "" {
					host = "127.0.0.1"
				}
				cfg.Addr = net.JoinHostPort(host, strconv.Itoa(api.Port))
			}
			cfg.Token = strings.TrimSpace(api.Token)
		}
	}

	if value, ok := lookupExternalAPIEnv("DEMO_DESKTOP_API_ENABLED", "VELO_DEMO_API_ENABLED"); ok {
		cfg.Enabled = parseExternalAPIBool(value, cfg.Enabled)
	}
	if value, ok := lookupExternalAPIEnv("DEMO_DESKTOP_API_ADDR", "VELO_DEMO_API_ADDR"); ok && strings.TrimSpace(value) != "" {
		cfg.Addr = strings.TrimSpace(value)
	} else if value, ok := lookupExternalAPIEnv("DEMO_DESKTOP_API_PORT", "VELO_DEMO_API_PORT"); ok {
		if port, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && port > 0 {
			cfg.Addr = net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
		}
	}
	if value, ok := lookupExternalAPIEnv("DEMO_DESKTOP_API_TOKEN", "VELO_DEMO_API_TOKEN"); ok {
		cfg.Token = strings.TrimSpace(value)
	}
	if strings.TrimSpace(cfg.Addr) == "" {
		cfg.Addr = defaultExternalAPIAddr
	}
	return cfg
}

func lookupExternalAPIEnv(names ...string) (string, bool) {
	for _, name := range names {
		if value, ok := os.LookupEnv(name); ok {
			return value, true
		}
	}
	return "", false
}

func parseExternalAPIBool(value string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on", "enable", "enabled":
		return true
	case "0", "false", "no", "off", "disable", "disabled":
		return false
	default:
		return fallback
	}
}

func startExternalAPIServer(logger *zerolog.Logger) *http.Server {
	cfg := loadExternalAPIServerConfig()
	if !cfg.Enabled {
		logger.Info().Msg("external API server disabled")
		return nil
	}

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           newExternalAPIMux(cfg),
		ReadHeaderTimeout: 5 * time.Second,
	}
	logger.Info().Str("addr", cfg.Addr).Bool("auth", cfg.Token != "").Msg("starting external API server")
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error().Err(err).Str("addr", cfg.Addr).Msg("external API server stopped")
		}
	}()
	return server
}

func shutdownExternalAPIServer(server *http.Server, logger *zerolog.Logger) {
	if server == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Warn().Err(err).Msg("failed to shutdown external API server")
	}
}

func newExternalAPIMux(cfg externalAPIServerConfig) http.Handler {
	api := &externalAPIHandler{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", api.handleHealth)
	mux.HandleFunc("/api/vault/status", api.handleVaultStatus)
	mux.HandleFunc("/api/gtd/items", api.handleGTDItems)
	mux.HandleFunc("/api/gtd/items/", api.handleGTDItem)
	mux.HandleFunc("/api/gtd/milestones", api.handleGTDMilestones)
	mux.HandleFunc("/api/gtd/milestones/", api.handleGTDMilestone)
	return externalAPIAuthMiddleware(cfg, mux)
}

func externalAPIAuthMiddleware(cfg externalAPIServerConfig, next http.Handler) http.Handler {
	token := strings.TrimSpace(cfg.Token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if token != "" && !externalAPITokenMatches(token, externalAPITokenFromRequest(r)) {
			writeExternalAPIError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func externalAPITokenFromRequest(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(auth) > 7 && strings.EqualFold(auth[:7], "Bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return strings.TrimSpace(r.Header.Get("X-Velo-API-Token"))
}

func externalAPITokenMatches(want string, got string) bool {
	if want == "" || got == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(want), []byte(got)) == 1
}

func (h *externalAPIHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeExternalAPIMethodNotAllowed(w, "GET")
		return
	}
	writeExternalAPIOK(w, map[string]interface{}{
		"activeVaultSelected": activeVaultSnapshot() != nil,
		"mode":                appMode(),
		"ok":                  true,
		"version":             appVersion(),
	})
}

func (h *externalAPIHandler) handleVaultStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeExternalAPIMethodNotAllowed(w, "GET")
		return
	}
	writeExternalAPIOK(w, map[string]interface{}{
		"active": activeVaultSnapshot(),
	})
}

func (h *externalAPIHandler) handleGTDItems(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ctx, err := requireActiveVault()
		if err != nil {
			writeExternalAPIDomainError(w, err)
			return
		}
		items, err := listVaultGTDItems(ctx)
		if err != nil {
			writeExternalAPIDomainError(w, err)
			return
		}
		writeExternalAPIOK(w, map[string]interface{}{"items": filterExternalGTDItems(items, r.URL.Query())})
	case http.MethodPost:
		ctx, err := requireActiveVault()
		if err != nil {
			writeExternalAPIDomainError(w, err)
			return
		}
		var req GTDItemCreateRequest
		if err := decodeExternalAPIJSON(w, r, &req, false); err != nil {
			writeExternalAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		item, err := createVaultGTDItem(ctx, req)
		if err != nil {
			writeExternalAPIDomainError(w, err)
			return
		}
		writeExternalAPIOK(w, map[string]interface{}{"item": item})
	default:
		writeExternalAPIMethodNotAllowed(w, "GET, POST")
	}
}

func (h *externalAPIHandler) handleGTDItem(w http.ResponseWriter, r *http.Request) {
	parts, ok := externalAPIPathParts(r.URL.Path, "/api/gtd/items/")
	if !ok || len(parts) == 0 {
		h.handleGTDItems(w, r)
		return
	}

	if isExternalGTDItemAction(parts[0]) {
		if len(parts) != 1 {
			writeExternalAPIError(w, http.StatusNotFound, "not found")
			return
		}
		h.handleGTDItemAction(w, r, parts[0])
		return
	}

	id := parts[0]
	if len(parts) == 1 {
		h.handleGTDItemResource(w, r, id)
		return
	}
	if len(parts) == 2 {
		switch parts[1] {
		case "close", "complete":
			h.handleGTDItemComplete(w, r, id)
			return
		case "delete":
			if r.Method != http.MethodPost {
				writeExternalAPIMethodNotAllowed(w, "POST")
				return
			}
			h.deleteGTDItem(w, id)
			return
		}
	}
	writeExternalAPIError(w, http.StatusNotFound, "not found")
}

func (h *externalAPIHandler) handleGTDItemResource(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		ctx, err := requireActiveVault()
		if err != nil {
			writeExternalAPIDomainError(w, err)
			return
		}
		item, err := getVaultGTDItem(ctx, id)
		if err != nil {
			writeExternalAPIDomainError(w, err)
			return
		}
		writeExternalAPIOK(w, map[string]interface{}{"item": item})
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		ctx, err := requireActiveVault()
		if err != nil {
			writeExternalAPIDomainError(w, err)
			return
		}
		var req GTDItemUpdateRequest
		if err := decodeExternalAPIJSON(w, r, &req, false); err != nil {
			writeExternalAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		req.ID = id
		item, err := updateVaultGTDItem(ctx, req)
		if err != nil {
			writeExternalAPIDomainError(w, err)
			return
		}
		writeExternalAPIOK(w, map[string]interface{}{"item": item})
	case http.MethodDelete:
		h.deleteGTDItem(w, id)
	default:
		writeExternalAPIMethodNotAllowed(w, "GET, POST, PUT, PATCH, DELETE")
	}
}

func (h *externalAPIHandler) handleGTDItemAction(w http.ResponseWriter, r *http.Request, action string) {
	if r.Method != http.MethodPost {
		writeExternalAPIMethodNotAllowed(w, "POST")
		return
	}
	ctx, err := requireActiveVault()
	if err != nil {
		writeExternalAPIDomainError(w, err)
		return
	}

	switch action {
	case "create":
		var req GTDItemCreateRequest
		if err := decodeExternalAPIJSON(w, r, &req, false); err != nil {
			writeExternalAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		item, err := createVaultGTDItem(ctx, req)
		if err != nil {
			writeExternalAPIDomainError(w, err)
			return
		}
		writeExternalAPIOK(w, map[string]interface{}{"item": item})
	case "update":
		var req GTDItemUpdateRequest
		if err := decodeExternalAPIJSON(w, r, &req, false); err != nil {
			writeExternalAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		item, err := updateVaultGTDItem(ctx, req)
		if err != nil {
			writeExternalAPIDomainError(w, err)
			return
		}
		writeExternalAPIOK(w, map[string]interface{}{"item": item})
	case "close", "complete":
		var req externalGTDCompleteRequest
		if err := decodeExternalAPIJSON(w, r, &req, false); err != nil {
			writeExternalAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		item, err := completeVaultGTDItem(ctx, req.ID, req.Status)
		if err != nil {
			writeExternalAPIDomainError(w, err)
			return
		}
		writeExternalAPIOK(w, map[string]interface{}{"item": item})
	case "delete":
		var req GTDIDRequest
		if err := decodeExternalAPIJSON(w, r, &req, false); err != nil {
			writeExternalAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := deleteVaultGTDItem(ctx, req.ID); err != nil {
			writeExternalAPIDomainError(w, err)
			return
		}
		writeExternalAPIOK(w, map[string]interface{}{"success": true})
	}
}

func (h *externalAPIHandler) handleGTDItemComplete(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost && r.Method != http.MethodPatch {
		writeExternalAPIMethodNotAllowed(w, "POST, PATCH")
		return
	}
	ctx, err := requireActiveVault()
	if err != nil {
		writeExternalAPIDomainError(w, err)
		return
	}
	var req externalGTDCompleteRequest
	if err := decodeExternalAPIJSON(w, r, &req, true); err != nil {
		writeExternalAPIError(w, http.StatusBadRequest, err.Error())
		return
	}
	item, err := completeVaultGTDItem(ctx, id, req.Status)
	if err != nil {
		writeExternalAPIDomainError(w, err)
		return
	}
	writeExternalAPIOK(w, map[string]interface{}{"item": item})
}

func (h *externalAPIHandler) deleteGTDItem(w http.ResponseWriter, id string) {
	ctx, err := requireActiveVault()
	if err != nil {
		writeExternalAPIDomainError(w, err)
		return
	}
	if err := deleteVaultGTDItem(ctx, id); err != nil {
		writeExternalAPIDomainError(w, err)
		return
	}
	writeExternalAPIOK(w, map[string]interface{}{"success": true})
}

func (h *externalAPIHandler) handleGTDMilestones(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ctx, err := requireActiveVault()
		if err != nil {
			writeExternalAPIDomainError(w, err)
			return
		}
		file, err := listVaultGTDMilestones(ctx)
		if err != nil {
			writeExternalAPIDomainError(w, err)
			return
		}
		writeExternalAPIOK(w, map[string]interface{}{"milestones": filterExternalGTDMilestones(file.Milestones, r.URL.Query())})
	case http.MethodPost:
		ctx, err := requireActiveVault()
		if err != nil {
			writeExternalAPIDomainError(w, err)
			return
		}
		var req GTDMilestoneCreateRequest
		if err := decodeExternalAPIJSON(w, r, &req, false); err != nil {
			writeExternalAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		milestone, err := createVaultGTDMilestone(ctx, req)
		if err != nil {
			writeExternalAPIDomainError(w, err)
			return
		}
		writeExternalAPIOK(w, map[string]interface{}{"milestone": milestone})
	default:
		writeExternalAPIMethodNotAllowed(w, "GET, POST")
	}
}

func (h *externalAPIHandler) handleGTDMilestone(w http.ResponseWriter, r *http.Request) {
	parts, ok := externalAPIPathParts(r.URL.Path, "/api/gtd/milestones/")
	if !ok || len(parts) == 0 {
		h.handleGTDMilestones(w, r)
		return
	}

	if isExternalGTDMilestoneAction(parts[0]) {
		if len(parts) != 1 {
			writeExternalAPIError(w, http.StatusNotFound, "not found")
			return
		}
		h.handleGTDMilestoneAction(w, r, parts[0])
		return
	}

	id := parts[0]
	if len(parts) == 1 {
		h.handleGTDMilestoneResource(w, r, id)
		return
	}
	if len(parts) == 2 {
		switch parts[1] {
		case "complete":
			h.handleGTDMilestoneComplete(w, r, id)
			return
		case "delete":
			if r.Method != http.MethodPost {
				writeExternalAPIMethodNotAllowed(w, "POST")
				return
			}
			h.deleteGTDMilestone(w, id)
			return
		}
	}
	writeExternalAPIError(w, http.StatusNotFound, "not found")
}

func (h *externalAPIHandler) handleGTDMilestoneResource(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		ctx, err := requireActiveVault()
		if err != nil {
			writeExternalAPIDomainError(w, err)
			return
		}
		milestone, err := getVaultGTDMilestone(ctx, id)
		if err != nil {
			writeExternalAPIDomainError(w, err)
			return
		}
		writeExternalAPIOK(w, map[string]interface{}{"milestone": milestone})
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		ctx, err := requireActiveVault()
		if err != nil {
			writeExternalAPIDomainError(w, err)
			return
		}
		var req GTDMilestoneUpdateRequest
		if err := decodeExternalAPIJSON(w, r, &req, false); err != nil {
			writeExternalAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		req.ID = id
		milestone, err := updateVaultGTDMilestone(ctx, req)
		if err != nil {
			writeExternalAPIDomainError(w, err)
			return
		}
		writeExternalAPIOK(w, map[string]interface{}{"milestone": milestone})
	case http.MethodDelete:
		h.deleteGTDMilestone(w, id)
	default:
		writeExternalAPIMethodNotAllowed(w, "GET, POST, PUT, PATCH, DELETE")
	}
}

func (h *externalAPIHandler) handleGTDMilestoneAction(w http.ResponseWriter, r *http.Request, action string) {
	if r.Method != http.MethodPost {
		writeExternalAPIMethodNotAllowed(w, "POST")
		return
	}
	ctx, err := requireActiveVault()
	if err != nil {
		writeExternalAPIDomainError(w, err)
		return
	}

	switch action {
	case "create":
		var req GTDMilestoneCreateRequest
		if err := decodeExternalAPIJSON(w, r, &req, false); err != nil {
			writeExternalAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		milestone, err := createVaultGTDMilestone(ctx, req)
		if err != nil {
			writeExternalAPIDomainError(w, err)
			return
		}
		writeExternalAPIOK(w, map[string]interface{}{"milestone": milestone})
	case "update":
		var req GTDMilestoneUpdateRequest
		if err := decodeExternalAPIJSON(w, r, &req, false); err != nil {
			writeExternalAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		milestone, err := updateVaultGTDMilestone(ctx, req)
		if err != nil {
			writeExternalAPIDomainError(w, err)
			return
		}
		writeExternalAPIOK(w, map[string]interface{}{"milestone": milestone})
	case "complete":
		var req GTDIDRequest
		if err := decodeExternalAPIJSON(w, r, &req, false); err != nil {
			writeExternalAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		milestone, err := completeVaultGTDMilestone(ctx, req.ID)
		if err != nil {
			writeExternalAPIDomainError(w, err)
			return
		}
		writeExternalAPIOK(w, map[string]interface{}{"milestone": milestone})
	case "delete":
		var req GTDIDRequest
		if err := decodeExternalAPIJSON(w, r, &req, false); err != nil {
			writeExternalAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := deleteVaultGTDMilestone(ctx, req.ID); err != nil {
			writeExternalAPIDomainError(w, err)
			return
		}
		writeExternalAPIOK(w, map[string]interface{}{"success": true})
	}
}

func (h *externalAPIHandler) handleGTDMilestoneComplete(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost && r.Method != http.MethodPatch {
		writeExternalAPIMethodNotAllowed(w, "POST, PATCH")
		return
	}
	ctx, err := requireActiveVault()
	if err != nil {
		writeExternalAPIDomainError(w, err)
		return
	}
	milestone, err := completeVaultGTDMilestone(ctx, id)
	if err != nil {
		writeExternalAPIDomainError(w, err)
		return
	}
	writeExternalAPIOK(w, map[string]interface{}{"milestone": milestone})
}

func (h *externalAPIHandler) deleteGTDMilestone(w http.ResponseWriter, id string) {
	ctx, err := requireActiveVault()
	if err != nil {
		writeExternalAPIDomainError(w, err)
		return
	}
	if err := deleteVaultGTDMilestone(ctx, id); err != nil {
		writeExternalAPIDomainError(w, err)
		return
	}
	writeExternalAPIOK(w, map[string]interface{}{"success": true})
}

func isExternalGTDItemAction(value string) bool {
	switch value {
	case "create", "update", "close", "complete", "delete":
		return true
	default:
		return false
	}
}

func isExternalGTDMilestoneAction(value string) bool {
	switch value {
	case "create", "update", "complete", "delete":
		return true
	default:
		return false
	}
}

func externalAPIPathParts(path string, prefix string) ([]string, bool) {
	if !strings.HasPrefix(path, prefix) {
		return nil, false
	}
	rest := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if rest == "" {
		return nil, true
	}
	rawParts := strings.Split(rest, "/")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		value, err := url.PathUnescape(part)
		if err != nil {
			value = part
		}
		value = strings.TrimSpace(value)
		if value != "" {
			parts = append(parts, value)
		}
	}
	return parts, true
}

func getVaultGTDItem(ctx *VaultContext, id string) (GTDItemRecord, error) {
	path, err := findGTDItemFilePath(ctx, id)
	if err != nil {
		return GTDItemRecord{}, err
	}
	return readGTDItemFile(ctx, path)
}

func completeVaultGTDItem(ctx *VaultContext, id string, status string) (GTDItemRecord, error) {
	next := gtdItemStatusClosed
	if strings.TrimSpace(status) != "" {
		next = normalizeGTDItemStatus(status)
		if !isClosedGTDItemStatus(next) {
			return GTDItemRecord{}, fmt.Errorf("complete status must be closed or resolved")
		}
	}
	return updateVaultGTDItem(ctx, GTDItemUpdateRequest{ID: id, Status: &next})
}

func getVaultGTDMilestone(ctx *VaultContext, id string) (GTDMilestoneRecord, error) {
	id = sanitizeGTDMilestoneID(id)
	if id == "" {
		return GTDMilestoneRecord{}, fmt.Errorf("milestone id is required")
	}
	file, err := loadGTDMilestones(ctx)
	if err != nil {
		return GTDMilestoneRecord{}, err
	}
	for _, milestone := range file.Milestones {
		if milestone.ID == id {
			return milestone, nil
		}
	}
	return GTDMilestoneRecord{}, fmt.Errorf("milestone not found: %s", id)
}

func completeVaultGTDMilestone(ctx *VaultContext, id string) (GTDMilestoneRecord, error) {
	status := gtdMilestoneStatusCompleted
	return updateVaultGTDMilestone(ctx, GTDMilestoneUpdateRequest{ID: id, Status: &status})
}

func filterExternalGTDItems(items []GTDItemRecord, query url.Values) []GTDItemRecord {
	status := strings.TrimSpace(query.Get("status"))
	projectID := sanitizeProjectID(query.Get("projectId"))
	milestoneID := sanitizeGTDMilestoneID(query.Get("milestoneId"))
	itemType := strings.TrimSpace(query.Get("type"))
	label := strings.TrimSpace(query.Get("label"))
	if status == "" && projectID == "" && milestoneID == "" && itemType == "" && label == "" {
		return items
	}

	next := make([]GTDItemRecord, 0, len(items))
	for _, item := range items {
		if status != "" && item.Status != normalizeGTDItemStatus(status) {
			continue
		}
		if projectID != "" && item.ProjectID != projectID {
			continue
		}
		if milestoneID != "" && item.MilestoneID != milestoneID {
			continue
		}
		if itemType != "" && item.Type != normalizeGTDItemType(itemType) {
			continue
		}
		if label != "" && !stringListContainsFold(item.Labels, label) {
			continue
		}
		next = append(next, item)
	}
	return next
}

func filterExternalGTDMilestones(milestones []GTDMilestoneRecord, query url.Values) []GTDMilestoneRecord {
	status := strings.TrimSpace(query.Get("status"))
	projectID := sanitizeProjectID(query.Get("projectId"))
	if status == "" && projectID == "" {
		return milestones
	}

	next := make([]GTDMilestoneRecord, 0, len(milestones))
	for _, milestone := range milestones {
		if status != "" && milestone.Status != normalizeGTDMilestoneStatus(status) {
			continue
		}
		if projectID != "" && !stringListContains(milestone.ProjectIDs, projectID) {
			continue
		}
		next = append(next, milestone)
	}
	return next
}

func stringListContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func stringListContainsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

func decodeExternalAPIJSON(w http.ResponseWriter, r *http.Request, dst interface{}, allowEmpty bool) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, externalAPIMaxBodySize))
	if err := decoder.Decode(dst); err != nil {
		if allowEmpty && errors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("invalid json: %w", err)
	}
	return nil
}

func writeExternalAPIOK(w http.ResponseWriter, data interface{}) {
	writeExternalAPIJSON(w, http.StatusOK, externalAPIResponse{Code: 0, Msg: "success", Data: data})
}

func writeExternalAPIError(w http.ResponseWriter, status int, message string) {
	if strings.TrimSpace(message) == "" {
		message = http.StatusText(status)
	}
	writeExternalAPIJSON(w, status, externalAPIResponse{Code: 100, Msg: message, Data: nil})
}

func writeExternalAPIDomainError(w http.ResponseWriter, err error) {
	writeExternalAPIError(w, externalAPIDomainErrorStatus(err), err.Error())
}

func writeExternalAPIMethodNotAllowed(w http.ResponseWriter, allowed string) {
	w.Header().Set("Allow", allowed)
	writeExternalAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func writeExternalAPIJSON(w http.ResponseWriter, status int, response externalAPIResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func externalAPIDomainErrorStatus(err error) int {
	if err == nil {
		return http.StatusInternalServerError
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "not found"):
		return http.StatusNotFound
	case strings.Contains(message, "vault is not selected"):
		return http.StatusConflict
	case strings.Contains(message, "is required"), strings.Contains(message, "invalid"):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
