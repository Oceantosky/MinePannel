package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"cloudsync-v6/auth"
	"cloudsync-v6/types"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// --- Slave Node Operations ---

// sendPushToSlave sends a sync request to a slave node for a specific instance.
func sendPushToSlave(ip, instanceID, secretKey string) {
	timestamp := time.Now().Unix()
	path := "/api/v1/slave/sync"
	method := "POST"
	body := map[string]string{"instance_id": instanceID}
	bodyBytes, _ := json.Marshal(body)
	hash := sha256.Sum256(bodyBytes)
	bodyHash := hex.EncodeToString(hash[:])
	token := auth.GenerateMAC(secretKey, timestamp, method, path, bodyHash)

	req, err := http.NewRequest(method, fmt.Sprintf("http://%s:%d%s", ip, s.Config().ManagePort, path), bytes.NewReader(bodyBytes))
	if err != nil {
		broadcastLog("__system__", fmt.Sprintf("ERROR: [Dist] Failed to build push request for %s", ip))
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Timestamp", fmt.Sprintf("%d", timestamp))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		broadcastLog("__system__", fmt.Sprintf("WARN: [Dist] Failed to reach slave %s: %v", ip, err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		broadcastLog("__system__", fmt.Sprintf("WARN: [Dist] Slave %s returned %d", ip, resp.StatusCode))
	}
}

// --- Slave HTTP Handlers ---

func slaveSyncHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		InstanceID string `json:"instance_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if req.InstanceID == "" {
		sendError(w, http.StatusBadRequest, "Missing instance_id")
		return
	}

	if err := validateInstanceID(req.InstanceID); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid instance id")
		return
	}

	go syncFromMaster(req.InstanceID)
	sendSuccess(w, "Sync started")
}

func syncFromMaster(instanceID string) {
	masterIP := getMasterIP()
	if masterIP == "" {
		broadcastLog(instanceID, "ERROR: [Sync] No master node found in config")
		return
	}

	broadcastLog(instanceID, fmt.Sprintf("INFO: [Sync] Pulling from master %s...", masterIP))

	// Fetch instance metadata from master via HMAC
	metaURL := fmt.Sprintf("http://%s:%d/api/v1/master/instances/metadata?id=%s", masterIP, s.Config().ManagePort, instanceID)
	resp, err := hmacGet(metaURL)
	if err != nil {
		broadcastLog(instanceID, fmt.Sprintf("ERROR: [Sync] Failed to fetch metadata from master: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		broadcastLog(instanceID, fmt.Sprintf("ERROR: [Sync] Master returned %d for metadata", resp.StatusCode))
		return
	}

	var masterMeta types.InstanceMetadata
	if err := json.NewDecoder(resp.Body).Decode(&masterMeta); err != nil {
		broadcastLog(instanceID, fmt.Sprintf("ERROR: [Sync] Failed to parse metadata: %v", err))
		return
	}

	// H5: Uses Releases[len-1] which may be a commit with empty FileName.
	// Should scan backward for first packet with non-empty FileName (see releaseDownloadHandler).
	var latestRelease *types.ReleaseEntry
	for i := len(masterMeta.Releases) - 1; i >= 0; i-- {
		if masterMeta.Releases[i].FileName != "" {
			latestRelease = &masterMeta.Releases[i]
			break
		}
	}

	if latestRelease != nil {
		releasePath := filepath.Join(".", "instances", instanceID, "releases", latestRelease.FileName)

		os.MkdirAll(filepath.Join(".", "instances", instanceID, "releases"), 0755)

		// Download the release zip from master
		dlURL := fmt.Sprintf("http://%s:%d/api/v1/master/releases/download?id=%s", masterIP, s.Config().ManagePort, instanceID)
		dlResp, err := hmacGet(dlURL)
		if err != nil {
			broadcastLog(instanceID, fmt.Sprintf("ERROR: [Sync] Failed to download release: %v", err))
			return
		}
		defer dlResp.Body.Close()

		if dlResp.StatusCode != 200 {
			broadcastLog(instanceID, fmt.Sprintf("ERROR: [Sync] Master returned %d for release download", dlResp.StatusCode))
			return
		}

		tmpPath := releasePath + ".tmp"
		out, err := os.Create(tmpPath)
		if err != nil {
			broadcastLog(instanceID, fmt.Sprintf("ERROR: [Sync] Failed to create temp file: %v", err))
			return
		}
		if _, err := io.Copy(out, dlResp.Body); err != nil {
			out.Close()
			os.Remove(tmpPath)
			broadcastLog(instanceID, fmt.Sprintf("ERROR: [Sync] Failed to write release file: %v", err))
			return
		}
		out.Close()

		if err := os.Rename(tmpPath, releasePath); err != nil {
			if copyErr := copyFile(tmpPath, releasePath); copyErr != nil {
				broadcastLog(instanceID, fmt.Sprintf("ERROR: [Sync] Failed to move release: %v", copyErr))
			}
			os.Remove(tmpPath)
		}

		// Extract the zip into instance files directory
		filesDir := filepath.Join(".", "instances", instanceID, "files")
		os.RemoveAll(filesDir)
		os.MkdirAll(filesDir, 0755)

		zr, err := zip.OpenReader(releasePath)
		if err != nil {
			broadcastLog(instanceID, fmt.Sprintf("ERROR: [Sync] Failed to open release zip: %v", err))
			return
		}
		defer zr.Close()

		filesAbs, err := filepath.Abs(filesDir)
		if err != nil {
			broadcastLog(instanceID, "ERROR: [Sync] Failed to get absolute path of files dir")
			return
		}
		for _, f := range zr.File {
			name := f.Name
			if name == "manifest.json" {
				continue
			}
			if strings.HasPrefix(name, "overrides/") {
				name = strings.TrimPrefix(name, "overrides/")
			}

			dst := filepath.Join(filesDir, name)
			absDst, err := filepath.Abs(dst)
			if err != nil {
				continue
			}
			if !strings.HasPrefix(absDst, filesAbs+string(filepath.Separator)) && absDst != filesAbs {
				continue
			}
			if f.FileInfo().IsDir() {
				os.MkdirAll(dst, 0755)
				continue
			}
			os.MkdirAll(filepath.Dir(dst), 0755)

			src, err := f.Open()
			if err != nil {
				continue
			}
			out, err := os.Create(dst)
			if err != nil {
				src.Close()
				continue
			}
			if _, err := io.Copy(out, src); err != nil {
				broadcastLog(instanceID, fmt.Sprintf("WARN: [Sync] Copy error: %v", err))
			}
			src.Close()
			out.Close()
		}

		// Save local metadata (strip release history for slaves)
		masterMeta.Releases = nil
		saveInstanceMetadata(instanceID, masterMeta)

		broadcastLog(instanceID, fmt.Sprintf("SUCCESS: [Sync] Synced to version %s", masterMeta.ActiveVersion))
	} else {
		broadcastLog(instanceID, "WARN: [Sync] Master has no releases for this instance")
	}
}

func copyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()

	_, err = io.Copy(d, s)
	return err
}

// M2: getMasterIP logic is fragile — assumes "not me = master".
// Should use explicit role field in types.NodeConfig instead.
func getMasterIP() string {
	for _, node := range s.Config().Nodes {
		if node.Name == "master" || node.DisplayName == "Master" {
			return node.IP
		}
	}
	// Fallback to old behavior
	for _, node := range s.Config().Nodes {
		if node.IP != s.Config().PublicIP {
			return node.IP
		}
	}
	return ""
}

func slaveStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	instances := make(map[string]string)
	entries, err := os.ReadDir("./instances")
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to read instances")
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		meta, err := loadInstanceMetadata(e.Name())
		if err != nil {
			continue
		}
		instances[e.Name()] = meta.ActiveVersion
	}

	sendSuccess(w, map[string]any{
		"node_name":   s.Config().NodeName,
		"instances":   instances,
		"config_hash": computeConfigHash(),
		"uptime":      int64(time.Since(s.startTime).Seconds()),
	})
}

func slaveConfigSyncHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		SecretKey      string       `json:"secret_key"`
		JWTSecret      string       `json:"jwt_secret"`
		PreAuthSecret  string       `json:"pre_auth_secret"`
		PreAuthEnabled bool         `json:"pre_auth_enabled"`
		Nodes          []types.NodeConfig `json:"nodes"`
		DistPolicy     string       `json:"dist_policy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	cfg := s.Config()
	if req.SecretKey != "" {
		cfg.SecretKey = req.SecretKey
	}
	if req.JWTSecret != "" {
		cfg.JWTSecret = req.JWTSecret
		auth.InitJWT(req.JWTSecret)
	}
	if req.PreAuthSecret != "" {
		cfg.PreAuthSecret = req.PreAuthSecret
	}
	cfg.PreAuthEnabled = req.PreAuthEnabled
	if len(req.Nodes) > 0 {
		cfg.Nodes = req.Nodes
	}
	if req.DistPolicy != "" {
		cfg.DistPolicy = req.DistPolicy
	}

	s.SetConfig(cfg)
	if err := saveConfig(); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to save synced config")
		return
	}

	broadcastLog("__system__", "SUCCESS: [ConfigSync] types.Config updated from master")
	sendSuccess(w, "Config synced")
}
