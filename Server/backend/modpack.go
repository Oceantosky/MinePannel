package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type ModrinthIndex struct {
	Files []struct {
		Path      string   `json:"path"`
		Downloads []string `json:"downloads"`
		FileSize  int64    `json:"fileSize"`
	} `json:"files"`
}

type CFManifest struct {
	Files []struct {
		ProjectID int  `json:"projectID"`
		FileID    int  `json:"fileID"`
		Required  bool `json:"required"`
	} `json:"files"`
}

func ProcessModpack(id string, stagingDir string) {
	mrPackPath := filepath.Join(stagingDir, "modrinth.index.json")
	if _, err := os.Stat(mrPackPath); err == nil {
		broadcastLog(id, "INFO: Modrinth pack detected. Resolving dependencies...")
		processModrinth(id, stagingDir, mrPackPath)
	} else {
		cfPackPath := filepath.Join(stagingDir, "manifest.json")
		if _, err := os.Stat(cfPackPath); err == nil {
			broadcastLog(id, "INFO: CurseForge pack detected. Resolving dependencies...")
			processCurseForge(id, stagingDir, cfPackPath)
		}
	}

	// Merge overrides into staging root
	overridesDir := filepath.Join(stagingDir, "overrides")
	if _, err := os.Stat(overridesDir); err == nil {
		broadcastLog(id, "INFO: Merging overrides...")
		mergeDir(overridesDir, stagingDir)
		os.RemoveAll(overridesDir)
	}

	clientOverridesDir := filepath.Join(stagingDir, "client-overrides")
	if _, err := os.Stat(clientOverridesDir); err == nil {
		broadcastLog(id, "INFO: Merging client-overrides...")
		mergeDir(clientOverridesDir, stagingDir)
		os.RemoveAll(clientOverridesDir)
	}

	// Clean up pack descriptors
	os.Remove(mrPackPath)
	os.Remove(filepath.Join(stagingDir, "manifest.json"))
}

func processModrinth(id string, stagingDir string, packPath string) {
	var index ModrinthIndex
	b, err := os.ReadFile(packPath)
	if err != nil {
		return
	}
	if err := json.Unmarshal(b, &index); err != nil {
		broadcastLog(id, fmt.Sprintf("WARN: Failed to parse modrinth index: %v", err))
		return
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, 10) // 10 concurrent downloads
	total := len(index.Files)
	completed := 0
	var mu sync.Mutex

	for _, file := range index.Files {
		if len(file.Downloads) == 0 {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(f struct {
			Path      string   `json:"path"`
			Downloads []string `json:"downloads"`
			FileSize  int64    `json:"fileSize"`
		}) {
			defer wg.Done()
			defer func() { <-sem }()

			destPath := filepath.Join(stagingDir, filepath.FromSlash(f.Path))
			os.MkdirAll(filepath.Dir(destPath), 0755)

			// Fallback through mirrors
			success := false
			for _, url := range f.Downloads {
				resp, err := s.httpClient.Get(url)
				if err == nil && resp.StatusCode == 200 {
					out, err := os.Create(destPath)
					if err == nil {
						if _, err := io.Copy(out, resp.Body); err != nil {
							broadcastLog(id, fmt.Sprintf("WARN: Failed to write %s: %v", f.Path, err))
						}
						out.Close()
					}
					resp.Body.Close()
					success = true
					break // success
				}
				if resp != nil {
					resp.Body.Close()
				}
			}
			if !success {
				broadcastLog(id, fmt.Sprintf("ERROR: Failed to download %s from all mirrors", f.Path))
			}

			mu.Lock()
			completed++
			if completed%10 == 0 || completed == total {
				broadcastLog(id, fmt.Sprintf("INFO: Downloading Modrinth mods... %d/%d", completed, total))
			}
			mu.Unlock()
		}(file)
	}
	wg.Wait()
}

func processCurseForge(id string, stagingDir string, packPath string) {
	var manifest CFManifest
	b, err := os.ReadFile(packPath)
	if err != nil {
		return
	}
	if err := json.Unmarshal(b, &manifest); err != nil {
		broadcastLog(id, fmt.Sprintf("WARN: Failed to parse CF manifest: %v", err))
		return
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)
	total := len(manifest.Files)
	completed := 0
	var mu sync.Mutex

	for _, file := range manifest.Files {
		wg.Add(1)
		sem <- struct{}{}
		go func(projectID, fileID int) {
			defer wg.Done()
			defer func() { <-sem }()

			apiUrl := fmt.Sprintf("https://mod.mcimirror.top/curseforge/v1/mods/%d/files/%d", projectID, fileID)
			resp, err := s.httpClient.Get(apiUrl)
			if err != nil || resp.StatusCode != 200 {
				if resp != nil {
					resp.Body.Close()
				}
				return
			}

			var apiResp struct {
				Data struct {
					FileName    string `json:"fileName"`
					DownloadUrl string `json:"downloadUrl"`
				} `json:"data"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
				resp.Body.Close()
				broadcastLog(id, fmt.Sprintf("WARN: Failed to decode CF API response for mod %d: %v", fileID, err))
				return
			}
			resp.Body.Close()

			if apiResp.Data.DownloadUrl == "" && apiResp.Data.FileName != "" {
				idStr := fmt.Sprintf("%d", fileID)
				if len(idStr) >= 4 {
					apiResp.Data.DownloadUrl = fmt.Sprintf("https://edge.forgecdn.net/files/%s/%s/%s", idStr[:4], idStr[4:], apiResp.Data.FileName)
				}
			}

			if apiResp.Data.DownloadUrl != "" {
				destPath := filepath.Join(stagingDir, "mods", apiResp.Data.FileName)
				os.MkdirAll(filepath.Dir(destPath), 0755)

				dResp, dErr := s.httpClient.Get(apiResp.Data.DownloadUrl)
				if dErr == nil && dResp.StatusCode == 200 {
					out, err := os.Create(destPath)
					if err == nil {
						if _, err := io.Copy(out, dResp.Body); err != nil {
							broadcastLog(id, fmt.Sprintf("WARN: Failed to write %s: %v", apiResp.Data.FileName, err))
						}
						out.Close()
					}
				}
				if dResp != nil {
					dResp.Body.Close()
				}
			}

			mu.Lock()
			completed++
			if completed%10 == 0 || completed == total {
				broadcastLog(id, fmt.Sprintf("INFO: Downloading CurseForge mods... %d/%d", completed, total))
			}
			mu.Unlock()
		}(file.ProjectID, file.FileID)
	}
	wg.Wait()
}

func mergeDir(src string, dest string) {
	filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(src, path)
		destPath := filepath.Join(dest, rel)
		os.MkdirAll(filepath.Dir(destPath), 0755)
		if err := os.Rename(path, destPath); err != nil {
			srcFile, err := os.Open(path)
			if err != nil {
				return nil
			}
			dstFile, err := os.Create(destPath)
			if err != nil {
				srcFile.Close()
				return nil
			}
			io.Copy(dstFile, srcFile)
			srcFile.Close()
			dstFile.Close()
		}
		return nil
	})
}
