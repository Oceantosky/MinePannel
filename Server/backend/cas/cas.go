package cas

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ManifestFile represents a single file in the version snapshot
type ManifestFile struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

// Manifest represents a complete snapshot of an instance
type Manifest struct {
	VersionID string                  `json:"version_id"`
	Timestamp int64                   `json:"timestamp"`
	Files     map[string]ManifestFile `json:"files"`
}

// ComputeHash calculates the SHA-256 hash of a file
func ComputeHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// StoreBlob hashes the file and copies it to the CAS data pool
func StoreBlob(filePath string, objectsDir string) (string, int64, error) {
	hash, err := ComputeHash(filePath)
	if err != nil {
		return "", 0, err
	}

	stat, err := os.Stat(filePath)
	if err != nil {
		return hash, 0, err
	}
	size := stat.Size()

	// e.g., data/objects/a1/a1b2c3d4...blob
	blobDir := filepath.Join(objectsDir, hash[:2])
	if err := os.MkdirAll(blobDir, 0755); err != nil {
		return hash, size, err
	}

	blobPath := filepath.Join(blobDir, hash+".blob")

	// If blob already exists, we skip copying (Extreme Deduplication)
	if _, err := os.Stat(blobPath); err == nil {
		return hash, size, nil
	}

	// Copy file to blob pool
	src, err := os.Open(filePath)
	if err != nil {
		return hash, size, err
	}
	defer src.Close()

	dst, err := os.Create(blobPath)
	if err != nil {
		return hash, size, err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return hash, size, err
	}

	return hash, size, nil
}

// GenerateManifest scans the instance files and builds a manifest, storing new files to the object pool
func GenerateManifest(instanceID string, versionID string, filesDir string, objectsDir string) (Manifest, error) {
	manifest := Manifest{
		VersionID: versionID,
		Timestamp: time.Now().Unix(),
		Files:     make(map[string]ManifestFile),
	}

	err := filepath.Walk(filesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(filesDir, path)
		if err != nil {
			return err
		}

		// Convert backslashes to forward slashes for cross-platform consistency
		relPath = filepath.ToSlash(relPath)

		hash, size, err := StoreBlob(path, objectsDir)
		if err != nil {
			return err
		}

		manifest.Files[relPath] = ManifestFile{
			Hash: hash,
			Size: size,
		}
		return nil
	})

	return manifest, err
}

// SaveManifest writes the manifest to the instance's manifests directory
func SaveManifest(instanceID string, manifest Manifest) error {
	manifestDir := filepath.Join(".", "instances", instanceID, "manifests")
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		return err
	}
	manifestPath := filepath.Join(manifestDir, "manifest_"+manifest.VersionID+".json")

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(manifestPath, data, 0644)
}
