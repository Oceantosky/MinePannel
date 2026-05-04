package main

import (
	"archive/zip"
	"minepannel-v6/cas"
	"minepannel-v6/types"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// --- Handlers ---

func healthHandler(w http.ResponseWriter, r *http.Request) {
	sendSuccess(w, map[string]any{
		"status":    "healthy",
		"node":      s.Config().NodeName,
		"timestamp": time.Now().Unix(),
	})
}

func listInstancesHandler(w http.ResponseWriter, r *http.Request) {
	// Determine if JWT-authenticated user (for filtering)
	var allowedIDs map[string]bool
	if r.Header.Get("X-Auth-Method") == "jwt" && r.Header.Get("X-User-Role") != "admin" {
		userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
		user, err := s.store.GetUserByID(userID)
		if err == nil {
			var instances []string
			if err := json.Unmarshal([]byte(user.Instances), &instances); err == nil {
				allowedIDs = make(map[string]bool)
				for _, id := range instances {
					allowedIDs[id] = true
				}
			} else {
				fmt.Fprintf(os.Stderr, "Failed to unmarshal instances for user %d: %v\n", userID, err)
			}
		}
	}
	// Pre-auth instance filtering
	if r.Header.Get("X-Auth-Method") == "preauth" {
		allowed := r.Header.Get("X-Allowed-Instances")
		if allowed != "" {
			allowedIDs = make(map[string]bool)
			for _, id := range strings.Split(allowed, ",") {
				allowedIDs[id] = true
			}
		}
	}

	list := []types.Instance{}
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
		} else if t, ok := s.instanceLocks.Load(id); ok {
			if s, ok2 := t.(string); ok2 {
				status = s
			}
		}
		v := meta.ActiveVersion
		f := "full_client.zip"
		if v != "" {
			// Use actual filename from latest packet release
			for i := len(meta.Releases) - 1; i >= 0; i-- {
				if meta.Releases[i].FileName != "" {
					f = meta.Releases[i].FileName
					break
				}
			}
		}
		/// Scheme-less URL — PCL client prepends https:// or http:// as negotiated
		u := fmt.Sprintf("%s:%d/sync/%s/releases/%s", s.Config().PublicIP, s.Config().BusinessPort, id, f)
		if s.Config().DistPolicy == "cdn" && meta.CDNLink != "" {
			u = meta.CDNLink
		}
		// Collect mod filenames for client-side matching (avoids N+1 /info calls)
		var modList []string
		if entries, err := os.ReadDir(filepath.Join(".", "instances", id, "files", "mods")); err == nil {
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".jar") {
					modList = append(modList, e.Name())
				}
			}
		}

		list = append(list, types.Instance{
			ID:          id,
			DisplayName: meta.DisplayName,
			Status:      status,
			FullPackURL: u,
			McVersion:   meta.McVersion,
			Modloader:   meta.ModLoader,
			VersionID:   meta.ActiveVersion,
			ModList:     modList,
		})
	}
	sendSuccess(w, map[string]any{"instance_list": list})
}

// infoHandler returns the information database (manifest) for a specific instance.
func infoHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		sendError(w, http.StatusBadRequest, "Missing id parameter")
		return
	}
	if err := validateInstanceID(id); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid instance id")
		return
	}

	meta, err := loadInstanceMetadata(id)
	if err != nil {
		sendError(w, http.StatusNotFound, "Instance not found")
		return
	}

	manifestPath := filepath.Join(".", "instances", id, "manifests", "manifest_"+meta.ActiveVersion+".json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		sendError(w, http.StatusNotFound, "Manifest not found for active version")
		return
	}

	var manifest cas.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to parse manifest")
		return
	}

	sendSuccess(w, map[string]any{
		"instance_id": id,
		"version_id":  manifest.VersionID,
		"mc_version":  meta.McVersion,
		"modloader":   meta.ModLoader,
		"files":       manifest.Files,
	})
}

func pushHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if err := validateInstanceID(id); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid instance id")
		return
	}
	broadcastLog(id, "INFO: [Dist] Signaling all slave nodes...")
	for _, node := range s.Config().Nodes {
		if node.IP != "127.0.0.1" && node.IP != s.Config().PublicIP {
			broadcastLog(id, "DEBUG: Notify -> "+node.DisplayName)
			sendPushToSlave(node.IP, id, s.Config().SecretKey)
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
	var statuses = make([]NodeStatus, len(s.Config().Nodes))
	var wg sync.WaitGroup
	for i, node := range s.Config().Nodes {
		wg.Add(1)
		go func(idx int, n types.NodeConfig) {
			defer wg.Done()
			start := time.Now()
			client := s.probeHTTPClient
			resp, err := client.Get(fmt.Sprintf("http://%s:%d/health", n.IP, s.Config().ManagePort))
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
	sendSuccess(w, s.Config())
}

func publicConfigHandler(w http.ResponseWriter, r *http.Request) {
	sendSuccess(w, map[string]any{
		"dist_policy":      s.Config().DistPolicy,
		"public_ip":        s.Config().PublicIP,
		"node_name":        s.Config().NodeName,
		"role":             s.Config().Role,
		"pre_auth_enabled": s.Config().PreAuthEnabled,
	})
}

func configWriteHandler(w http.ResponseWriter, r *http.Request) {
	var c types.Config
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
	oldSecretKey := s.Config().SecretKey

	// Preserve existing JWTSecret if none provided to prevent lockout
	if c.JWTSecret == "" {
		c.JWTSecret = s.Config().JWTSecret
	}
	// Preserve existing PreAuthSecret if none provided
	if c.PreAuthSecret == "" {
		c.PreAuthSecret = s.Config().PreAuthSecret
	}
	// Preserve fields that shouldn't be overwritten by SettingsView
	if c.Role == "" {
		c.Role = s.Config().Role
	}
	if c.PublicIP == "" {
		c.PublicIP = s.Config().PublicIP
	}
	if c.NodeName == "" {
		c.NodeName = s.Config().NodeName
	}
	if len(c.Nodes) == 0 {
		c.Nodes = s.Config().Nodes
	}

	s.SetConfig(c)
	if err := saveConfig(); err != nil {
		sendError(w, http.StatusInternalServerError, "Failed to save config")
		return
	}

	if oldSecretKey != s.Config().SecretKey {
		broadcastLog("__system__", "INFO: [types.Config] PSK changed, propagating to slave nodes...")
		for _, node := range s.Config().Nodes {
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
	saveInstanceMetadata(id, types.InstanceMetadata{DisplayName: name})

	// Auto-assign to creating JWT user
	if r.Header.Get("X-Auth-Method") == "jwt" {
		userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
		user, err := s.store.GetUserByID(userID)
		if err == nil && user.Role != "admin" {
			var instances []string
			json.Unmarshal([]byte(user.Instances), &instances)
			instances = append(instances, id)
			s.store.UpdateUser(userID, user.Username, user.Role, user.IsActive, instances)
		}
	}

	broadcastLog(id, "SUCCESS: types.Instance created with display name ["+name+"]")
	sendSuccess(w, map[string]string{"id": id})
}

func deleteInstanceHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if err := validateInstanceID(id); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid instance id")
		return
	}
	os.RemoveAll(filepath.Join("./instances", id))

	// M1: Clean up the instance from all users
	if users, err := s.store.ListUsers(); err == nil {
		for _, u := range users {
			var instances []string
			if err := json.Unmarshal([]byte(u.Instances), &instances); err == nil {
				var newInstances []string
				changed := false
				for _, instID := range instances {
					if instID == id {
						changed = true
					} else {
						newInstances = append(newInstances, instID)
					}
				}
				if changed {
					s.store.UpdateUser(u.ID, u.Username, u.Role, u.IsActive, newInstances)
				}
			}
		}
	}

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
	var m types.InstanceMetadata
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	// Backend Enforcement: If global CDN policy is not enabled, forcefully clear any submitted CDN link
	if s.Config().DistPolicy != "cdn" {
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

func computeNextVersion(meta types.InstanceMetadata, isPacket bool) string {
	packets := 0
	commitsSincePacket := 0
	for _, r := range meta.Releases {
		if r.Type == "packet" {
			packets++
			commitsSincePacket = 0
		} else if r.Type == "commit" {
			commitsSincePacket++
		}
	}
	if isPacket {
		packets++
		commitsSincePacket = 0
	} else {
		commitsSincePacket++
	}
	return fmt.Sprintf("%d.%d", packets, commitsSincePacket)
}

func commitHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if err := validateInstanceID(id); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid instance id")
		return
	}
	if _, loaded := s.instanceLocks.LoadOrStore(id, "Committing"); loaded {
		sendError(w, http.StatusConflict, "Instance is already processing")
		return
	}
	go func() {
		defer s.instanceLocks.Delete(id)
		m, err := loadInstanceMetadata(id)
		if err != nil {
			return
		}
		v := computeNextVersion(m, false)
		broadcastLog(id, "INFO: Generating cas.Manifest [v"+v+"]...")
		manifest, err := cas.GenerateManifest(id, v, filepath.Join("./instances", id, "files"), "./data/objects")
		if err == nil {
			cas.SaveManifest(id, manifest)
			broadcastLog(id, "SUCCESS: cas.Manifest created and objects deduplicated.")
			m.ActiveVersion = v
			m.Releases = append(m.Releases, types.ReleaseEntry{
				VersionID: v,
				Type:      "commit",
				Message:   fmt.Sprintf("Commit: %d files tracked", len(manifest.Files)),
				Time:      time.Now(),
			})
			saveInstanceMetadata(id, m)
		} else {
			broadcastLog(id, "ERROR: "+err.Error())
		}
	}()
	sendSuccess(w, "Commit Started")
}

func packetUpHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if err := validateInstanceID(id); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid instance id")
		return
	}
	if _, loaded := s.instanceLocks.LoadOrStore(id, "Packing"); loaded {
		sendError(w, http.StatusConflict, "Instance is already processing")
		return
	}
	go func() {
		defer s.instanceLocks.Delete(id)
		m, err := loadInstanceMetadata(id)
		if err != nil {
			return
		}
		v := computeNextVersion(m, true)
		safeName := "Client"
		if m.DisplayName != "" {
			safeName = strings.ReplaceAll(m.DisplayName, " ", "_")
			// Remove any characters that are not alphanumeric, underscore, dash, or Chinese
			safeName = safeFilenameRe.ReplaceAllString(safeName, "")
		}
		if safeName == "" {
			safeName = "Client"
		}
		f := fmt.Sprintf("%s_v%s.zip", safeName, v)
		t := filepath.Join("./instances", id, "releases", f)
		os.MkdirAll(filepath.Dir(t), 0755)
		broadcastLog(id, "INFO: Creating CDN Base Release [v"+v+"]...")
		err = zipSource(id, filepath.Join("./instances", id, "files"), t, m)
		if err != nil {
			broadcastLog(id, "ERROR: Failed to create zip: "+err.Error())
		} else {
			s, _ := os.Stat(t)
			m.Releases = append(m.Releases, types.ReleaseEntry{
				VersionID: v,
				Type:      "packet",
				Message:   fmt.Sprintf("Packet Up: %s (%.1f MB)", f, float64(s.Size())/(1024*1024)),
				FileName:  f,
				Size:      s.Size(),
				Time:      time.Now(),
			})
			m.ActiveVersion = v
			saveInstanceMetadata(id, m)
			broadcastLog(id, "SUCCESS: CDN Base Release published.")
		}
		s.instanceLocks.Delete(id)
	}()
	sendSuccess(w, "Packet Up Accepted")
}

func sseHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan string, 10)
	s.logMu.Lock()
	actual, _ := s.logChannels.LoadOrStore(id, []chan string{})
	if chs, ok := actual.([]chan string); ok {
		s.logChannels.Store(id, append(chs, ch))
	} else {
		s.logChannels.Store(id, []chan string{ch})
	}
	s.logMu.Unlock()

	defer func() {
		s.logMu.Lock()
		if a, ok := s.logChannels.Load(id); ok {
			if cs, ok := a.([]chan string); ok {
				var n []chan string
				for _, c := range cs {
					if c != ch {
						n = append(n, c)
					}
				}
				s.logChannels.Store(id, n)
			}
		}
		s.logMu.Unlock()
		close(ch)
	}()

	if hist, ok := s.logHistory.Load(id); ok {
		if msgs, ok := hist.([]string); ok {
			for _, msg := range msgs {
				fmt.Fprintf(w, "data: %s\n\n", msg)
			}
		}
	}
	f, _ := w.(http.Flusher)
	f.Flush()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if _, err := fmt.Fprintf(w, ": ping\n\n"); err != nil {
				return
			}
			f.Flush()
		case m := <-ch:
			if _, err := fmt.Fprintf(w, "data: %s\n\n", m); err != nil {
				return
			}
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
		if _, locked := s.instanceLocks.Load(id); locked {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Instance is currently locked for maintenance (Status: 503)"))
			return
		}
		meta, err := loadInstanceMetadata(id)
		if err != nil || meta.IsPaused {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Instance is paused or path is invalid"))
			return
		}

		// Resolve relative path safely (strip instance id prefix from URL)
		relPath := strings.TrimPrefix(r.URL.Path, "/sync/"+id+"/")
		if relPath == "" || strings.Contains(relPath, "..") || strings.Contains(relPath, "\\") {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Invalid path"))
			return
		}

		filePath := filepath.Join(".", "instances", id, filepath.FromSlash(relPath))
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		basePath, err := filepath.Abs(filepath.Join(".", "instances", id))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !strings.HasPrefix(absPath, basePath) {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Path traversal detected"))
			return
		}

		// Use http.ServeFile directly — preserves sendfile() zero-copy on Linux
		http.ServeFile(w, r, absPath)
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
	w.Write([]byte("Instance is paused or path is invalid"))
}

