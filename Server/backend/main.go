package main

import (
	"minepannel-v6/auth"
	"minepannel-v6/middleware"
	"minepannel-v6/store"
	"minepannel-v6/types"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
)

type metadataCacheEntry struct {
	meta     types.InstanceMetadata
	cachedAt time.Time
}

// Server holds all application state, replacing package-level globals.
type Server struct {
	store           *store.Store
	config          types.Config
	configMu        sync.RWMutex
	instanceLocks   sync.Map
	logChannels     sync.Map
	logHistory      sync.Map
	logMu           sync.Mutex
	downloadTickets sync.Map
	metadataCache   sync.Map
	startTime       time.Time
	responseBytes   int64
	httpClient      *http.Client
	probeHTTPClient *http.Client
	heartbeatDone   chan struct{}
}

var safeFilenameRe = regexp.MustCompile(`[^a-zA-Z0-9_\-\x{4e00}-\x{9fa5}]`)
var pclNameCleanRe = regexp.MustCompile(`\s*\([^)]+\)$`)

// s is the package-level server instance, used during transition to methods.
var s *Server

func (s *Server) Config() types.Config {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.config
}

func (s *Server) SetConfig(c types.Config) {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	s.config = c
}

// NewServer creates a Server, loads config, and opens the database.
func NewServer(dbPath string) (*Server, error) {
	srv := &Server{
		startTime:       time.Now(),
		httpClient:      &http.Client{Timeout: 30 * time.Second},
		probeHTTPClient: &http.Client{Timeout: 5 * time.Second},
	}
	// Set the global for package-level functions during transition
	s = srv

	// Load or create config (calls auth.InitJWT internally)
	loadConfig()

	// Open database
	var err error
	srv.store, err = store.NewStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return srv, nil
}

func (srv *Server) Close() error {
	return srv.store.Close()
}

func checkFile(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func validateInstanceID(id string) error {
	decoded, err := url.QueryUnescape(id)
	if err == nil {
		id = decoded
	}
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
	token := auth.GenerateMAC(s.Config().SecretKey, timestamp, "GET", reqPath, bodyHash)
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Timestamp", fmt.Sprintf("%d", timestamp))
	return s.httpClient.Do(req)
}

// setJWTCookie writes the JWT as an httpOnly, SameSite=Strict cookie.
func setJWTCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt_token",
		Value:    token,
		Path:     "/",
		MaxAge:   86400, // 24 hours
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   false, // set to true behind TLS termination
	})
}

// clearJWTCookie removes the JWT cookie.
func clearJWTCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
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
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%032x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func broadcastLog(id, message string) {
	s.logMu.Lock()
	defer s.logMu.Unlock()
	val, _ := s.logHistory.LoadOrStore(id, []string{})
	if history, ok := val.([]string); ok {
		if len(history) > 20 {
			history = history[1:]
		}
		s.logHistory.Store(id, append(history, message))
	} else {
		s.logHistory.Store(id, []string{message})
	}
	if chsVal, ok := s.logChannels.Load(id); ok {
		if chs, ok := chsVal.([]chan string); ok {
			for _, ch := range chs {
				select {
				case ch <- message:
				default:
				}
			}
		}
	}
}

func loadInstanceMetadata(id string) (types.InstanceMetadata, error) {
	if entry, ok := s.metadataCache.Load(id); ok {
		if e, ok := entry.(metadataCacheEntry); ok && time.Since(e.cachedAt) < 1*time.Second {
			return e.meta, nil
		}
	}
	var m types.InstanceMetadata
	f, err := os.ReadFile(filepath.Join("./instances", id, "instance.json"))
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(f, &m); err != nil {
		return m, fmt.Errorf("parse instance metadata: %w", err)
	}
	s.metadataCache.Store(id, metadataCacheEntry{meta: m, cachedAt: time.Now()})
	return m, nil
}

func saveInstanceMetadata(id string, m types.InstanceMetadata) {
	d, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		broadcastLog(id, fmt.Sprintf("ERROR: Failed to marshal metadata: %v", err))
		return
	}
	if err := os.WriteFile(filepath.Join("./instances", id, "instance.json"), d, 0644); err != nil {
		broadcastLog(id, fmt.Sprintf("ERROR: Failed to write metadata: %v", err))
	}
	s.metadataCache.Store(id, metadataCacheEntry{meta: m, cachedAt: time.Now()})
}

