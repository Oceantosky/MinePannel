#!/usr/bin/env python3
"""Rebuild main.go from scratch after corruption."""

GO_SOURCE = r'''package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// --- Types ---

type NodeConfig struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	IP          string `json:"ip"`
	Flag        string `json:"flag"`
}

type Config struct {
	NodeName        string       `json:"node_name"`
	Role            string       `json:"role"`
	SecretKey       string       `json:"secret_key"`
	PublicIP        string       `json:"public_ip"`
	DistPolicy      string       `json:"dist_policy"`
	JWTSecret       string       `json:"jwt_secret"`
	PreAuthSecret   string       `json:"pre_auth_secret"`
	PreAuthEnabled  bool         `json:"pre_auth_enabled"`
	Nodes           []NodeConfig `json:"nodes"`
}

type ReleaseEntry struct {
	VersionID string    `json:"version_id"`
	Type      string    `json:"type"`      // "commit" or "packet"
	Message   string    `json:"message"`   // human-readable summary
	FileName  string    `json:"file_name"` // zip filename, empty for commits
	Size      int64     `json:"size"`      // file bytes, 0 for commits
	Time      time.Time `json:"time"`
}

type InstanceMetadata struct {
	DisplayName   string         `json:"display_name"`
	CDNLink       string         `json:"cdn_link"`
	IsPaused      bool           `json:"is_paused"`
	ActiveVersion string         `json:"active_version"`
	Releases      []ReleaseEntry `json:"releases"`
}

type Instance struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
	FullPackURL string `json:"full_pack_url"`
}

// --- Globals ---

var (
	globalConfig  Config
	globalStore   *Store
	instanceLocks sync.Map
	logChannels   sync.Map
	logHistory    sync.Map
	logMutex      sync.Mutex
	startTime     = time.Now()
)

func validateInstanceID(id string) error {
	if id == "" || strings.Contains(id, "..") || strings.ContainsAny(id, `/\`) {
		return fmt.Errorf("invalid instance id")
	}
	return nil
}

func getSafePath(id, p string) (string, error) {
	if err := validateInstanceID(id); err != nil {
		return "", err
	}
	basePath, err := filepath.Abs(filepath.Join(".", "instances", id, "files"))
	if err != nil {
		return "", err
	}
	targetPath, err := filepath.Abs(filepath.Join(basePath, p))
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(targetPath, basePath) {
		return "", fmt.Errorf("path traversal detected")
	}
	return targetPath, nil
}

const authTimeWindow = 300 // seconds

// --- HMAC Auth ---

func generateHMAC(secret string, timestamp int64, method, path, bodyHash string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(fmt.Sprintf("%d:%s:%s:%s", timestamp, method, path, bodyHash)))
	return hex.EncodeToString(h.Sum(nil))
}

func validateHMAC(token string, timestamp int64, secret, method, path, bodyHash string) bool {
	expected := generateHMAC(secret, timestamp, method, path, bodyHash)
	return hmac.Equal([]byte(token), []byte(expected))
}

// hmacGet performs an HTTP GET with HMAC signing for inter-node / PCL client requests.
func hmacGet(urlStr string) (*http.Response, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	reqPath := u.Path
	if u.RawQuery != "" {
		reqPath += "?" + u.RawQuery
	}
	timestamp := time.Now().Unix()
	hash := sha256.Sum256(nil)
	bodyHash := hex.EncodeToString(hash[:])
	token := generateHMAC(globalConfig.SecretKey, timestamp, "GET", reqPath, bodyHash)
	req, _ := http.NewRequest("GET", urlStr, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Timestamp", fmt.Sprintf("%d", timestamp))
	return (&http.Client{Timeout: 30 * time.Second}).Do(req)
}

func withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			sendError(w, http.StatusUnauthorized, "Missing authorization header")
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			sendError(w, http.StatusUnauthorized, "Invalid authorization format, expected: Bearer <token>")
			return
		}
		clientToken := parts[1]
		tsStr := r.Header.Get("X-Timestamp")
		if tsStr == "" {
			sendError(w, http.StatusUnauthorized, "Missing X-Timestamp header")
			return
		}
		timestamp, err := strconv.ParseInt(tsStr, 10, 64)
		if err != nil {
			sendError(w, http.StatusUnauthorized, "Invalid X-Timestamp value")
			return
		}
		now := time.Now().Unix()
		diff := now - timestamp
		if diff < 0 {
			diff = -diff
		}
		if diff > authTimeWindow {
			sendError(w, http.StatusUnauthorized, fmt.Sprintf("Request expired (window: %ds)", authTimeWindow))
			return
		}
		var bodyHash string
		if r.Body != nil && !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			bodyBytes, err := io.ReadAll(r.Body)
			if err == nil {
				hash := sha256.Sum256(bodyBytes)
				bodyHash = hex.EncodeToString(hash[:])
				r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
		} else {
			hash := sha256.Sum256(nil)
			bodyHash = hex.EncodeToString(hash[:])
		}
		if !validateHMAC(clientToken, timestamp, globalConfig.SecretKey, r.Method, r.URL.Path, bodyHash) {
			sendError(w, http.StatusUnauthorized, "Invalid authentication token")
			return
		}
		next(w, r)
	}
}

// --- JWT Web Auth ---

func withJWT(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			sendError(w, http.StatusUnauthorized, "Missing authorization header")
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			sendError(w, http.StatusUnauthorized, "Invalid authorization format")
			return
		}
		claims, err := ValidateToken(parts[1])
		if err != nil {
			sendError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}
		user, err := globalStore.GetUserByID(claims.UserID)
		if err != nil || !user.IsActive {
			sendError(w, http.StatusUnauthorized, "User not found or disabled")
			return
		}
		r.Header.Set("X-User-ID", fmt.Sprintf("%d", claims.UserID))
		r.Header.Set("X-User-Role", claims.Role)
		r.Header.Set("X-Username", claims.Username)
		r.Header.Set("X-Auth-Method", "jwt")
		next(w, r)
	}
}

// withOptionalJWT tries to parse JWT but doesn't fail if missing
func withOptionalJWT(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 {
				claims, err := ValidateToken(parts[1])
				if err == nil {
					user, err := globalStore.GetUserByID(claims.UserID)
					if err == nil && user.IsActive {
						r.Header.Set("X-User-ID", fmt.Sprintf("%d", claims.UserID))
						r.Header.Set("X-User-Role", claims.Role)
						r.Header.Set("X-Username", claims.Username)
						r.Header.Set("X-Auth-Method", "jwt")
					}
				}
			}
		}
		next(w, r)
	}
}

// withWebAuth tries JWT first, then falls back to HMAC.
// Also accepts ?token=<jwt> as a query parameter for download links that can't set headers.
func withWebAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if token := r.URL.Query().Get("token"); token != "" && !strings.HasPrefix(authHeader, "Bearer ") {
			authHeader = "Bearer " + token
		}
		if strings.HasPrefix(authHeader, "Bearer ") {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 {
				// If it looks like a JWT (contains dots), validate as JWT and do not fall back.
				if strings.Contains(parts[1], ".") {
					claims, err := ValidateToken(parts[1])
					if err == nil {
						user, err := globalStore.GetUserByID(claims.UserID)
						if err == nil && user.IsActive {
							r.Header.Set("X-User-ID", fmt.Sprintf("%d", claims.UserID))
							r.Header.Set("X-User-Role", claims.Role)
							r.Header.Set("X-Username", claims.Username)
							r.Header.Set("X-Auth-Method", "jwt")
							next(w, r)
							return
						} else {
							sendError(w, http.StatusUnauthorized, "User not found or disabled")
							return
						}
					} else {
						sendError(w, http.StatusUnauthorized, "Invalid or expired JWT token")
						return
					}
				}
			}
		}
		// Fall through to HMAC auth only if it does not look like a JWT
		withAuth(next)(w, r)
	}
}

// withPreAuth guards routes with the pre-auth device binding system.
// When pre-auth is OFF, it is completely transparent (backwards compatible).
// When pre-auth is ON, it validates in order:
//  1. JWT (for web panel users)
//  2. HMAC signature (for inter-node slave sync / internal trust)
//  3. X-Binding-Token + X-Device-ID (for PCL client device binding)
func withPreAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !globalConfig.PreAuthEnabled {
			next(w, r)
			return
		}
		// Also accept ?token=<jwt> for download links
		authHeader := r.Header.Get("Authorization")
		if token := r.URL.Query().Get("token"); token != "" && !strings.HasPrefix(authHeader, "Bearer ") {
			authHeader = "Bearer " + token
		}
		// Try JWT first (for web panel users)
		if strings.HasPrefix(authHeader, "Bearer ") {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.Contains(parts[1], ".") {
				claims, err := ValidateToken(parts[1])
				if err == nil {
					user, err := globalStore.GetUserByID(claims.UserID)
					if err == nil && user.IsActive {
						r.Header.Set("X-User-ID", fmt.Sprintf("%d", claims.UserID))
						r.Header.Set("X-User-Role", claims.Role)
						r.Header.Set("X-Username", claims.Username)
						r.Header.Set("X-Auth-Method", "jwt")
						next(w, r)
						return
					}
				}
			}
		}
		// Try HMAC auth (for inter-node/internal trust)
		tsStr := r.Header.Get("X-Timestamp")
		if strings.HasPrefix(authHeader, "Bearer ") && tsStr != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 {
				timestamp, err := strconv.ParseInt(tsStr, 10, 64)
				if err == nil {
					now := time.Now().Unix()
					diff := now - timestamp
					if diff < 0 {
						diff = -diff
					}
					if diff <= authTimeWindow {
						var bodyHash string
						if r.Body != nil && !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
							bodyBytes, err := io.ReadAll(r.Body)
							if err == nil {
								hash := sha256.Sum256(bodyBytes)
								bodyHash = hex.EncodeToString(hash[:])
								r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
							}
						} else {
							hash := sha256.Sum256(nil)
							bodyHash = hex.EncodeToString(hash[:])
						}
						if validateHMAC(parts[1], timestamp, globalConfig.SecretKey, r.Method, r.URL.Path, bodyHash) {
							r.Header.Set("X-Auth-Method", "hmac")
							next(w, r)
							return
						}
					}
				}
			}
		}
		// Try device binding auth
		bindingToken := r.Header.Get("X-Binding-Token")
		deviceID := r.Header.Get("X-Device-ID")
		if bindingToken == "" {
			sendError(w, http.StatusUnauthorized, "Missing X-Binding-Token header")
			return
		}
		if deviceID == "" {
			sendError(w, http.StatusUnauthorized, "Missing X-Device-ID header")
			return
		}
		binding, err := globalStore.GetDeviceBindingByToken(bindingToken)
		if err != nil || !binding.IsActive {
			sendError(w, http.StatusUnauthorized, "Invalid or revoked binding token")
			return
		}
		key, err := globalStore.GetPreAuthKeyByID(binding.KeyID)
		if err != nil || !key.IsActive {
			sendError(w, http.StatusUnauthorized, "Pre-authorization key has been revoked")
			return
		}
		// Verify device hash: SHA256(PreAuthSecret + ":" + deviceID)
		expectedHash := hex.EncodeToString(sha256Hash(globalConfig.PreAuthSecret + ":" + deviceID))
		if binding.DeviceHash != expectedHash {
			sendError(w, http.StatusUnauthorized, "Device identity mismatch — hardware may have changed")
			return
		}
		// Async update last_seen
		go globalStore.UpdateDeviceBindingLastSeen(binding.ID)
		r.Header.Set("X-Auth-Method", "preauth")
		r.Header.Set("X-Binding-ID", fmt.Sprintf("%d", binding.ID))
		r.Header.Set("X-Key-ID", fmt.Sprintf("%d", key.ID))
		next(w, r)
	}
}

func sha256Hash(s string) []byte {
	h := sha256.Sum256([]byte(s))
	return h[:]
}

// adminOnly middleware - requires JWT with admin role
func adminOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-Method") != "jwt" {
			sendError(w, http.StatusForbidden, "JWT authentication required")
			return
		}
		if r.Header.Get("X-User-Role") != "admin" {
			sendError(w, http.StatusForbidden, "Admin access required")
			return
		}
		next(w, r)
	}
}

// withInstanceAccess checks JWT user has permission to access the instance in ?id=
func withInstanceAccess(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// HMAC/preauth/system-level auth bypasses instance checks
		if r.Header.Get("X-Auth-Method") != "jwt" {
			next(w, r)
			return
		}
		// Admin users bypass instance checks
		if r.Header.Get("X-User-Role") == "admin" {
			next(w, r)
			return
		}
		instanceID := r.URL.Query().Get("id")
		if instanceID == "" {
			next(w, r)
			return
		}
		userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
		if !globalStore.UserHasInstance(userID, instanceID) {
			sendError(w, http.StatusForbidden, "Access denied to this instance")
			return
		}
		next(w, r)
	}
}

// --- CORS ---

func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "http://localhost:5173" || origin == "http://127.0.0.1:5173" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", fmt.Sprintf("http://%s:55000", globalConfig.PublicIP))
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Timestamp, X-Binding-Token, X-Device-ID")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

// --- Response Helpers ---

func sendJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func sendError(w http.ResponseWriter, status int, message string) {
	sendJSON(w, status, map[string]any{"success": false, "message": message})
}

func sendSuccess(w http.ResponseWriter, data any) {
	sendJSON(w, http.StatusOK, map[string]any{"success": true, "data": data})
}

// --- Utilities ---

func generateRandomID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%08x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func broadcastLog(id, message string) {
	logMutex.Lock()
	val, _ := logHistory.LoadOrStore(id, []string{})
	history := val.([]string)
	if len(history) > 20 {
		history = history[1:]
	}
	logHistory.Store(id, append(history, message))
	logMutex.Unlock()
	if chs, ok := logChannels.Load(id); ok {
		for _, ch := range chs.([]chan string) {
			select {
			case ch <- message:
			default:
			}
		}
	}
}

func loadInstanceMetadata(id string) (InstanceMetadata, error) {
	var m InstanceMetadata
	f, err := os.ReadFile(filepath.Join("./instances", id, "instance.json"))
	if err != nil {
		return m, err
	}
	json.Unmarshal(f, &m)
	return m, nil
}

func saveInstanceMetadata(id string, m InstanceMetadata) {
	d, _ := json.MarshalIndent(m, "", "  ")
	os.WriteFile(filepath.Join("./instances", id, "instance.json"), d, 0644)
}

func loadConfig() {
	f, _ := os.ReadFile("config.json")
	json.Unmarshal(f, &globalConfig)

	modified := false
	if len(globalConfig.Nodes) == 0 {
		globalConfig.Nodes = []NodeConfig{{Name: "bj", DisplayName: "Master", IP: "127.0.0.1", Flag: "CN"}}
		modified = true
	}
	if globalConfig.NodeName == "" {
		globalConfig.NodeName = "Default_Node"
		modified = true
	}
	if globalConfig.Role == "" {
		globalConfig.Role = "master"
		modified = true
	}
	if globalConfig.SecretKey == "" {
		globalConfig.SecretKey = "CloudAbroad_Secret_" + generateRandomID()
		modified = true
	}
	if globalConfig.PublicIP == "" {
		globalConfig.PublicIP = "127.0.0.1"
		modified = true
	}
	if globalConfig.DistPolicy == "" {
		globalConfig.DistPolicy = "direct"
		modified = true
	}
	if globalConfig.JWTSecret == "" {
		globalConfig.JWTSecret = "CloudAbroad_JWT_" + generateRandomID()
		modified = true
	}
	if globalConfig.PreAuthSecret == "" {
		globalConfig.PreAuthSecret = "CloudSync_PreAuth_" + generateRandomID()
		modified = true
	}

	if modified {
		saveConfig()
	}
	InitJWT(globalConfig.JWTSecret)
}

func saveConfig() {
	d, _ := json.MarshalIndent(globalConfig, "", "  ")
	os.WriteFile("config.json", d, 0644)
}

// --- Handlers ---

func healthHandler(w http.ResponseWriter, r *http.Request) {
	sendSuccess(w, map[string]any{
		"status":    "healthy",
		"node":      globalConfig.NodeName,
		"timestamp": time.Now().Unix(),
	})
}

func listInstancesHandler(w http.ResponseWriter, r *http.Request) {
	// Determine if JWT-authenticated user (for filtering)
	var allowedIDs map[string]bool
	if r.Header.Get("X-Auth-Method") == "jwt" && r.Header.Get("X-User-Role") != "admin" {
		userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
		user, err := globalStore.GetUserByID(userID)
		if err == nil {
			var instances []string
			json.Unmarshal([]byte(user.Instances), &instances)
			allowedIDs = make(map[string]bool)
			for _, id := range instances {
				allowedIDs[id] = true
			}
		}
	}

	list := []Instance{}
	es, _ := os.ReadDir("./instances")
	for _, e := range es {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		// Filter by user's allowed instances
		if allowedIDs != nil && !allowedIDs[id] {
			continue
		}
		meta, err := loadInstanceMetadata(id)
		if err != nil {
			continue
		}
		status := "正常"
		if meta.IsPaused {
			status = "暂停"
		} else if t, ok := instanceLocks.Load(id); ok {
			status = t.(string)
		}
		v := meta.ActiveVersion
		f := "full_client.zip"
		if v != "" {
			f = "full_client_" + v + ".zip"
		}
		u := fmt.Sprintf("http://%s:55001/sync/%s/releases/%s", globalConfig.PublicIP, id, f)
		if globalConfig.DistPolicy == "cdn" && meta.CDNLink != "" {
			u = meta.CDNLink
		}
		list = append(list, Instance{
			ID:          id,
			DisplayName: meta.DisplayName,
			Status:      status,
			FullPackURL: u,
		})
	}
	sendSuccess(w, map[string]any{"instance_list": list})
}

func pushHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if err := validateInstanceID(id); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid instance id")
		return
	}
	broadcastLog(id, "INFO: [Dist] Signaling all slave nodes...")
	for _, node := range globalConfig.Nodes {
		if node.IP != "127.0.0.1" && node.IP != globalConfig.PublicIP {
			broadcastLog(id, "DEBUG: Notify -> "+node.DisplayName)
			sendPushToSlave(node.IP, id, globalConfig.SecretKey)
		}
	}
	broadcastLog(id, "SUCCESS: Signal sent to all nodes.")
	sendSuccess(w, "Push Started")
}

func nodesStatusHandler(w http.ResponseWriter, r *http.Request) {
	type NodeStatus struct {
		Name    string `json:"name"`
		Online  bool   `json:"online"`
		Latency int64  `json:"latency"`
	}
	var statuses = make([]NodeStatus, len(globalConfig.Nodes))
	var wg sync.WaitGroup
	for i, node := range globalConfig.Nodes {
		wg.Add(1)
		go func(idx int, n NodeConfig) {
			defer wg.Done()
			start := time.Now()
			client := http.Client{Timeout: 3 * time.Second}
			resp, err := client.Get(fmt.Sprintf("http://%s:55000/health", n.IP))
			latency := int64(-1)
			online := false
			if err == nil {
				latency = time.Since(start).Milliseconds()
				defer resp.Body.Close()
				if resp.StatusCode == 200 {
					online = true
				}
			}
			statuses[idx] = NodeStatus{
				Name:    n.Name,
				Online:  online,
				Latency: latency,
			}
		}(i, node)
	}
	wg.Wait()
	sendSuccess(w, statuses)
}

func configReadHandler(w http.ResponseWriter, r *http.Request) {
	sendSuccess(w, globalConfig)
}

func publicConfigHandler(w http.ResponseWriter, r *http.Request) {
	sendSuccess(w, map[string]any{
		"dist_policy":       globalConfig.DistPolicy,
		"public_ip":         globalConfig.PublicIP,
		"node_name":         globalConfig.NodeName,
		"role":              globalConfig.Role,
		"pre_auth_enabled":  globalConfig.PreAuthEnabled,
	})
}

func configWriteHandler(w http.ResponseWriter, r *http.Request) {
	var c Config
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if c.SecretKey == "" {
		sendError(w, http.StatusBadRequest, "Secret key cannot be empty")
		return
	}
	if c.DistPolicy != "direct" && c.DistPolicy != "cdn" {
		sendError(w, http.StatusBadRequest, "Distribution policy must be 'direct' or 'cdn'")
		return
	}
	oldSecretKey := globalConfig.SecretKey

	// Preserve existing JWTSecret if none provided to prevent lockout
	if c.JWTSecret == "" {
		c.JWTSecret = globalConfig.JWTSecret
	}
	// Preserve existing PreAuthSecret if none provided
	if c.PreAuthSecret == "" {
		c.PreAuthSecret = globalConfig.PreAuthSecret
	}
	// Preserve fields that shouldn't be overwritten by SettingsView
	if c.Role == "" {
		c.Role = globalConfig.Role
	}
	if c.PublicIP == "" {
		c.PublicIP = globalConfig.PublicIP
	}
	if c.NodeName == "" {
		c.NodeName = globalConfig.NodeName
	}
	if len(c.Nodes) == 0 {
		c.Nodes = globalConfig.Nodes
	}

	globalConfig = c
	saveConfig()

	if oldSecretKey != globalConfig.SecretKey {
		broadcastLog("__system__", "INFO: [Config] PSK changed, propagating to slave nodes...")
		for _, node := range globalConfig.Nodes {
			if !isLocalNode(node) {
				go sendConfigToSlave(node, oldSecretKey)
			}
		}
	}
	sendSuccess(w, "Config saved")
}

func createInstanceHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	id := generateRandomID()
	os.MkdirAll(filepath.Join("./instances", id, "files"), 0755)
	saveInstanceMetadata(id, InstanceMetadata{DisplayName: name})

	// Auto-assign to creating JWT user
	if r.Header.Get("X-Auth-Method") == "jwt" {
		userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
		user, err := globalStore.GetUserByID(userID)
		if err == nil && user.Role != "admin" {
			var instances []string
			json.Unmarshal([]byte(user.Instances), &instances)
			instances = append(instances, id)
			globalStore.UpdateUser(userID, user.Username, user.Role, user.IsActive, instances)
		}
	}

	broadcastLog(id, "SUCCESS: Instance created with display name ["+name+"]")
	sendSuccess(w, map[string]string{"id": id})
}

func deleteInstanceHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if err := validateInstanceID(id); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid instance id")
		return
	}
	os.RemoveAll(filepath.Join("./instances", id))
	sendSuccess(w, "Deleted")
}

func metadataReadHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if err := validateInstanceID(id); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid instance id")
		return
	}
	meta, err := loadInstanceMetadata(id)
	if err != nil {
		sendError(w, http.StatusNotFound, "Metadata not found")
		return
	}
	sendSuccess(w, meta)
}

func metadataWriteHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if err := validateInstanceID(id); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid instance id")
		return
	}
	var m InstanceMetadata
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	// Backend Enforcement: If global CDN policy is not enabled, forcefully clear any submitted CDN link
	if globalConfig.DistPolicy != "cdn" {
		m.CDNLink = ""
	}
	saveInstanceMetadata(id, m)
	sendSuccess(w, "Metadata updated")
}

func metadataDispatchHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		metadataReadHandler(w, r)
	case http.MethodPost:
		metadataWriteHandler(w, r)
	default:
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func commitHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if err := validateInstanceID(id); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid instance id")
		return
	}
	instanceLocks.Store(id, "Committing")
	go func() {
		v := time.Now().Format("20060102.150405")
		broadcastLog(id, "INFO: Generating Manifest ["+v+"]...")
		manifest, err := GenerateManifest(id, v, filepath.Join("./instances", id, "files"), "./data/objects")
		if err == nil {
			SaveManifest(id, manifest)
			broadcastLog(id, "SUCCESS: Manifest created and objects deduplicated.")
			m, metaErr := loadInstanceMetadata(id)
			if metaErr == nil {
				m.ActiveVersion = v
				m.Releases = append(m.Releases, ReleaseEntry{
					VersionID: v,
					Type:      "commit",
					Message:   fmt.Sprintf("Commit: %d files tracked", len(manifest.Files)),
					Time:      time.Now(),
				})
				saveInstanceMetadata(id, m)
			}
		} else {
			broadcastLog(id, "ERROR: "+err.Error())
		}
		instanceLocks.Delete(id)
	}()
	sendSuccess(w, "Commit Started")
}

func packetUpHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if err := validateInstanceID(id); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid instance id")
		return
	}
	instanceLocks.Store(id, "Packing")
	go func() {
		m, err := loadInstanceMetadata(id)
		if err != nil {
			instanceLocks.Delete(id)
			return
		}
		v := m.ActiveVersion
		if v == "" {
			v = time.Now().Format("20060102.150405")
		}
		f := "full_client_" + v + ".zip"
		t := filepath.Join("./instances", id, "releases", f)
		os.MkdirAll(filepath.Dir(t), 0755)
		broadcastLog(id, "INFO: Creating CDN Base Release ["+v+"]...")
		err = zipSource(id, filepath.Join("./instances", id, "files"), t, m)
		if err != nil {
			broadcastLog(id, "ERROR: Failed to create zip: "+err.Error())
		} else {
			s, _ := os.Stat(t)
			m.Releases = append(m.Releases, ReleaseEntry{
				VersionID: v,
				Type:      "packet",
				Message:   fmt.Sprintf("Packet Up: %s (%.1f MB)", f, float64(s.Size())/(1024*1024)),
				FileName:  f,
				Size:      s.Size(),
				Time:      time.Now(),
			})
			saveInstanceMetadata(id, m)
			broadcastLog(id, "SUCCESS: CDN Base Release published.")
		}
		instanceLocks.Delete(id)
	}()
	sendSuccess(w, "Packet Up Accepted")
}

func fileListHandler(w http.ResponseWriter, r *http.Request) {
	id, p := r.URL.Query().Get("id"), r.URL.Query().Get("path")
	safeDir, err := getSafePath(id, p)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	es, err := os.ReadDir(safeDir)
	if err != nil {
		sendError(w, http.StatusNotFound, "Directory not found")
		return
	}
	res := []map[string]any{}
	for _, e := range es {
		info, _ := e.Info()
		res = append(res, map[string]any{
			"name":   e.Name(),
			"is_dir": e.IsDir(),
			"size":   info.Size(),
			"time":   info.ModTime(),
		})
	}
	sendSuccess(w, res)
}

func fileReadHandler(w http.ResponseWriter, r *http.Request) {
	id, p := r.URL.Query().Get("id"), r.URL.Query().Get("path")
	safeFile, err := getSafePath(id, p)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	c, err := os.ReadFile(safeFile)
	if err != nil {
		sendError(w, http.StatusNotFound, "File not found")
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(c)
}

func fileWriteHandler(w http.ResponseWriter, r *http.Request) {
	id, p := r.URL.Query().Get("id"), r.URL.Query().Get("path")
	targetPath, err := getSafePath(id, p)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	content, err := io.ReadAll(r.Body)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	defer r.Body.Close()
	os.MkdirAll(filepath.Dir(targetPath), 0755)
	if err := os.WriteFile(targetPath, content, 0644); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to write file")
		return
	}
	sendSuccess(w, "Saved")
}

func fileUploadHandler(w http.ResponseWriter, r *http.Request) {
	id, p := r.URL.Query().Get("id"), r.URL.Query().Get("path")
	file, header, err := r.FormFile("file")
	if err != nil {
		sendError(w, http.StatusBadRequest, "Missing file in request")
		return
	}
	defer file.Close()

	targetPath, err := getSafePath(id, filepath.Join(p, header.Filename))
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid path")
		return
	}

	os.MkdirAll(filepath.Dir(targetPath), 0755)
	dst, err := os.Create(targetPath)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to create file")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to write file")
		return
	}
	sendSuccess(w, "Uploaded")
}

func fileMkdirHandler(w http.ResponseWriter, r *http.Request) {
	id, p := r.URL.Query().Get("id"), r.URL.Query().Get("path")
	dirPath, err := getSafePath(id, p)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to create directory")
		return
	}
	sendSuccess(w, "Created")
}

func fileDeleteHandler(w http.ResponseWriter, r *http.Request) {
	id, p := r.URL.Query().Get("id"), r.URL.Query().Get("path")
	targetPath, err := getSafePath(id, p)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	if err := os.RemoveAll(targetPath); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to delete")
		return
	}
	sendSuccess(w, "Deleted")
}

func fileRenameHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	id, src := r.URL.Query().Get("id"), r.URL.Query().Get("path")
	dst := r.URL.Query().Get("to")
	srcPath, err := getSafePath(id, src)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid source path")
		return
	}
	dstPath, err := getSafePath(id, dst)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid destination path")
		return
	}
	if err := os.Rename(srcPath, dstPath); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to rename")
		return
	}
	sendSuccess(w, "Renamed")
}

func releaseDownloadHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if err := validateInstanceID(id); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid instance id")
		return
	}
	meta, err := loadInstanceMetadata(id)
	if err != nil || len(meta.Releases) == 0 {
		sendError(w, http.StatusNotFound, "No release available")
		return
	}
	latest := meta.Releases[len(meta.Releases)-1]
	if strings.Contains(latest.FileName, "..") || strings.Contains(latest.FileName, "/") || strings.Contains(latest.FileName, "\\") {
		sendError(w, http.StatusForbidden, "Invalid release filename")
		return
	}
	releasePath := filepath.Join(".", "instances", id, "releases", latest.FileName)
	if _, err := os.Stat(releasePath); os.IsNotExist(err) {
		sendError(w, http.StatusNotFound, "Release file not found")
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", latest.FileName))
	w.Header().Set("Content-Type", "application/zip")
	http.ServeFile(w, r, releasePath)
}

func uploadPclHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if err := validateInstanceID(id); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid instance id")
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		sendError(w, http.StatusBadRequest, "Missing file")
		return
	}
	z := filepath.Join("./instances", id, "temp.zip")
	out, err := os.Create(z)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to create temp file")
		return
	}
	if _, err := io.Copy(out, file); err != nil {
		out.Close()
		sendError(w, http.StatusInternalServerError, "Failed to write temp file")
		return
	}
	out.Close()
	file.Close()

	instanceLocks.Store(id, "Deploying")
	sendSuccess(w, "Upload & Deploy Started")

	go func() {
		defer func() { instanceLocks.Delete(id); os.Remove(z) }()

		broadcastLog(id, "INFO: Unzipping uploaded pack...")
		stagingDir := filepath.Join("./instances", id, "staging_unzip")
		os.RemoveAll(stagingDir)
		os.MkdirAll(stagingDir, 0755)

		zr, err := zip.OpenReader(z)
		if err != nil {
			broadcastLog(id, "ERROR: Failed to read ZIP")
			return
		}
		stagingAbs, _ := filepath.Abs(stagingDir)
		for _, f := range zr.File {
			p := filepath.Join(stagingDir, f.Name)
			absP, _ := filepath.Abs(p)
			if !strings.HasPrefix(absP, stagingAbs+string(filepath.Separator)) && absP != stagingAbs {
				broadcastLog(id, "WARN: Skipping path traversal in ZIP: "+f.Name)
				continue
			}
			if f.FileInfo().IsDir() {
				os.MkdirAll(p, 0755)
				continue
			}
			os.MkdirAll(filepath.Dir(p), 0755)
			rc, err := f.Open()
			if err != nil {
				broadcastLog(id, "WARN: Failed to open ZIP entry: "+f.Name)
				continue
			}
			out, err := os.Create(p)
			if err != nil {
				rc.Close()
				broadcastLog(id, "WARN: Failed to create file: "+f.Name)
				continue
			}
			if _, err := io.Copy(out, rc); err != nil {
				broadcastLog(id, "WARN: Failed to write file: "+f.Name)
			}
			out.Close()
			rc.Close()
		}
		zr.Close()

		broadcastLog(id, "INFO: Analyzing pack structure...")
		meta, _ := loadInstanceMetadata(id)

		pclIniPath := filepath.Join(stagingDir, "PCL.ini")
		if content, err := os.ReadFile(pclIniPath); err == nil {
			broadcastLog(id, "SUCCESS: PCL pack detected.")
			for line := range strings.SplitSeq(string(content), "\n") {
				line = strings.TrimSpace(line)
				if val, ok := strings.CutPrefix(line, "Name:"); ok {
					meta.DisplayName = val
				}
			}
		}

		// Inject modpack resolution here
		ProcessModpack(id, stagingDir)

		srcDir := stagingDir
		if _, err := os.Stat(filepath.Join(stagingDir, ".minecraft")); err == nil {
			srcDir = filepath.Join(stagingDir, ".minecraft")
			broadcastLog(id, "DEBUG: Found .minecraft directory.")
		}

		targetDir := filepath.Join("./instances", id, "files")
		os.RemoveAll(targetDir)
		os.MkdirAll(targetDir, 0755)

		broadcastLog(id, "INFO: Aligning files to instance...")
		filepath.Walk(srcDir, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(srcDir, p)
			dest := filepath.Join(targetDir, rel)
			os.MkdirAll(filepath.Dir(dest), 0755)
			if err := os.Rename(p, dest); err != nil {
				srcFile, err := os.Open(p)
				if err != nil {
					broadcastLog(id, "WARN: Fallback copy failed to open source: "+rel)
					return nil
				}
				dstFile, err := os.Create(dest)
				if err != nil {
					srcFile.Close()
					broadcastLog(id, "WARN: Fallback copy failed to create dest: "+rel)
					return nil
				}
				if _, err := io.Copy(dstFile, srcFile); err != nil {
					broadcastLog(id, "WARN: Fallback copy failed: "+rel)
				}
				srcFile.Close()
				dstFile.Close()
			}
			return nil
		})

		os.RemoveAll(stagingDir)

		v := time.Now().Format("20060102.150405")
		broadcastLog(id, "INFO: Auto-committing initial snapshot ["+v+"]...")
		manifest, err := GenerateManifest(id, v, targetDir, "./data/objects")
		if err == nil {
			SaveManifest(id, manifest)
			meta.ActiveVersion = v
			meta.Releases = append(meta.Releases, ReleaseEntry{
				VersionID: v,
				Type:      "packet",
				Message:   "PCL Pack Deployed: " + meta.DisplayName,
				Time:      time.Now(),
			})
			saveInstanceMetadata(id, meta)
			broadcastLog(id, "SUCCESS: Deployment & Commit complete.")
		} else {
			broadcastLog(id, "ERROR: Auto-commit failed: "+err.Error())
		}
	}()
}

func marketDownloadHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	var req struct {
		URL      string `json:"url"`
		FileName string `json:"file_name"`
		Type     string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	parsedURL, err := url.Parse(req.URL)
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid URL")
		return
	}
	host := parsedURL.Hostname()
	allowed := false
	for _, domain := range []string{"modrinth.com", "curseforge.com", "mcimirror.top", "github.com", "githubusercontent.com"} {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			allowed = true
			break
		}
	}
	if !allowed {
		sendError(w, http.StatusForbidden, "Domain not allowed")
		return
	}

	targetPath, err := getSafePath(id, filepath.Join(req.Type, req.FileName))
	if err != nil {
		sendError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	os.MkdirAll(filepath.Dir(targetPath), 0755)

	broadcastLog(id, "INFO: [Market] Downloading "+req.FileName+"...")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(req.URL)
	if err != nil {
		broadcastLog(id, "ERROR: Download failed: "+err.Error())
		sendError(w, http.StatusBadGateway, "Download failed")
		return
	}
	if resp.StatusCode != 200 {
		resp.Body.Close()
		broadcastLog(id, "ERROR: Download failed (Status: "+fmt.Sprintf("%d", resp.StatusCode)+")")
		sendError(w, http.StatusBadGateway, "Download failed")
		return
	}
	defer resp.Body.Close()

	out, err := os.Create(targetPath)
	if err != nil {
		broadcastLog(id, "ERROR: Failed to create file: "+err.Error())
		sendError(w, http.StatusInternalServerError, "Failed to create file")
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		broadcastLog(id, "ERROR: Failed to write file: "+err.Error())
		sendError(w, http.StatusInternalServerError, "Failed to write file")
		return
	}

	broadcastLog(id, "SUCCESS: [Market] Saved to "+req.Type+"/"+req.FileName)
	sendSuccess(w, "Downloaded")
}

func modrinthProxyHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		sendError(w, http.StatusBadRequest, "Missing path parameter")
		return
	}
	if !strings.HasPrefix(path, "/v2/") || strings.Contains(path, "..") {
		sendError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	proxyURL := "https://api.modrinth.com" + path
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(proxyURL)
	if err != nil {
		sendError(w, http.StatusBadGateway, "Proxy request failed")
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	if _, err := io.Copy(w, resp.Body); err != nil {
		// Client disconnect during streaming is expected
	}
}

func curseProxyHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		sendError(w, http.StatusBadRequest, "Missing path parameter")
		return
	}
	if !strings.HasPrefix(path, "/v1/") || strings.Contains(path, "..") {
		sendError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	proxyURL := "https://mod.mcimirror.top/curseforge" + path
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(proxyURL)
	if err != nil {
		sendError(w, http.StatusBadGateway, "Proxy request failed")
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	if _, err := io.Copy(w, resp.Body); err != nil {
		// Client disconnect during streaming is expected
	}
}

func sseHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan string, 10)
	logMutex.Lock()
	actual, _ := logChannels.LoadOrStore(id, []chan string{})
	logChannels.Store(id, append(actual.([]chan string), ch))
	logMutex.Unlock()

	defer func() {
		logMutex.Lock()
		if a, ok := logChannels.Load(id); ok {
			cs := a.([]chan string)
			var n []chan string
			for _, c := range cs {
				if c != ch {
					n = append(n, c)
				}
			}
			logChannels.Store(id, n)
		}
		logMutex.Unlock()
		close(ch)
	}()

	if hist, ok := logHistory.Load(id); ok {
		for _, msg := range hist.([]string) {
			fmt.Fprintf(w, "data: %s\n\n", msg)
		}
	}
	f, _ := w.(http.Flusher)
	f.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case m := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", m)
			f.Flush()
		}
	}
}

func syncFileHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/sync/"), "/")
	if len(parts) >= 1 {
		id := parts[0]
		if err := validateInstanceID(id); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid instance id"))
			return
		}
		// Lock check (503 Service Unavailable during commit/pack/sync)
		if _, locked := instanceLocks.Load(id); locked {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Instance is currently locked for maintenance (Status: 503)"))
			return
		}
		meta, err := loadInstanceMetadata(id)
		if err == nil && !meta.IsPaused {
			http.StripPrefix("/sync/", http.FileServer(http.Dir("./instances"))).ServeHTTP(w, r)
			return
		}
	}
	w.WriteHeader(http.StatusServiceUnavailable)
	w.Write([]byte("Instance is paused or path is invalid"))
}

func zipSource(id, src, dst string, meta InstanceMetadata) error {
	z, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer z.Close()

	a := zip.NewWriter(z)
	defer a.Close()

	pclIniContent := fmt.Sprintf("Version:CloudAbroad\nSecret:%s\nName:%s (%s)\nInfo:由 CloudAbroad 驱动的高速同步客户端\nSyncUrl:http://%s:55000/sync/%s/instance.json\nSyncPolicy:Enforce\n",
		globalConfig.SecretKey, meta.DisplayName, meta.ActiveVersion, globalConfig.PublicIP, id)

	broadcastLog(id, "DEBUG: Injecting PCL.ini...")
	hPcl := &zip.FileHeader{Name: "PCL.ini", Method: zip.Deflate, Modified: time.Now()}
	writerPcl, err := a.CreateHeader(hPcl)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(writerPcl, pclIniContent); err != nil {
		return err
	}

	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		broadcastLog(id, "DEBUG: Compressing -> "+rel)
		h, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		h.Name = filepath.ToSlash(rel)
		h.Method = zip.Deflate
		writer, err := a.CreateHeader(h)
		if err != nil {
			return err
		}
		fr, err := os.Open(p)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, fr)
		fr.Close()
		if err != nil {
			return err
		}
		return nil
	})
}

// --- Heartbeat & Config Sync ---

func isLocalNode(node NodeConfig) bool {
	return node.IP == "127.0.0.1" || node.IP == globalConfig.PublicIP
}

func computeConfigHash() string {
	h := sha256.New()
	h.Write([]byte(globalConfig.SecretKey + globalConfig.JWTSecret))
	return hex.EncodeToString(h.Sum(nil))
}

func probeNode(node NodeConfig) {
	start := time.Now()
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s:55000/health", node.IP))
	if err != nil {
		broadcastLog("__system__", fmt.Sprintf("WARN: [Heartbeat] %s is UNREACHABLE", node.DisplayName))
		return
	}
	resp.Body.Close()
	latency := time.Since(start).Milliseconds()

	timestamp := time.Now().Unix()
	path := "/api/v1/slave/status"
	method := "POST"
	hash := sha256.Sum256(nil)
	bodyHash := hex.EncodeToString(hash[:])
	token := generateHMAC(globalConfig.SecretKey, timestamp, method, path, bodyHash)

	req, _ := http.NewRequest(method, fmt.Sprintf("http://%s:55000%s", node.IP, path), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Timestamp", fmt.Sprintf("%d", timestamp))

	sResp, err := client.Do(req)
	if err != nil {
		broadcastLog("__system__", fmt.Sprintf("WARN: [Heartbeat] %s status fetch failed after health OK", node.DisplayName))
		return
	}
	defer sResp.Body.Close()

	if sResp.StatusCode != 200 {
		broadcastLog("__system__", fmt.Sprintf("WARN: [Heartbeat] %s returned %d for status", node.DisplayName, sResp.StatusCode))
		return
	}

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			NodeName   string            `json:"node_name"`
			Instances  map[string]string `json:"instances"`
			ConfigHash string            `json:"config_hash"`
			Uptime     int64             `json:"uptime"`
		} `json:"data"`
	}
	if err := json.NewDecoder(sResp.Body).Decode(&result); err != nil {
		broadcastLog("__system__", fmt.Sprintf("WARN: [Heartbeat] %s response parse failed", node.DisplayName))
		return
	}
	if !result.Success {
		return
	}

	masterHash := computeConfigHash()
	if result.Data.ConfigHash != masterHash {
		broadcastLog("__system__", fmt.Sprintf("WARN: [Heartbeat] %s config hash mismatch — may need PSK sync", node.DisplayName))
	}

	syncedCount := 0
	for instID, slaveVersion := range result.Data.Instances {
		masterMeta, err := loadInstanceMetadata(instID)
		if err != nil {
			continue
		}
		if masterMeta.ActiveVersion != slaveVersion && masterMeta.ActiveVersion != "" {
			broadcastLog("__system__", fmt.Sprintf("INFO: [Heartbeat] %s instance %s version mismatch (slave=%s, master=%s), syncing...", node.DisplayName, instID, slaveVersion, masterMeta.ActiveVersion))
			sendPushToSlave(node.IP, instID, globalConfig.SecretKey)
			syncedCount++
		}
	}

	broadcastLog("__system__", fmt.Sprintf("INFO: [Heartbeat] %s online | latency=%dms | uptime=%ds | instances=%d | synced=%d",
		node.DisplayName, latency, result.Data.Uptime, len(result.Data.Instances), syncedCount))
}

func heartbeatLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		for _, node := range globalConfig.Nodes {
			if isLocalNode(node) {
				continue
			}
			go probeNode(node)
		}
	}
}

func sendConfigToSlave(node NodeConfig, signKey string) {
	timestamp := time.Now().Unix()
	path := "/api/v1/slave/config-sync"
	method := "POST"
	body := map[string]any{
		"secret_key":       globalConfig.SecretKey,
		"jwt_secret":       globalConfig.JWTSecret,
		"pre_auth_secret":  globalConfig.PreAuthSecret,
		"pre_auth_enabled": globalConfig.PreAuthEnabled,
		"nodes":            globalConfig.Nodes,
		"dist_policy":      globalConfig.DistPolicy,
	}
	bodyBytes, _ := json.Marshal(body)
	bodyHash := fmt.Sprintf("%x", sha256.Sum256(bodyBytes))
	token := generateHMAC(signKey, timestamp, method, path, bodyHash)

	req, err := http.NewRequest(method, fmt.Sprintf("http://%s:55000%s", node.IP, path), bytes.NewReader(bodyBytes))
	if err != nil {
		broadcastLog("__system__", fmt.Sprintf("ERROR: [ConfigSync] Failed to build request for %s", node.DisplayName))
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Timestamp", fmt.Sprintf("%d", timestamp))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		broadcastLog("__system__", fmt.Sprintf("ERROR: [ConfigSync] Failed to send config to %s: %v", node.DisplayName, err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		broadcastLog("__system__", fmt.Sprintf("SUCCESS: [ConfigSync] %s updated to new PSK", node.DisplayName))
	} else {
		broadcastLog("__system__", fmt.Sprintf("ERROR: [ConfigSync] %s returned %d", node.DisplayName, resp.StatusCode))
	}
}

// --- Main ---

func main() {
	// Initialize store and load config
	loadConfig()
	var err error
	globalStore, err = NewStore("./data.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer globalStore.Close()

	mux := http.NewServeMux()

	// --- Public endpoints (no auth required) ---
	mux.HandleFunc("/health", withCORS(healthHandler))
	mux.HandleFunc("/api/v1/master/console/stream", withCORS(sseHandler))

	// --- Pre-Auth client endpoints (public) ---
	mux.HandleFunc("/api/v1/preauth/status", withCORS(preAuthStatusHandler))
	mux.HandleFunc("/api/v1/preauth/bind", withCORS(preAuthBindHandler))

	// --- Auth endpoints ---
	mux.HandleFunc("/api/v1/auth/login", withCORS(loginHandler(globalStore)))
	mux.HandleFunc("/api/v1/auth/status", withCORS(withJWT(authStatusHandler(globalStore))))
	mux.HandleFunc("/api/v1/auth/change-password", withCORS(withJWT(changePasswordHandler(globalStore))))

	// --- Admin endpoints (JWT + admin only) ---
	mux.HandleFunc("/api/v1/admin/users", withCORS(withJWT(adminOnly(adminUsersDispatchHandler(globalStore)))))
	mux.HandleFunc("/api/v1/admin/users/", withCORS(withJWT(adminOnly(adminUsersDispatchHandler(globalStore)))))
	mux.HandleFunc("/api/v1/admin/login-attempts", withCORS(withJWT(adminOnly(listLoginAttemptsHandler(globalStore)))))
	mux.HandleFunc("/api/v1/admin/preauth/keys", withCORS(withJWT(adminOnly(adminPreAuthKeysDispatchHandler))))
	mux.HandleFunc("/api/v1/admin/preauth/keys/", withCORS(withJWT(adminOnly(adminPreAuthKeysDispatchHandler))))
	mux.HandleFunc("/api/v1/admin/preauth/bindings/", withCORS(withJWT(adminOnly(adminDeleteBindingHandler))))

	mux.HandleFunc("/api/v1/master/config/read", withCORS(withJWT(adminOnly(configReadHandler))))
	mux.HandleFunc("/api/v1/master/config/write", withCORS(withJWT(adminOnly(configWriteHandler))))
	mux.HandleFunc("/api/v1/master/config/public", withCORS(publicConfigHandler))
	mux.HandleFunc("/api/v1/master/nodes/status", withCORS(withJWT(adminOnly(nodesStatusHandler))))
	mux.HandleFunc("/api/v1/master/stats", withCORS(withJWT(adminOnly(statsHandler))))

	// --- Web panel instance endpoints (JWT or HMAC + instance access) ---
	mux.HandleFunc("/api/v1/master/instances/metadata", withCORS(withWebAuth(withInstanceAccess(metadataDispatchHandler))))
	mux.HandleFunc("/api/v1/master/instances/create", withCORS(withWebAuth(createInstanceHandler)))
	mux.HandleFunc("/api/v1/master/instances/delete", withCORS(withWebAuth(withInstanceAccess(deleteInstanceHandler))))
	mux.HandleFunc("/api/v1/master/instances/commit", withCORS(withWebAuth(withInstanceAccess(commitHandler))))
	mux.HandleFunc("/api/v1/master/packet_up", withCORS(withWebAuth(withInstanceAccess(packetUpHandler))))
	mux.HandleFunc("/api/v1/master/push", withCORS(withWebAuth(withInstanceAccess(pushHandler))))

	// --- File endpoints (JWT or HMAC + instance access) ---
	mux.HandleFunc("/api/v1/master/files/list", withCORS(withWebAuth(withInstanceAccess(fileListHandler))))
	mux.HandleFunc("/api/v1/master/files/read", withCORS(withWebAuth(withInstanceAccess(fileReadHandler))))
	mux.HandleFunc("/api/v1/master/files/write", withCORS(withWebAuth(withInstanceAccess(fileWriteHandler))))
	mux.HandleFunc("/api/v1/master/files/upload", withCORS(withWebAuth(withInstanceAccess(fileUploadHandler))))
	mux.HandleFunc("/api/v1/master/files/mkdir", withCORS(withWebAuth(withInstanceAccess(fileMkdirHandler))))
	mux.HandleFunc("/api/v1/master/files/delete", withCORS(withWebAuth(withInstanceAccess(fileDeleteHandler))))
	mux.HandleFunc("/api/v1/master/files/rename", withCORS(withWebAuth(withInstanceAccess(fileRenameHandler))))
	mux.HandleFunc("/api/v1/master/files/upload_pcl_pack", withCORS(withWebAuth(withInstanceAccess(uploadPclHandler))))

	// --- Market endpoints (JWT or HMAC + instance access) ---
	mux.HandleFunc("/api/v1/master/releases/download", withCORS(withWebAuth(withInstanceAccess(releaseDownloadHandler))))
	mux.HandleFunc("/api/v1/master/market/download", withCORS(withWebAuth(withInstanceAccess(marketDownloadHandler))))
	mux.HandleFunc("/api/v1/master/market/curse-proxy", withCORS(withWebAuth(withInstanceAccess(curseProxyHandler))))
	mux.HandleFunc("/api/v1/master/market/modrinth-proxy", withCORS(withWebAuth(withInstanceAccess(modrinthProxyHandler))))

	// --- Client instance list (pre-auth guarded, transparent when off) ---
	mux.HandleFunc("/api/v1/sync/instances", withCORS(withPreAuth(listInstancesHandler)))

	// --- Data plane: static file serving (pre-auth guarded, transparent when off) ---
	mux.HandleFunc("/sync/", withPreAuth(syncFileHandler))

	// --- Slave endpoints (HMAC internal trust) ---
	mux.HandleFunc("/api/v1/slave/sync", withCORS(withAuth(slaveSyncHandler)))
	mux.HandleFunc("/api/v1/slave/status", withCORS(withAuth(slaveStatusHandler)))
	mux.HandleFunc("/api/v1/slave/config-sync", withCORS(withAuth(slaveConfigSyncHandler)))

	// --- SPA fallback: serve web/ for all unmatched paths ---
	if globalConfig.Role == "slave" {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not Found (Slave Node)"))
		})
	} else {
		mux.Handle("/", http.FileServer(http.Dir("./web")))
	}

	adminServer := &http.Server{
		Addr:         ":55000",
		Handler:      withByteCounter(mux),
		ReadTimeout:  15 * time.Minute,
		WriteTimeout: 15 * time.Minute,
		IdleTimeout:  120 * time.Second,
	}

	dataMux := http.NewServeMux()
	dataMux.HandleFunc("/sync/", withPreAuth(syncFileHandler))
	dataServer := &http.Server{
		Addr:         ":55001",
		Handler:      dataMux,
		ReadTimeout:  15 * time.Minute,
		WriteTimeout: 15 * time.Minute,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		fmt.Println("[Control Plane] Listening on :55000")
		if err := adminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Control plane error: %v\n", err)
		}
	}()

	go func() {
		fmt.Println("[Data Plane] Listening on :55001")
		if err := dataServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Data plane error: %v\n", err)
		}
	}()

	fmt.Println("========================================")
	fmt.Println("  CloudAbroad Master Node (v6-security)")
	fmt.Println("  Health: http://localhost:55000/health")
	fmt.Println("========================================")

	// Start heartbeat loop for Master nodes
	if globalConfig.Role == "master" {
		go heartbeatLoop()
		fmt.Println("[Heartbeat] Master-Slave heartbeat started (30s interval)")
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\nShutting down servers...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := adminServer.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Admin server shutdown error: %v\n", err)
	}
	if err := dataServer.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Data server shutdown error: %v\n", err)
	}
	fmt.Println("Servers stopped gracefully.")
}
'''

with open('main.go', 'w', encoding='utf-8', newline='\n') as f:
    f.write(GO_SOURCE)

print("main.go rebuilt successfully")
ilt successfully")
