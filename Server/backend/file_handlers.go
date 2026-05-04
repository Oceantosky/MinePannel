package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
)

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