func loadConfig() {
	var c types.Config
	f, err := os.ReadFile("config.json")
	if err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Failed to read config.json: %v\n", err)
		os.Exit(1)
	}
	if err == nil {
		if err := json.Unmarshal(f, &c); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse config.json: %v\n", err)
			os.Exit(1)
		}
	}

	modified := false
	if len(c.Nodes) == 0 {
		c.Nodes = []types.NodeConfig{{Name: "bj", DisplayName: "Master", IP: "127.0.0.1", Flag: "CN"}}
		modified = true
	}
	if c.NodeName == "" {
		c.NodeName = "Default_Node"
		modified = true
	}
	if c.Role == "" {
		c.Role = "master"
		modified = true
	}
	if c.SecretKey == "" {
		c.SecretKey = "CloudAbroad_Secret_" + generateRandomID()
		modified = true
	}
	if c.PublicIP == "" {
		c.PublicIP = "127.0.0.1"
		modified = true
	}
	if c.DistPolicy == "" {
		c.DistPolicy = "direct"
		modified = true
	}
	if c.JWTSecret == "" {
		c.JWTSecret = "CloudAbroad_JWT_" + generateRandomID()
		modified = true
	}
	if c.PreAuthSecret == "" {
		c.PreAuthSecret = "MinePannel_PreAuth_" + generateRandomID()
		modified = true
	}
	if c.ManagePort == 0 {
		c.ManagePort = 55000
		modified = true
	}
	if c.BusinessPort == 0 {
		c.BusinessPort = 55001
		modified = true
	}

	s.SetConfig(c)
	if modified {
		if err := saveConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to save initial config: %v\n", err)
		}
	}
	auth.InitJWT(c.JWTSecret)
}

func saveConfig() error {
	d, err := json.MarshalIndent(s.Config(), "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal config: %v\n", err)
		return err
	}
	if err := os.WriteFile("config.json", d, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write config.json: %v\n", err)
		return err
	}
	return nil
}

// --- Main ---

