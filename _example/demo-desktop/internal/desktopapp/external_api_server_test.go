package desktopapp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type externalAPITestEnvelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func TestExternalAPIGTDItemLifecycle(t *testing.T) {
	ctx, _, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	setActiveVault(ctx)
	defer setActiveVault(nil)

	server := httptest.NewServer(newExternalAPIMux(externalAPIServerConfig{}))
	defer server.Close()

	status, envelope := externalAPITestRequest(t, server.URL, http.MethodPost, "/api/gtd/items", map[string]interface{}{
		"labels": []string{"api"},
		"title":  "External inbox item",
		"type":   "feature",
	}, "")
	if status != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("create status=%d envelope=%+v", status, envelope)
	}
	var createData struct {
		Item GTDItemRecord `json:"item"`
	}
	if err := json.Unmarshal(envelope.Data, &createData); err != nil {
		t.Fatalf("decode create data: %v", err)
	}
	if createData.Item.ID == "" {
		t.Fatalf("created item id is empty")
	}

	itemPath := "/api/gtd/items/" + url.PathEscape(createData.Item.ID)
	status, envelope = externalAPITestRequest(t, server.URL, http.MethodPatch, itemPath, map[string]interface{}{
		"decision": "ship the external API",
		"title":    "External API item",
	}, "")
	if status != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("update status=%d envelope=%+v", status, envelope)
	}
	var updateData struct {
		Item GTDItemRecord `json:"item"`
	}
	if err := json.Unmarshal(envelope.Data, &updateData); err != nil {
		t.Fatalf("decode update data: %v", err)
	}
	if updateData.Item.Title != "External API item" || updateData.Item.Decision != "ship the external API" {
		t.Fatalf("updated item = %+v", updateData.Item)
	}

	status, envelope = externalAPITestRequest(t, server.URL, http.MethodPost, itemPath+"/complete", nil, "")
	if status != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("complete status=%d envelope=%+v", status, envelope)
	}
	var completeData struct {
		Item GTDItemRecord `json:"item"`
	}
	if err := json.Unmarshal(envelope.Data, &completeData); err != nil {
		t.Fatalf("decode complete data: %v", err)
	}
	if completeData.Item.Status != gtdItemStatusClosed || completeData.Item.ClosedAt == "" {
		t.Fatalf("completed item = %+v", completeData.Item)
	}

	status, envelope = externalAPITestRequest(t, server.URL, http.MethodGet, "/api/gtd/items?status=closed", nil, "")
	if status != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("list status=%d envelope=%+v", status, envelope)
	}
	var listData struct {
		Items []GTDItemRecord `json:"items"`
	}
	if err := json.Unmarshal(envelope.Data, &listData); err != nil {
		t.Fatalf("decode list data: %v", err)
	}
	if len(listData.Items) != 1 || listData.Items[0].ID != createData.Item.ID {
		t.Fatalf("listed items = %+v", listData.Items)
	}

	status, envelope = externalAPITestRequest(t, server.URL, http.MethodDelete, itemPath, nil, "")
	if status != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("delete status=%d envelope=%+v", status, envelope)
	}
	status, envelope = externalAPITestRequest(t, server.URL, http.MethodGet, itemPath, nil, "")
	if status != http.StatusNotFound || envelope.Code == 0 {
		t.Fatalf("get deleted status=%d envelope=%+v", status, envelope)
	}
}

