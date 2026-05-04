package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"minepannel-v6/auth"
	"strconv"
	"strings"
)

// --- Key Code Generation ---

func generateKeyCode() string {
	b := make([]byte, 6) // 6 bytes = 12 hex chars → 3 groups of 4
	if _, err := rand.Read(b); err != nil {
		// fallback: crypto/rand failure is near-impossible but handle gracefully
		return "CLOUD-0000-0000-0000"
	}
	hexStr := strings.ToUpper(hex.EncodeToString(b))
	return fmt.Sprintf("CLOUD-%s-%s-%s", hexStr[0:4], hexStr[4:8], hexStr[8:12])
}

func generateBindingToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// --- Client Endpoints ---

func preAuthStatusHandler(w http.ResponseWriter, r *http.Request) {
	sendSuccess(w, map[string]any{
		"pre_auth_enabled": s.Config().PreAuthEnabled,
	})
}

func preAuthBindHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		KeyCode     string `json:"key_code"`
		DeviceID    string `json:"device_id"`
		DeviceLabel string `json:"device_label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if req.KeyCode == "" {
		sendError(w, http.StatusBadRequest, "Missing key_code")
		return
	}
	if req.DeviceID == "" {
		sendError(w, http.StatusBadRequest, "Missing device_id")
		return
	}
	if req.DeviceLabel == "" {
		req.DeviceLabel = "Unknown Device"
	}

	// Normalize key code: uppercase, strip dashes for lookup, but accept both formats
	normalized := strings.ToUpper(strings.ReplaceAll(req.KeyCode, "-", ""))
	if len(normalized) == 12 {
		req.KeyCode = fmt.Sprintf("CLOUD-%s-%s-%s", normalized[0:4], normalized[4:8], normalized[8:12])
	}

	key, err := s.store.GetPreAuthKeyByCode(req.KeyCode)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid pre-authorization key")
		return
	}

	if !key.IsActive {
		sendError(w, http.StatusForbidden, "This pre-authorization key has been revoked")
		return
	}

	if key.DevicesBound >= key.MaxDevices {
		sendError(w, http.StatusForbidden, "No remaining device slots — contact the server administrator")
		return
	}

	deviceHash := hex.EncodeToString(auth.Sha256Hash(s.Config().PreAuthSecret + ":" + req.DeviceID))
	bindingToken := generateBindingToken()

	binding, err := s.store.CreateDeviceBinding(key.ID, deviceHash, bindingToken, req.DeviceLabel)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to create device binding")
		return
	}

	s.store.UpdatePreAuthKeyLastUsed(key.ID)

	sendSuccess(w, map[string]any{
		"binding_token": binding.BindingToken,
		"device_label":  binding.DeviceLabel,
	})
}

// --- Admin Endpoints ---

func adminListPreAuthKeysHandler(w http.ResponseWriter, r *http.Request) {
	keys, err := s.store.ListPreAuthKeys()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to list pre-auth keys")
		return
	}
	sendSuccess(w, keys)
}

func adminCreatePreAuthKeyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		PlayerName  string   `json:"player_name"`
		MaxDevices  int      `json:"max_devices"`
		InstanceIDs []string `json:"instance_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if req.MaxDevices <= 0 {
		req.MaxDevices = 3
	}

	userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	keyCode := generateKeyCode()

	key, err := s.store.CreatePreAuthKey(keyCode, req.PlayerName, req.MaxDevices, userID, req.InstanceIDs)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to create pre-auth key")
		return
	}

	sendSuccess(w, key)
}

func adminUpdatePreAuthKeyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/preauth/keys/")
	idStr = strings.TrimSuffix(idStr, "/bindings")
	idStr = strings.TrimSuffix(idStr, "/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid key ID")
		return
	}

	var req struct {
		PlayerName  string   `json:"player_name"`
		MaxDevices  int      `json:"max_devices"`
		IsActive    bool     `json:"is_active"`
		InstanceIDs []string `json:"instance_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if req.MaxDevices <= 0 {
		req.MaxDevices = 3
	}

	if err := s.store.UpdatePreAuthKey(id, req.PlayerName, req.MaxDevices, req.IsActive, req.InstanceIDs); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to update pre-auth key")
		return
	}

	key, _ := s.store.GetPreAuthKeyByID(id)
	sendSuccess(w, key)
}

func adminDeletePreAuthKeyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/preauth/keys/")
	idStr = strings.TrimSuffix(idStr, "/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid key ID")
		return
	}

	if err := s.store.DeletePreAuthKey(id); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to delete pre-auth key")
		return
	}

	sendSuccess(w, "Pre-auth key deleted")
}

func adminListKeyBindingsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/preauth/keys/")
	path = strings.TrimSuffix(path, "/bindings")
	id, err := strconv.Atoi(path)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid key ID")
		return
	}

	bindings, err := s.store.ListDeviceBindingsByKey(id)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to list bindings")
		return
	}

	sendSuccess(w, bindings)
}

func adminDeleteBindingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/preauth/bindings/")
	idStr = strings.TrimSuffix(idStr, "/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid binding ID")
		return
	}

	_, err = s.store.GetDeviceBindingByID(id)
	if err != nil {
		sendError(w, http.StatusNotFound, "Binding not found")
		return
	}

	if err := s.store.DeactivateDeviceBinding(id); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to deactivate binding")
		return
	}

	sendSuccess(w, "Binding revoked")
}

// --- Admin Dispatch ---

func adminPreAuthKeysDispatchHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/preauth/keys")
	path = strings.TrimSuffix(path, "/")

	if path == "" {
		switch r.Method {
		case http.MethodGet:
			adminListPreAuthKeysHandler(w, r)
		case http.MethodPost:
			adminCreatePreAuthKeyHandler(w, r)
		default:
			sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
		return
	}

	// /api/v1/admin/preauth/keys/{id}/bindings
	if strings.HasSuffix(path, "/bindings") {
		adminListKeyBindingsHandler(w, r)
		return
	}

	// /api/v1/admin/preauth/keys/{id}
	switch r.Method {
	case http.MethodPut:
		adminUpdatePreAuthKeyHandler(w, r)
	case http.MethodDelete:
		adminDeletePreAuthKeyHandler(w, r)
	default:
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}