func main() {
	srv, err := NewServer("./data.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize server: %v\n", err)
		os.Exit(1)
	}
	defer srv.Close()

	mw := middleware.New(s.store, s.Config)

	mux := http.NewServeMux()
	dataMux := http.NewServeMux()

	// --- Public endpoints (no auth required) ---
	mux.HandleFunc("/health", mw.WithCORS(healthHandler))
	dataMux.HandleFunc("/health", mw.WithCORS(healthHandler))
	mux.HandleFunc("/api/v1/master/console/stream", mw.WithCORS(sseHandler))

	// --- Pre-Auth client endpoints (PCL CE business logic - Data Plane) ---
	dataMux.HandleFunc("/api/v1/preauth/status", mw.WithCORS(preAuthStatusHandler))
	dataMux.HandleFunc("/api/v1/preauth/bind", mw.WithCORS(preAuthBindHandler))

	// --- Auth endpoints ---
	mux.HandleFunc("/api/v1/auth/login", mw.WithCORS(loginHandler(s.store)))
	mux.HandleFunc("/api/v1/auth/status", mw.WithCORS(mw.WithJWT(authStatusHandler(s.store))))
	mux.HandleFunc("/api/v1/auth/change-password", mw.WithCORS(mw.WithJWT(changePasswordHandler(s.store))))
	mux.HandleFunc("/api/v1/auth/logout", mw.WithCORS(logoutHandler(s.store)))

	// --- Admin endpoints (JWT + admin only) ---
	mux.HandleFunc("/api/v1/admin/users", mw.WithCORS(mw.WithJWT(mw.AdminOnly(adminUsersDispatchHandler(s.store)))))
	mux.HandleFunc("/api/v1/admin/users/", mw.WithCORS(mw.WithJWT(mw.AdminOnly(adminUsersDispatchHandler(s.store)))))
	mux.HandleFunc("/api/v1/admin/login-attempts", mw.WithCORS(mw.WithJWT(mw.AdminOnly(listLoginAttemptsHandler(s.store)))))
	mux.HandleFunc("/api/v1/admin/preauth/keys", mw.WithCORS(mw.WithJWT(mw.AdminOnly(adminPreAuthKeysDispatchHandler))))
	mux.HandleFunc("/api/v1/admin/preauth/keys/", mw.WithCORS(mw.WithJWT(mw.AdminOnly(adminPreAuthKeysDispatchHandler))))
	mux.HandleFunc("/api/v1/admin/preauth/bindings/", mw.WithCORS(mw.WithJWT(mw.AdminOnly(adminDeleteBindingHandler))))

	mux.HandleFunc("/api/v1/preauth/status", mw.WithCORS(preAuthStatusHandler))
	mux.HandleFunc("/api/v1/preauth/bind", mw.WithCORS(preAuthBindHandler))

	mux.HandleFunc("/api/v1/master/config/read", mw.WithCORS(mw.WithJWT(mw.AdminOnly(configReadHandler))))
	mux.HandleFunc("/api/v1/master/config/write", mw.WithCORS(mw.WithJWT(mw.AdminOnly(configWriteHandler))))
	mux.HandleFunc("/api/v1/master/config/public", mw.WithCORS(publicConfigHandler))
	mux.HandleFunc("/api/v1/master/nodes/status", mw.WithCORS(mw.WithJWT(mw.AdminOnly(nodesStatusHandler))))
	mux.HandleFunc("/api/v1/master/stats", mw.WithCORS(mw.WithJWT(mw.AdminOnly(statsHandler))))

	// --- Web panel instance endpoints (JWT or HMAC + instance access) ---
	mux.HandleFunc("/api/v1/master/instances/metadata", mw.WithCORS(mw.WithWebAuth(mw.WithInstanceAccess(metadataDispatchHandler))))
	mux.HandleFunc("/api/v1/master/instances/create", mw.WithCORS(mw.WithWebAuth(createInstanceHandler)))
	mux.HandleFunc("/api/v1/master/instances/delete", mw.WithCORS(mw.WithWebAuth(mw.WithInstanceAccess(deleteInstanceHandler))))
	mux.HandleFunc("/api/v1/master/instances/commit", mw.WithCORS(mw.WithWebAuth(mw.WithInstanceAccess(commitHandler))))
	mux.HandleFunc("/api/v1/master/packet_up", mw.WithCORS(mw.WithWebAuth(mw.WithInstanceAccess(packetUpHandler))))
	mux.HandleFunc("/api/v1/master/push", mw.WithCORS(mw.WithWebAuth(mw.WithInstanceAccess(pushHandler))))

	// --- File endpoints (JWT or HMAC + instance access) ---
	mux.HandleFunc("/api/v1/master/files/list", mw.WithCORS(mw.WithWebAuth(mw.WithInstanceAccess(fileListHandler))))
	mux.HandleFunc("/api/v1/master/files/read", mw.WithCORS(mw.WithWebAuth(mw.WithInstanceAccess(fileReadHandler))))
	mux.HandleFunc("/api/v1/master/files/write", mw.WithCORS(mw.WithWebAuth(mw.WithInstanceAccess(fileWriteHandler))))
	mux.HandleFunc("/api/v1/master/files/upload", mw.WithCORS(mw.WithWebAuth(mw.WithInstanceAccess(fileUploadHandler))))
	mux.HandleFunc("/api/v1/master/files/mkdir", mw.WithCORS(mw.WithWebAuth(mw.WithInstanceAccess(fileMkdirHandler))))
	mux.HandleFunc("/api/v1/master/files/delete", mw.WithCORS(mw.WithWebAuth(mw.WithInstanceAccess(fileDeleteHandler))))
	mux.HandleFunc("/api/v1/master/files/rename", mw.WithCORS(mw.WithWebAuth(mw.WithInstanceAccess(fileRenameHandler))))
	mux.HandleFunc("/api/v1/master/files/upload_pcl_pack", mw.WithCORS(mw.WithWebAuth(mw.WithInstanceAccess(uploadPclHandler))))

	// --- Market endpoints (JWT or HMAC + instance access) ---
	mux.HandleFunc("/api/v1/master/releases/download", mw.WithCORS(mw.WithWebAuth(mw.WithInstanceAccess(releaseDownloadHandler))))
	mux.HandleFunc("/api/v1/master/releases/download-ticket", mw.WithCORS(mw.WithWebAuth(mw.WithInstanceAccess(releaseTicketHandler))))
	mux.HandleFunc("/api/v1/master/releases/download-by-ticket", mw.WithCORS(releaseDownloadByTicketHandler))
	mux.HandleFunc("/api/v1/master/market/download", mw.WithCORS(mw.WithWebAuth(mw.WithInstanceAccess(marketDownloadHandler))))
	mux.HandleFunc("/api/v1/master/market/curse-proxy", mw.WithCORS(mw.WithWebAuth(mw.WithInstanceAccess(curseProxyHandler))))
	mux.HandleFunc("/api/v1/master/market/modrinth-proxy", mw.WithCORS(mw.WithWebAuth(mw.WithInstanceAccess(modrinthProxyHandler))))

	// --- Instance list (both ports: web panel on manage, PCL clients on business) ---
	dataMux.HandleFunc("/api/v1/sync/instances", mw.WithCORS(mw.WithPreAuth(listInstancesHandler)))
	dataMux.HandleFunc("/api/v1/sync/info", mw.WithCORS(mw.WithPreAuth(mw.WithInstanceAccess(infoHandler))))
	// Also expose on manage port for the web frontend (JWT-based)
	mux.HandleFunc("/api/v1/sync/instances", mw.WithCORS(mw.WithOptionalJWT(listInstancesHandler)))
	mux.HandleFunc("/api/v1/sync/info", mw.WithCORS(mw.WithOptionalJWT(infoHandler)))


	// --- Slave endpoints (HMAC internal trust) ---
	mux.HandleFunc("/api/v1/slave/sync", mw.WithCORS(mw.WithAuth(slaveSyncHandler)))
	mux.HandleFunc("/api/v1/slave/status", mw.WithCORS(mw.WithAuth(slaveStatusHandler)))
	mux.HandleFunc("/api/v1/slave/config-sync", mw.WithCORS(mw.WithAuth(slaveConfigSyncHandler)))

	// --- SPA fallback: serve web/ for all unmatched paths ---
	if s.Config().Role == "slave" {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not Found (Slave Node)"))
		})
	} else {
		mux.Handle("/", http.FileServer(http.Dir("./web")))
	}

	adminServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", srv.Config().ManagePort),
		Handler:      middleware.WithByteCounter(&s.responseBytes)(mux),
		ReadTimeout:  15 * time.Minute,
		WriteTimeout: 15 * time.Minute,
		IdleTimeout:  120 * time.Second,
	}

	dataMux.HandleFunc("/api/v1/master/config/public", mw.WithCORS(publicConfigHandler))
	dataMux.HandleFunc("/sync/", mw.WithPreAuth(mw.WithInstanceAccess(syncFileHandler)))
	dataServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", srv.Config().BusinessPort),
		Handler:      dataMux,
		ReadTimeout:  15 * time.Minute,
		WriteTimeout: 15 * time.Minute,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		fmt.Printf("[Control Plane] Listening on :%d\n", srv.Config().ManagePort)
		if err := adminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Control plane error: %v\n", err)
		}
	}()

	go func() {
		fmt.Printf("[Data Plane] Listening on :%d\n", srv.Config().BusinessPort)
		if err := dataServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Data plane error: %v\n", err)
		}
	}()

	fmt.Println("========================================")
	fmt.Println("  CloudAbroad Master Node (v6-security)")
	fmt.Printf("  Health: localhost:%d/health\n", srv.Config().ManagePort)
	fmt.Println("========================================")

	// Start heartbeat loop for Master nodes
	if srv.Config().Role == "master" {
		srv.heartbeatDone = make(chan struct{})
		go heartbeatLoop(srv.heartbeatDone)
		fmt.Println("[Heartbeat] Master-Slave heartbeat started (30s interval)")
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\nShutting down servers...")
	if srv.heartbeatDone != nil {
		close(srv.heartbeatDone)
	}
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
