package main

import (
	"bytes"
	"minepannel-v6/auth"
	"minepannel-v6/types"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// --- Heartbeat & types.Config Sync ---

func isLocalNode(node types.NodeConfig) bool {
	return node.IP == "127.0.0.1" || node.IP == s.Config().PublicIP
}

func computeConfigHash() string {
	h := sha256.New()
	h.Write([]byte(s.Config().SecretKey + s.Config().JWTSecret))
	return hex.EncodeToString(h.Sum(nil))
}

func probeNode(node types.NodeConfig) {
	start := time.Now()
	client := s.probeHTTPClient
	resp, err := client.Get(fmt.Sprintf("http://%s:%d/health", node.IP, s.Config().ManagePort))
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
	token := auth.GenerateMAC(s.Config().SecretKey, timestamp, method, path, bodyHash)

	req, err := http.NewRequest(method, fmt.Sprintf("http://%s:%d%s", node.IP, s.Config().ManagePort, path), nil)
	if err != nil {
		broadcastLog("__system__", fmt.Sprintf("WARN: [Heartbeat] %s create request failed: %v", node.DisplayName, err))
		return
	}
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
			sendPushToSlave(node.IP, instID, s.Config().SecretKey)
			syncedCount++
		}
	}

	broadcastLog("__system__", fmt.Sprintf("INFO: [Heartbeat] %s online | latency=%dms | uptime=%ds | instances=%d | synced=%d",
		node.DisplayName, latency, result.Data.Uptime, len(result.Data.Instances), syncedCount))
}

func heartbeatLoop(done <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			for _, node := range s.Config().Nodes {
				if isLocalNode(node) {
					continue
				}
				go probeNode(node)
			}
		}
	}
}

func sendConfigToSlave(node types.NodeConfig, signKey string) {
	timestamp := time.Now().Unix()
	path := "/api/v1/slave/config-sync"
	method := "POST"
	body := map[string]any{
		"secret_key":       s.Config().SecretKey,
		"jwt_secret":       s.Config().JWTSecret,
		"pre_auth_secret":  s.Config().PreAuthSecret,
		"pre_auth_enabled": s.Config().PreAuthEnabled,
		"nodes":            s.Config().Nodes,
		"dist_policy":      s.Config().DistPolicy,
	}
	bodyBytes, _ := json.Marshal(body)
	bodyHash := fmt.Sprintf("%x", sha256.Sum256(bodyBytes))
	token := auth.GenerateMAC(signKey, timestamp, method, path, bodyHash)

	req, err := http.NewRequest(method, fmt.Sprintf("http://%s:%d%s", node.IP, s.Config().ManagePort, path), bytes.NewReader(bodyBytes))
	if err != nil {
		broadcastLog("__system__", fmt.Sprintf("ERROR: [ConfigSync] Failed to build request for %s", node.DisplayName))
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Timestamp", fmt.Sprintf("%d", timestamp))
	req.Header.Set("Content-Type", "application/json")

	client := s.httpClient
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