func zipSource(id, src, dst string, meta types.InstanceMetadata) error {
	z, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer z.Close()

	a := zip.NewWriter(z)
	defer a.Close()

	addFile := func(name string, content string) error {
		h := &zip.FileHeader{Name: name, Method: zip.Deflate, Modified: time.Now()}
		w, err := a.CreateHeader(h)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, content)
		return err
	}

	clientSecret := fmt.Sprintf("%x", sha256.Sum256([]byte(s.Config().SecretKey+"_CLIENT")))[:16]
	pclIniContent := fmt.Sprintf("Version:CloudAbroad\nSecret:%s\nName:%s (%s)\nInfo:由 CloudAbroad 驱动的高速同步客户端\nSyncUrl:%s:%d/api/v1/sync/info?id=%s\nSyncPolicy:Enforce\n",
		clientSecret, meta.DisplayName, meta.ActiveVersion, s.Config().PublicIP, s.Config().BusinessPort, id)

	broadcastLog(id, "DEBUG: Injecting PCL.ini...")
	if err := addFile("PCL.ini", pclIniContent); err != nil {
		return err
	}

	// Determine ZIP structure: CurseForge format if MC version is known, otherwise flat
	overridesPrefix := ""
	if meta.McVersion != "" {
		overridesPrefix = "overrides/"
		broadcastLog(id, "INFO: Packaging as CurseForge modpack (MC "+meta.McVersion+")")

		loaderArray := "[]"
		if meta.ModLoader != "" {
			loaderArray = fmt.Sprintf(`[{"id":"%s","primary":true}]`, meta.ModLoader)
		}
		manifestContent := fmt.Sprintf(`{
  "minecraft": {
    "version": "%s",
    "modLoaders": %s
  },
  "manifestType": "minecraftModpack",
  "manifestVersion": 1,
  "name": "%s",
  "version": "1.0.0",
  "author": "MinePannel",
  "files": [],
  "overrides": "overrides"
}`, meta.McVersion, loaderArray, meta.DisplayName)
		if err := addFile("manifest.json", manifestContent); err != nil {
			return err
		}
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
		h.Name = filepath.ToSlash(overridesPrefix + rel)
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

