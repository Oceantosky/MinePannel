package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"cloudsync-v6/auth"
	"cloudsync-v6/store"
	"strings"
	"time"
)

// loginProtection config
type loginProtectionConfig struct {
	Enabled                bool `json:"enabled"`
	MaxFailuresPerIP       int  `json:"max_failures_per_ip"`
	MaxFailuresPerUsername int  `json:"max_failures_per_username"`
	LockoutDurationMinutes int  `json:"lockout_duration_minutes"`
}

var loginProtection = loginProtectionConfig{
	Enabled:                true,
	MaxFailuresPerIP:       5,
	MaxFailuresPerUsername: 3,
	LockoutDurationMinutes: 15,
}

// --- Auth Handlers ---

func loginHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		ip := getRealIP(r)

		// Brute-force check
		if loginProtection.Enabled {
			since := time.Now().Add(-time.Duration(loginProtection.LockoutDurationMinutes) * time.Minute)
			failuresByIP, _ := st.CountLoginFailures("ip", ip, since)
			failuresByUser, _ := st.CountLoginFailures("username", req.Username, since)

			if failuresByIP >= loginProtection.MaxFailuresPerIP {
				sendError(w, http.StatusTooManyRequests,
					fmt.Sprintf("Too many login attempts from this IP. Try again in %d minutes.", loginProtection.LockoutDurationMinutes))
				return
			}
			if failuresByUser >= loginProtection.MaxFailuresPerUsername {
				sendError(w, http.StatusTooManyRequests,
					fmt.Sprintf("Too many login attempts for this user. Try again in %d minutes.", loginProtection.LockoutDurationMinutes))
				return
			}
		}

		user, err := st.GetUserByUsername(req.Username)
		if err != nil {
			st.CreateLoginAttempt(ip, req.Username, false)
			sendError(w, http.StatusUnauthorized, "Invalid username or password")
			return
		}

		if !user.IsActive {
			sendError(w, http.StatusUnauthorized, "Account is disabled")
			return
		}

		// Check if password needs setup (empty hash)
		if user.NeedsPasswordSetup() {
			if req.Password == "" {
				// First login with no password set — allow and flag
				token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
				if err != nil {
					sendError(w, http.StatusInternalServerError, "Failed to generate token")
					return
				}
				st.CreateLoginAttempt(ip, req.Username, true)
				st.UpdateLastLogin(user.ID)
				st.CreateLoginLog(user.ID, ip)
				setJWTCookie(w, token)
			sendSuccess(w, map[string]any{
					"token":                token,
					"user":                 user.ToPublic(),
					"needs_password_setup": true,
				})
				return
			}
			// User provided a password on first login — set it
			hash, err := auth.HashPassword(req.Password)
			if err != nil {
				sendError(w, http.StatusInternalServerError, "Failed to hash password")
				return
			}
			st.UpdateUserPassword(user.ID, hash)
			user.PasswordHash = hash
		} else {
			// Normal password verification
			if err := auth.VerifyPassword(user.PasswordHash, req.Password); err != nil {
				st.CreateLoginAttempt(ip, req.Username, false)
				sendError(w, http.StatusUnauthorized, "Invalid username or password")
				return
			}
		}

		token, err := auth.GenerateToken(user.ID, user.Username, user.Role)
		if err != nil {
			sendError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		st.CreateLoginAttempt(ip, req.Username, true)
		st.ClearLoginAttempts(ip, req.Username) // Clear failures on success
		st.UpdateLastLogin(user.ID)
		st.CreateLoginLog(user.ID, ip)

		setJWTCookie(w, token)
		sendSuccess(w, map[string]any{
			"token": token,
			"user":  user.ToPublic(),
		})
	}
}

func authStatusHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
		user, err := st.GetUserByID(userID)
		if err != nil {
			sendError(w, http.StatusUnauthorized, "User not found")
			return
		}
		sendSuccess(w, map[string]any{
			"user": user.ToPublic(),
		})
	}
}

func changePasswordHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		var req struct {
			CurrentPassword string `json:"current_password"`
			NewPassword     string `json:"new_password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		if len(req.NewPassword) < 6 {
			sendError(w, http.StatusBadRequest, "New password must be at least 6 characters long")
			return
		}

		userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
		user, err := st.GetUserByID(userID)
		if err != nil {
			sendError(w, http.StatusUnauthorized, "User not found")
			return
		}

		if !user.NeedsPasswordSetup() {
			if err := auth.VerifyPassword(user.PasswordHash, req.CurrentPassword); err != nil {
				sendError(w, http.StatusBadRequest, "Current password is incorrect")
				return
			}
		}

		hash, err := auth.HashPassword(req.NewPassword)
		if err != nil {
			sendError(w, http.StatusInternalServerError, "Failed to hash password")
			return
		}

		if err := st.UpdateUserPassword(userID, hash); err != nil {
			sendError(w, http.StatusInternalServerError, "Failed to update password")
			return
		}

		sendSuccess(w, "Password updated")
	}
}

// --- Admin User Management Handlers ---

func listUsersHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := st.ListUsers()
		if err != nil {
			sendError(w, http.StatusInternalServerError, "Failed to list users")
			return
		}
		var publics []store.UserPublic
		for _, u := range users {
			publics = append(publics, u.ToPublic())
		}
		sendSuccess(w, publics)
	}
}

func createUserHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		var req struct {
			Username  string   `json:"username"`
			Password  string   `json:"password"`
			Role      string   `json:"role"`
			Instances []string `json:"instances"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		if req.Username == "" {
			sendError(w, http.StatusBadRequest, "Username is required")
			return
		}
		if req.Role != "admin" && req.Role != "user" {
			sendError(w, http.StatusBadRequest, "Role must be 'admin' or 'user'")
			return
		}
		if req.Instances == nil {
			req.Instances = []string{}
		}

		hash := ""
		if req.Password != "" {
			var err error
			hash, err = auth.HashPassword(req.Password)
			if err != nil {
				sendError(w, http.StatusInternalServerError, "Failed to hash password")
				return
			}
		}

		if err := st.CreateUser(req.Username, hash, req.Role, req.Instances); err != nil {
			sendError(w, http.StatusConflict, err.Error())
			return
		}

		sendSuccess(w, "User created")
	}
}

func updateUserHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		// Extract user ID from path: /api/v1/admin/users/{id}
		idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/users/")
		if idStr == "" {
			sendError(w, http.StatusBadRequest, "Missing user ID")
			return
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			sendError(w, http.StatusBadRequest, "Invalid user ID")
			return
		}

		var req struct {
			Username  string   `json:"username"`
			Role      string   `json:"role"`
			IsActive  bool     `json:"is_active"`
			Instances []string `json:"instances"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		if req.Instances == nil {
			req.Instances = []string{}
		}
		if req.Role != "admin" && req.Role != "user" {
			sendError(w, http.StatusBadRequest, "Role must be 'admin' or 'user'")
			return
		}

		if err := st.UpdateUser(id, req.Username, req.Role, req.IsActive, req.Instances); err != nil {
			sendError(w, http.StatusConflict, err.Error())
			return
		}

		sendSuccess(w, "User updated")
	}
}

func deleteUserHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/users/")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			sendError(w, http.StatusBadRequest, "Invalid user ID")
			return
		}

		userID, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
		if id == userID {
			sendError(w, http.StatusBadRequest, "Cannot delete your own account")
			return
		}

		if err := st.DeleteUser(id); err != nil {
			sendError(w, http.StatusConflict, err.Error())
			return
		}

		sendSuccess(w, "User deleted")
	}
}

func resetUserPasswordHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		// Path: /api/v1/admin/users/{id}/password
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/users/")
		path = strings.TrimSuffix(path, "/password")
		id, err := strconv.Atoi(path)
		if err != nil {
			sendError(w, http.StatusBadRequest, "Invalid user ID")
			return
		}

		var req struct {
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		if len(req.Password) < 6 {
			sendError(w, http.StatusBadRequest, "Password must be at least 6 characters")
			return
		}

		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			sendError(w, http.StatusInternalServerError, "Failed to hash password")
			return
		}

		if err := st.UpdateUserPassword(id, hash); err != nil {
			sendError(w, http.StatusInternalServerError, "Failed to update password")
			return
		}

		sendSuccess(w, "Password updated")
	}
}

func listLoginAttemptsHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get recent login attempts for admin monitoring
		attempts, err := st.ListLoginAttempts(50)
		if err != nil {
			sendError(w, http.StatusInternalServerError, "Failed to query")
			return
		}

		sendSuccess(w, attempts)
	}
}

// adminDispatchHandler routes /api/v1/admin/users/* paths
func adminUsersDispatchHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/users")
		path = strings.TrimSuffix(path, "/")

		if path == "" || path == "/" {
			// /api/v1/admin/users
			switch r.Method {
			case http.MethodGet:
				listUsersHandler(st)(w, r)
			case http.MethodPost:
				createUserHandler(st)(w, r)
			default:
				sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
			}
			return
		}

		// /api/v1/admin/users/{id}/password
		if strings.HasSuffix(path, "/password") {
			resetUserPasswordHandler(st)(w, r)
			return
		}

		// /api/v1/admin/users/{id}
		switch r.Method {
		case http.MethodPut:
			updateUserHandler(st)(w, r)
		case http.MethodDelete:
			deleteUserHandler(st)(w, r)
		default:
			sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	}
}

func logoutHandler(_ *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		clearJWTCookie(w)
		sendSuccess(w, "Logged out")
	}
}

// --- IP Helpers ---

func getRealIP(r *http.Request) string {
	if s.Config().TrustedProxy {
		if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
			return cfIP
		}
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return xri
		}
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if len(parts) > 0 {
				return strings.TrimSpace(parts[0])
			}
		}
	}
	host, _, found := strings.Cut(r.RemoteAddr, ":")
	if !found {
		return r.RemoteAddr
	}
	return host
}
