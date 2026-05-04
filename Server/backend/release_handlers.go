package main

import (
	"archive/zip"
	"minepannel-v6/cas"
	"minepannel-v6/types"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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
	var latest types.ReleaseEntry
	for i := len(meta.Releases) - 1; i >= 0; i-- {
		if meta.Releases[i].FileName != "" {
			latest = meta.Releases[i]
			break
		}
	}
	if latest.FileName == "" {
		sendError(w, http.StatusNotFound, "No packet release available")
		return
	}
	if strings.Contains(latest.FileName, "..") || strings.Contains(latest.FileName, "/") || strings.Contains(latest.FileName, "\\") {
		sendError(w, http.StatusForbidden, "Invalid release filename")
		return
	}
	releasePath := filepath.Join(".", "instances", id, "releases", latest.FileName)
	if _, err := os.Stat(releasePath); os.IsNotExist(err) {
		sendError(w, http.StatusNotFound, "Release file not found")
		return
	}
	encodedName := url.PathEscape(latest.FileName)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", latest.FileName, encodedName))
	w.Header().Set("Content-Type", "application/zip")
	http.ServeFile(w, r, releasePath)
}

type downloadTicketInfo struct {
	InstanceID string
	ExpiresAt  time.Time
}

func releaseTicketHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if err := validateInstanceID(id); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid instance id")
		return
	}
	ticket := generateRandomID() + generateRandomID()
	s.downloadTickets.Store(ticket, downloadTicketInfo{
		InstanceID: id,
		ExpiresAt:  time.Now().Add(5 * time.Minute),
	})
	sendSuccess(w, map[string]string{"ticket": ticket})
}