func TestExternalAPISupportsActionStyleGTDItemRoutes(t *testing.T) {
	ctx, _, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	setActiveVault(ctx)
	defer setActiveVault(nil)

	server := httptest.NewServer(newExternalAPIMux(externalAPIServerConfig{}))
	defer server.Close()

	status, envelope := externalAPITestRequest(t, server.URL, http.MethodPost, "/api/gtd/items/create", map[string]interface{}{
		"title": "Action route item",
	}, "")
	if status != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("create status=%d envelope=%+v", status, envelope)
	}
	var createData struct {
		Item GTDItemRecord `json:"item"`
	}
	if err := json.Unmarshal(envelope.Data, &createData); err != nil {
		t.Fatalf("decode create data: %v", err)
	}

	status, envelope = externalAPITestRequest(t, server.URL, http.MethodPost, "/api/gtd/items/update", map[string]interface{}{
		"id":     createData.Item.ID,
		"status": gtdItemStatusWaiting,
	}, "")
	if status != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("update status=%d envelope=%+v", status, envelope)
	}

	status, envelope = externalAPITestRequest(t, server.URL, http.MethodPost, "/api/gtd/items/close", map[string]interface{}{
		"id": createData.Item.ID,
	}, "")
	if status != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("close status=%d envelope=%+v", status, envelope)
	}

	status, envelope = externalAPITestRequest(t, server.URL, http.MethodPost, "/api/gtd/items/delete", map[string]interface{}{
		"id": createData.Item.ID,
	}, "")
	if status != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("delete status=%d envelope=%+v", status, envelope)
	}
}

func TestExternalAPIGTDMilestoneLifecycle(t *testing.T) {
	ctx, _, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	setActiveVault(ctx)
	defer setActiveVault(nil)

	server := httptest.NewServer(newExternalAPIMux(externalAPIServerConfig{}))
	defer server.Close()

	status, envelope := externalAPITestRequest(t, server.URL, http.MethodPost, "/api/gtd/milestones", map[string]interface{}{
		"status": "active",
		"title":  "External API milestone",
	}, "")
	if status != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("create status=%d envelope=%+v", status, envelope)
	}
	var createData struct {
		Milestone GTDMilestoneRecord `json:"milestone"`
	}
	if err := json.Unmarshal(envelope.Data, &createData); err != nil {
		t.Fatalf("decode create data: %v", err)
	}
	if createData.Milestone.ID == "" {
		t.Fatalf("created milestone id is empty")
	}

	milestonePath := "/api/gtd/milestones/" + url.PathEscape(createData.Milestone.ID)
	status, envelope = externalAPITestRequest(t, server.URL, http.MethodPatch, milestonePath, map[string]interface{}{
		"title": "External API milestone updated",
	}, "")
	if status != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("update status=%d envelope=%+v", status, envelope)
	}

	status, envelope = externalAPITestRequest(t, server.URL, http.MethodPost, milestonePath+"/complete", nil, "")
	if status != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("complete status=%d envelope=%+v", status, envelope)
	}
	var completeData struct {
		Milestone GTDMilestoneRecord `json:"milestone"`
	}
	if err := json.Unmarshal(envelope.Data, &completeData); err != nil {
		t.Fatalf("decode complete data: %v", err)
	}
	if completeData.Milestone.Status != gtdMilestoneStatusCompleted || completeData.Milestone.CompletedAt == "" {
		t.Fatalf("completed milestone = %+v", completeData.Milestone)
	}

	status, envelope = externalAPITestRequest(t, server.URL, http.MethodDelete, milestonePath, nil, "")
	if status != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("delete status=%d envelope=%+v", status, envelope)
	}
}

func TestExternalAPIRequiresConfiguredToken(t *testing.T) {
	server := httptest.NewServer(newExternalAPIMux(externalAPIServerConfig{Token: "secret"}))
	defer server.Close()

	status, envelope := externalAPITestRequest(t, server.URL, http.MethodGet, "/api/health", nil, "")
	if status != http.StatusUnauthorized || envelope.Code != 100 {
		t.Fatalf("unauthorized status=%d envelope=%+v", status, envelope)
	}

	status, envelope = externalAPITestRequest(t, server.URL, http.MethodGet, "/api/health", nil, "secret")
	if status != http.StatusOK || envelope.Code != 0 {
		t.Fatalf("authorized status=%d envelope=%+v", status, envelope)
	}
}

func externalAPITestRequest(t *testing.T, baseURL string, method string, path string, body interface{}, token string) (int, externalAPITestEnvelope) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, baseURL+path, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	var envelope externalAPITestEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("decode response %s: %v", string(raw), err)
	}
	return resp.StatusCode, envelope
}
