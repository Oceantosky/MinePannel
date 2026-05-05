package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"minepannel-v6/auth"
	"minepannel-v6/store"
	"minepannel-v6/types"
)

// Middleware holds dependencies needed by HTTP middleware functions.
type Middleware struct {
	Store  *store.Store
	Config func() types.Config
}

// New creates a Middleware with the given dependencies.
func New(st *store.Store, cfgFn func() types.Config) *Middleware {
	return &Middleware{Store: st, Config: cfgFn}
}

// sendError is a helper to write JSON error responses.
func sendError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"success":false,"message":%q}`, message)
}

// extractJWT extracts a JWT token from Cookie, Authorization header, or ?token= query param.
func extractJWT(r *http.Request) string {
	if cookie, err := r.Cookie("jwt_token"); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	if authHdr := r.Header.Get("Authorization"); strings.HasPrefix(authHdr, "Bearer ") {
		token := strings.TrimPrefix(authHdr, "Bearer ")
		// Only treat as JWT if it looks like one (JWT has 3 dot-separated segments).
		// HMAC signatures are plain hex and must fall through to HMAC auth.
		if strings.Contains(token, ".") {
			return token
		}
		return ""
	}
	return ""
}

// authorizeJWT validates JWT and sets X- headers. Returns true if authenticated.
func (mw *Middleware) authorizeJWT(w http.ResponseWriter, r *http.Request) bool {
	token := extractJWT(r)
	if token == "" || !strings.Contains(token, ".") {
		return false
	}
	claims, err := auth.ValidateToken(token)
	if err != nil {
		return false
	}
	user, err := mw.Store.GetUserByID(claims.UserID)
	if err != nil || !user.IsActive {
		return false
	}
	r.Header.Set("X-User-ID", fmt.Sprintf("%d", claims.UserID))
	r.Header.Set("X-User-Role", claims.Role)
	r.Header.Set("X-Username", claims.Username)
	r.Header.Set("X-Auth-Method", "jwt")
	return true
}

// --- HMAC Auth Middleware ---

// WithAuth guards routes with HMAC signature verification (inter-node trust).
func (mw *Middleware) WithAuth(next http.HandlerFunc) http.HandlerFunc {
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
		if diff > auth.AuthTimeWindow {
			sendError(w, http.StatusUnauthorized, fmt.Sprintf("Request expired (window: %ds)", auth.AuthTimeWindow))
			return
		}
		var bodyHash string
		if r.Body != nil && !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			r.Body = http.MaxBytesReader(w, r.Body, 50*1024*1024)
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
		if !auth.ValidateMAC(clientToken, timestamp, mw.Config().SecretKey, r.Method, r.URL.Path, bodyHash) {
			sendError(w, http.StatusUnauthorized, "Invalid authentication token")
			return
		}
		next(w, r)
	}
}

// --- JWT Web Auth Middleware ---

// WithJWT guards routes with JWT authentication.
func (mw *Middleware) WithJWT(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !mw.authorizeJWT(w, r) {
			sendError(w, http.StatusUnauthorized, "Missing or invalid authentication")
			return
		}
		next(w, r)
	}
}

// WithOptionalJWT tries to parse JWT but doesn't fail if missing.
func (mw *Middleware) WithOptionalJWT(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mw.authorizeJWT(w, r)
		next(w, r)
	}
}

// WithWebAuth tries JWT first, then falls back to HMAC.
func (mw *Middleware) WithWebAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if extractJWT(r) != "" {
			if mw.authorizeJWT(w, r) {
				next(w, r)
				return
			}
			sendError(w, http.StatusUnauthorized, "Invalid or expired JWT token")
			return
		}
		mw.WithAuth(next)(w, r)
	}
}

// WithPreAuth guards routes with the pre-auth device binding system.
func (mw *Middleware) WithPreAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !mw.Config().PreAuthEnabled {
			next(w, r)
			return
		}
		// Try JWT first (for web panel users)
		if mw.authorizeJWT(w, r) {
			next(w, r)
			return
		}
		// Try HMAC auth (for inter-node/internal trust)
		authHeader := r.Header.Get("Authorization")
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
					if diff <= auth.AuthTimeWindow {
						var bodyHash string
						if r.Body != nil && !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
							r.Body = http.MaxBytesReader(w, r.Body, 50*1024*1024)
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
						if auth.ValidateMAC(parts[1], timestamp, mw.Config().SecretKey, r.Method, r.URL.Path, bodyHash) {
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
		binding, err := mw.Store.GetDeviceBindingByToken(bindingToken)
		if err != nil || !binding.IsActive {
			sendError(w, http.StatusUnauthorized, "Invalid or revoked binding token")
			return
		}
		key, err := mw.Store.GetPreAuthKeyByID(binding.KeyID)
		if err != nil || !key.IsActive {
			sendError(w, http.StatusUnauthorized, "Pre-authorization key has been revoked")
			return
		}
		expectedHash := hex.EncodeToString(auth.Sha256Hash(mw.Config().PreAuthSecret + ":" + deviceID))
		if binding.DeviceHash != expectedHash {
			sendError(w, http.StatusUnauthorized, "Device identity mismatch — hardware may have changed")
			return
		}
		go mw.Store.UpdateDeviceBindingLastSeen(binding.ID)
		r.Header.Set("X-Auth-Method", "preauth")
		r.Header.Set("X-Binding-ID", fmt.Sprintf("%d", binding.ID))
		r.Header.Set("X-Key-ID", fmt.Sprintf("%d", key.ID))
		// Inject allowed instances for instance-level access control (empty = all)
		if !key.CanAccessAll() {
			r.Header.Set("X-Allowed-Instances", strings.Join(key.GetAllowedInstances(), ","))
		}
		next(w, r)
	}
}

// AdminOnly requires JWT with admin role.
func (mw *Middleware) AdminOnly(next http.HandlerFunc) http.HandlerFunc {
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

// WithInstanceAccess checks that the authenticated client has permission to access the instance in ?id=
func (mw *Middleware) WithInstanceAccess(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authMethod := r.Header.Get("X-Auth-Method")

		// JWT admin bypass (existing)
		if authMethod == "jwt" && r.Header.Get("X-User-Role") == "admin" {
			next(w, r)
			return
		}

		instanceID := r.URL.Query().Get("id")
		// Also extract instance ID from /sync/{id}/... path for file handler
		if instanceID == "" && strings.HasPrefix(r.URL.Path, "/sync/") {
			parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/sync/"), "/")
			if len(parts) > 0 && parts[0] != "" {
				instanceID = parts[0]
			}
		}
		if instanceID == "" {
			next(w, r)
			return
		}

		// JWT user check (existing)
		if authMethod == "jwt" {
			userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
			if !mw.Store.UserHasInstance(userID, instanceID) {
				sendError(w, http.StatusForbidden, "Access denied to this instance")
				return
			}
			next(w, r)
			return
		}

		// Pre-auth device binding check
		if authMethod == "preauth" {
			allowed := r.Header.Get("X-Allowed-Instances")
			if allowed == "" {
				next(w, r) // empty = all instances
				return
			}
			for _, id := range strings.Split(allowed, ",") {
				if id == instanceID {
					next(w, r)
					return
				}
			}
			sendError(w, http.StatusForbidden, "Access denied to this instance")
			return
		}

		// Not JWT or preauth — no instance access check needed
		next(w, r)
	}
}

// WithCORS sets CORS headers for the frontend.
func (mw *Middleware) WithCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "http://localhost:5173" || origin == "http://127.0.0.1:5173" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if origin != "" {
			u, err := url.Parse(origin)
			if err == nil {
				host := u.Hostname()
				cfg := mw.Config()
				publicHost := cfg.PublicIP
				if strings.Contains(publicHost, ":") {
					publicHost = strings.Split(publicHost, ":")[0]
				}
				allowed := host == publicHost || host == "localhost" || host == "127.0.0.1"
				for _, node := range cfg.Nodes {
					if host == node.IP {
						allowed = true
						break
					}
				}
				if allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				} else {
					w.Header().Set("Access-Control-Allow-Origin", "null")
				}
			}
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Timestamp, X-Binding-Token, X-Device-ID")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}