func releaseDownloadByTicketHandler(w http.ResponseWriter, r *http.Request) {
	ticket := r.URL.Query().Get("ticket")
	if ticket == "" {
		sendError(w, http.StatusUnauthorized, "Missing ticket")
		return
	}
	val, ok := s.downloadTickets.Load(ticket)
	if !ok {
		sendError(w, http.StatusUnauthorized, "Invalid or expired ticket")
		return
	}
	info := val.(downloadTicketInfo)
	if time.Now().After(info.ExpiresAt) {
		s.downloadTickets.Delete(ticket)
		sendError(w, http.StatusUnauthorized, "Ticket expired")
		return
	}
	// Consume ticket (one-time use)
	s.downloadTickets.Delete(ticket)

	id := info.InstanceID

	meta, err := loadInstanceMetadata(id)
	if err != nil || len(meta.Releases) == 0 {
		sendError(w, http.StatusNotFound, "No release available")
		return
	}
	var latest types.ReleaseEntry
	for i := len(meta.Releases) - 1; i >= 0; i-- {
		if meta.Releases[i].FileName != "" {
			latest = meta.Releases[i]
			break
		}
	}
	if latest.FileName == "" {
		sendError(w, http.StatusNotFound, "No packet release available")
		return
	}
	if strings.Contains(latest.FileName, "..") || strings.Contains(latest.FileName, "/") || strings.Contains(latest.FileName, "\\") {
		sendError(w, http.StatusForbidden, "Invalid release filename")
		return
	}
	releasePath := filepath.Join(".", "instances", id, "releases", latest.FileName)
	if _, err := os.Stat(releasePath); os.IsNotExist(err) {
		sendError(w, http.StatusNotFound, "Release file not found")
		return
	}
	encodedName := url.PathEscape(latest.FileName)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", latest.FileName, encodedName))
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

	if _, loaded := s.instanceLocks.LoadOrStore(id, "Deploying"); loaded {
		sendError(w, http.StatusConflict, "Instance is already processing")
		return
	}
	sendSuccess(w, "Upload & Deploy Started")

	go func() {
		defer func() { s.instanceLocks.Delete(id); os.Remove(z) }()

		broadcastLog(id, "INFO: Unzipping uploaded pack...")
		stagingDir := filepath.Join("./instances", id, "staging_unzip")
		os.RemoveAll(stagingDir)
		os.MkdirAll(stagingDir, 0755)

		zr, err := zip.OpenReader(z)
		if err != nil {
			broadcastLog(id, "ERROR: Failed to read ZIP")
			return
		}
		stagingAbs, err := filepath.Abs(stagingDir)
		if err != nil {
			broadcastLog(id, "ERROR: Failed to get absolute path of staging dir")
			return
		}
		for _, f := range zr.File {
			p := filepath.Join(stagingDir, f.Name)
			absP, err := filepath.Abs(p)
			if err != nil {
				continue
			}
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
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "Name:") {
					name := strings.TrimPrefix(line, "Name:")
					meta.DisplayName = pclNameCleanRe.ReplaceAllString(name, "")
				}
			}
		}

		// Extract MC version & loader from modpack manifests before ProcessModpack deletes them
		if mrPath := filepath.Join(stagingDir, "modrinth.index.json"); checkFile(mrPath) {
			if raw, err := os.ReadFile(mrPath); err == nil {
				var mrIndex struct {
					Dependencies map[string]string `json:"dependencies"`
				}
				if json.Unmarshal(raw, &mrIndex) == nil {
					if v, ok := mrIndex.Dependencies["minecraft"]; ok {
						meta.McVersion = v
					}
					for _, loader := range []string{"forge", "fabric-loader", "quilt-loader", "neoforge"} {
						if lv, ok := mrIndex.Dependencies[loader]; ok && lv != "" {
							meta.ModLoader = loader + "-" + lv
							break
						}
					}
				}
			}
		} else if cfPath := filepath.Join(stagingDir, "manifest.json"); checkFile(cfPath) {
			if raw, err := os.ReadFile(cfPath); err == nil {
				var cfManifest struct {
					Minecraft struct {
						Version    string `json:"version"`
						ModLoaders []struct {
							ID      string `json:"id"`
							Primary bool   `json:"primary"`
						} `json:"modLoaders"`
					} `json:"minecraft"`
				}
				if json.Unmarshal(raw, &cfManifest) == nil {
					meta.McVersion = cfManifest.Minecraft.Version
					for _, ml := range cfManifest.Minecraft.ModLoaders {
						if ml.Primary || meta.ModLoader == "" {
							meta.ModLoader = ml.ID
						}
					}
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

		v := computeNextVersion(meta, false)
		broadcastLog(id, "INFO: Auto-committing initial snapshot [v"+v+"]...")
		manifest, err := cas.GenerateManifest(id, v, targetDir, "./data/objects")
		if err == nil {
			cas.SaveManifest(id, manifest)
			meta.ActiveVersion = v
			meta.Releases = append(meta.Releases, types.ReleaseEntry{
				VersionID: v,
				Type:      "commit",
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
	for _, domain := range []string{"modrinth.com", "curseforge.com", "mcimirror.top"} {
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
	client := s.httpClient
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

		// 自动触发 Commit 以更新 Manifest，确保客户端增量同步能看到新文件
		if _, loaded := s.instanceLocks.LoadOrStore(id, "Committing"); !loaded {
			go func() {
				defer s.instanceLocks.Delete(id)
				m, err := loadInstanceMetadata(id)
				if err != nil {
					return
				}
				v := computeNextVersion(m, false)
				broadcastLog(id, "INFO: Auto-committing market download [v"+v+"]...")
				manifest, err := cas.GenerateManifest(id, v, filepath.Join(".", "instances", id, "files"), "./data/objects")
				if err != nil {
					broadcastLog(id, "ERROR: Auto-commit failed: "+err.Error())
					return
				}
				cas.SaveManifest(id, manifest)
				m.ActiveVersion = v
				m.Releases = append(m.Releases, types.ReleaseEntry{
					VersionID: v,
					Type:      "commit",
					Message:   "Market: " + req.FileName,
					Time:      time.Now(),
				})
				saveInstanceMetadata(id, m)
				broadcastLog(id, "SUCCESS: Auto-commit complete [v"+v+"]")
			}()
		}
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
	client := s.httpClient
	resp, err := client.Get(proxyURL)
	if err != nil {
		sendError(w, http.StatusBadGateway, fmt.Sprintf("Proxy request failed: %v", err))
		return
	}
	defer resp.Body.Close()

	// Forward upstream status code and headers (header must be set before WriteHeader)
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
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
	client := s.httpClient
	resp, err := client.Get(proxyURL)
	if err != nil {
		sendError(w, http.StatusBadGateway, fmt.Sprintf("Proxy request failed: %v", err))
		return
	}
	defer resp.Body.Close()

	// Forward upstream status code and headers (header must be set before WriteHeader)
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		// Client disconnect during streaming is expected
	}
}
