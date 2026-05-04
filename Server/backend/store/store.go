package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Store wraps the SQLite database connection
type Store struct {
	db *sql.DB
}

// UserRow is the database representation of a user
type UserRow struct {
	ID           int        `json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"`
	Role         string     `json:"role"`
	Instances    string     `json:"instances"` // JSON array of instance IDs
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

// UserPublic is the safe representation returned to API clients
type UserPublic struct {
	ID          int        `json:"id"`
	Username    string     `json:"username"`
	Role        string     `json:"role"`
	Instances   []string   `json:"instances"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

// NeedsPasswordSetup returns true if user has no password set
func (u *UserRow) NeedsPasswordSetup() bool {
	return u.PasswordHash == ""
}

// ToPublic converts a UserRow to a UserPublic (with parsed instances)
func (u *UserRow) ToPublic() UserPublic {
	var instances []string
	json.Unmarshal([]byte(u.Instances), &instances)
	if instances == nil {
		instances = []string{}
	}
	return UserPublic{
		ID:          u.ID,
		Username:    u.Username,
		Role:        u.Role,
		Instances:   instances,
		IsActive:    u.IsActive,
		CreatedAt:   u.CreatedAt,
		LastLoginAt: u.LastLoginAt,
	}
}

// HasInstance checks if the user has access to a specific instance
func (u *UserRow) HasInstance(instanceID string) bool {
	if u.Role == "admin" {
		return true
	}
	var instances []string
	json.Unmarshal([]byte(u.Instances), &instances)
	for _, id := range instances {
		if id == instanceID {
			return true
		}
	}
	return false
}

// PreAuthKeyRow is the database representation of a pre-authorization key
type PreAuthKeyRow struct {
	ID           int        `json:"id"`
	KeyCode      string     `json:"key_code"`
	PlayerName   string     `json:"player_name"`
	MaxDevices   int        `json:"max_devices"`
	DevicesBound int        `json:"devices_bound"`
	IsActive     bool       `json:"is_active"`
	CreatedBy    int        `json:"created_by"`
	CreatedAt    time.Time  `json:"created_at"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`
	InstanceIDs  string     `json:"instance_ids"` // NULL or JSON array of instance IDs; empty = all instances
}

// GetAllowedInstances parses InstanceIDs JSON into a string slice.
// Returns nil for unrestricted (all instances) access.
func (k *PreAuthKeyRow) GetAllowedInstances() []string {
	if k.InstanceIDs == "" {
		return nil
	}
	var ids []string
	json.Unmarshal([]byte(k.InstanceIDs), &ids)
	return ids
}

// CanAccessAll returns true if this key has unrestricted access to all instances.
func (k *PreAuthKeyRow) CanAccessAll() bool {
	return k.InstanceIDs == "" || k.InstanceIDs == "[]"
}

// DeviceBindingRow is the database representation of a device binding
type DeviceBindingRow struct {
	ID           int       `json:"id"`
	KeyID        int       `json:"key_id"`
	DeviceHash   string    `json:"-"` // never returned to clients
	BindingToken string    `json:"binding_token"`
	DeviceLabel  string    `json:"device_label"`
	FirstBoundAt time.Time `json:"first_bound_at"`
	LastSeenAt   time.Time `json:"last_seen_at"`
	IsActive     bool      `json:"is_active"`
}

// DeviceBindingPublic is the safe representation returned to API clients
type DeviceBindingPublic struct {
	ID           int       `json:"id"`
	KeyID        int       `json:"key_id"`
	BindingToken string    `json:"binding_token"`
	DeviceLabel  string    `json:"device_label"`
	FirstBoundAt time.Time `json:"first_bound_at"`
	LastSeenAt   time.Time `json:"last_seen_at"`
	IsActive     bool      `json:"is_active"`
}

// LoginAttempt record
type LoginAttempt struct {
	ID        int       `json:"id"`
	IP        string    `json:"ip"`
	Username  string    `json:"username"`
	Success   bool      `json:"success"`
	CreatedAt time.Time `json:"created_at"`
}

// NewStore opens the SQLite database and initializes tables
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite is single-writer

	store := &Store{db: db}
	if err := store.initTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("init tables: %w", err)
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) initTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL DEFAULT '',
			role TEXT NOT NULL CHECK(role IN ('admin','user')),
			instances TEXT NOT NULL DEFAULT '[]',
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_login_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS login_attempts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ip TEXT NOT NULL,
			username TEXT NOT NULL,
			success INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS login_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			ip TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_login_attempts_ip ON login_attempts(ip, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_login_attempts_username ON login_attempts(username, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_login_log_user ON login_log(user_id, created_at)`,
		`CREATE TABLE IF NOT EXISTS pre_auth_keys (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				key_code TEXT NOT NULL UNIQUE,
				player_name TEXT NOT NULL DEFAULT '',
				max_devices INTEGER NOT NULL DEFAULT 3,
				devices_bound INTEGER NOT NULL DEFAULT 0,
				is_active INTEGER NOT NULL DEFAULT 1,
				created_by INTEGER NOT NULL,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				last_used_at DATETIME,
				instance_ids TEXT,
				FOREIGN KEY (created_by) REFERENCES users(id)
			)`,
		`CREATE TABLE IF NOT EXISTS device_bindings (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				key_id INTEGER NOT NULL,
				device_hash TEXT NOT NULL,
				binding_token TEXT NOT NULL UNIQUE,
				device_label TEXT NOT NULL DEFAULT '',
				first_bound_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				last_seen_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				is_active INTEGER NOT NULL DEFAULT 1,
				FOREIGN KEY (key_id) REFERENCES pre_auth_keys(id)
			)`,
		`CREATE INDEX IF NOT EXISTS idx_bindings_token ON device_bindings(binding_token)`,
		`CREATE INDEX IF NOT EXISTS idx_bindings_key ON device_bindings(key_id)`,
	}

	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("sql error: %w", err)
		}
	}

	// Migration: add instance_ids to pre_auth_keys for existing databases
	s.db.Exec("ALTER TABLE pre_auth_keys ADD COLUMN instance_ids TEXT")

	return s.createDefaultAdmin()
}

func (s *Store) createDefaultAdmin() error {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&count)
	if count > 0 {
		return nil
	}
	_, err := s.db.Exec("INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)",
		"admin", "", "admin")
	return err
}

// --- User CRUD ---

func (s *Store) CreateUser(username, passwordHash, role string, instances []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if role == "admin" {
		var count int
		tx.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&count)
		if count > 0 {
			return fmt.Errorf("only one admin account allowed")
		}
	}
	instJSON, _ := json.Marshal(instances)
	_, err = tx.Exec(
		"INSERT INTO users (username, password_hash, role, instances) VALUES (?, ?, ?, ?)",
		username, passwordHash, role, string(instJSON),
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) GetUserByUsername(username string) (*UserRow, error) {
	var u UserRow
	var ll sql.NullTime
	err := s.db.QueryRow(
		"SELECT id, username, password_hash, role, instances, is_active, created_at, last_login_at FROM users WHERE username = ?",
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.Instances, &u.IsActive, &u.CreatedAt, &ll)
	if err != nil {
		return nil, err
	}
	if ll.Valid {
		u.LastLoginAt = &ll.Time
	}
	return &u, nil
}

func (s *Store) GetUserByID(id int) (*UserRow, error) {
	var u UserRow
	var ll sql.NullTime
	err := s.db.QueryRow(
		"SELECT id, username, password_hash, role, instances, is_active, created_at, last_login_at FROM users WHERE id = ?",
		id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.Instances, &u.IsActive, &u.CreatedAt, &ll)
	if err != nil {
		return nil, err
	}
	if ll.Valid {
		u.LastLoginAt = &ll.Time
	}
	return &u, nil
}

func (s *Store) ListUsers() ([]*UserRow, error) {
	rows, err := s.db.Query(
		"SELECT id, username, password_hash, role, instances, is_active, created_at, last_login_at FROM users ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*UserRow
	for rows.Next() {
		var u UserRow
		var ll sql.NullTime
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.Instances, &u.IsActive, &u.CreatedAt, &ll); err != nil {
			return nil, err
		}
		if ll.Valid {
			u.LastLoginAt = &ll.Time
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}

func (s *Store) UpdateUser(id int, username, role string, isActive bool, instances []string) error {
	existing, err := s.GetUserByID(id)
	if err != nil {
		return err
	}
	
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Prevent last admin demotion
	if existing.Role == "admin" && role != "admin" {
		var count int
		tx.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&count)
		if count <= 1 {
			return fmt.Errorf("cannot demote the last admin")
		}
	}
	// Prevent second admin
	if role == "admin" && existing.Role != "admin" {
		var count int
		tx.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&count)
		if count > 0 {
			return fmt.Errorf("only one admin account allowed")
		}
	}
	instJSON, _ := json.Marshal(instances)
	_, err = tx.Exec(
		"UPDATE users SET username = ?, role = ?, is_active = ?, instances = ? WHERE id = ?",
		username, role, isActive, string(instJSON), id,
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) DeleteUser(id int) error {
	user, err := s.GetUserByID(id)
	if err != nil {
		return err
	}
	if user.Role == "admin" {
		var count int
		s.db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&count)
		if count <= 1 {
			return fmt.Errorf("cannot delete the last admin")
		}
	}
	_, err = s.db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

func (s *Store) UpdateUserPassword(id int, passwordHash string) error {
	_, err := s.db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", passwordHash, id)
	return err
}

func (s *Store) UpdateLastLogin(userID int) error {
	_, err := s.db.Exec("UPDATE users SET last_login_at = ? WHERE id = ?", time.Now(), userID)
	return err
}

// --- types.Instance Access ---

func (s *Store) UserHasInstance(userID int, instanceID string) bool {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return false
	}
	return user.HasInstance(instanceID)
}

// --- Login Logs ---

func (s *Store) CreateLoginLog(userID int, ip string) error {
	_, err := s.db.Exec("INSERT INTO login_log (user_id, ip) VALUES (?, ?)", userID, ip)
	return err
}

// --- Login Attempts (brute-force protection) ---

func (s *Store) CreateLoginAttempt(ip, username string, success bool) error {
	successInt := 0
	if success {
		successInt = 1
	}
	_, err := s.db.Exec(
		"INSERT INTO login_attempts (ip, username, success) VALUES (?, ?, ?)",
		ip, username, successInt,
	)
	return err
}

func (s *Store) CountLoginFailures(fieldType, value string, since time.Time) (int, error) {
	var count int
	var err error
	if fieldType == "ip" {
		err = s.db.QueryRow("SELECT COUNT(*) FROM login_attempts WHERE ip = ? AND success = 0 AND created_at > ?", value, since).Scan(&count)
	} else if fieldType == "username" {
		err = s.db.QueryRow("SELECT COUNT(*) FROM login_attempts WHERE username = ? AND success = 0 AND created_at > ?", value, since).Scan(&count)
	} else {
		return 0, fmt.Errorf("invalid field type: %s", fieldType)
	}
	return count, err
}

func (s *Store) ClearLoginAttempts(ip, username string) error {
	_, err := s.db.Exec("DELETE FROM login_attempts WHERE ip = ? AND username = ?", ip, username)
	return err
}

// --- Pre-Auth Keys ---

func (s *Store) CreatePreAuthKey(keyCode, playerName string, maxDevices int, createdBy int, instanceIDs []string) (*PreAuthKeyRow, error) {
	if maxDevices <= 0 {
		maxDevices = 3
	}
	var instJSON string
	if len(instanceIDs) > 0 {
		b, err := json.Marshal(instanceIDs)
		if err != nil {
			return nil, err
		}
		instJSON = string(b)
	}
	res, err := s.db.Exec(
		"INSERT INTO pre_auth_keys (key_code, player_name, max_devices, created_by, instance_ids) VALUES (?, ?, ?, ?, ?)",
		keyCode, playerName, maxDevices, createdBy, instJSON,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.GetPreAuthKeyByID(int(id))
}

func (s *Store) GetPreAuthKeyByID(id int) (*PreAuthKeyRow, error) {
	var k PreAuthKeyRow
	var lu sql.NullTime
	err := s.db.QueryRow(
		"SELECT id, key_code, player_name, max_devices, devices_bound, is_active, created_by, created_at, last_used_at, COALESCE(instance_ids,'') FROM pre_auth_keys WHERE id = ?",
		id,
	).Scan(&k.ID, &k.KeyCode, &k.PlayerName, &k.MaxDevices, &k.DevicesBound, &k.IsActive, &k.CreatedBy, &k.CreatedAt, &lu, &k.InstanceIDs)
	if err != nil {
		return nil, err
	}
	if lu.Valid {
		k.LastUsedAt = &lu.Time
	}
	return &k, nil
}

func (s *Store) GetPreAuthKeyByCode(keyCode string) (*PreAuthKeyRow, error) {
	var k PreAuthKeyRow
	var lu sql.NullTime
	err := s.db.QueryRow(
		"SELECT id, key_code, player_name, max_devices, devices_bound, is_active, created_by, created_at, last_used_at, COALESCE(instance_ids,'') FROM pre_auth_keys WHERE key_code = ?",
		keyCode,
	).Scan(&k.ID, &k.KeyCode, &k.PlayerName, &k.MaxDevices, &k.DevicesBound, &k.IsActive, &k.CreatedBy, &k.CreatedAt, &lu, &k.InstanceIDs)
	if err != nil {
		return nil, err
	}
	if lu.Valid {
		k.LastUsedAt = &lu.Time
	}
	return &k, nil
}

func (s *Store) ListPreAuthKeys() ([]*PreAuthKeyRow, error) {
	rows, err := s.db.Query(
		"SELECT id, key_code, player_name, max_devices, devices_bound, is_active, created_by, created_at, last_used_at, COALESCE(instance_ids,'') FROM pre_auth_keys ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*PreAuthKeyRow
	for rows.Next() {
		var k PreAuthKeyRow
		var lu sql.NullTime
		if err := rows.Scan(&k.ID, &k.KeyCode, &k.PlayerName, &k.MaxDevices, &k.DevicesBound, &k.IsActive, &k.CreatedBy, &k.CreatedAt, &lu, &k.InstanceIDs); err != nil {
			return nil, err
		}
		if lu.Valid {
			k.LastUsedAt = &lu.Time
		}
		keys = append(keys, &k)
	}
	return keys, rows.Err()
}

func (s *Store) UpdatePreAuthKey(id int, playerName string, maxDevices int, isActive bool, instanceIDs []string) error {
	var instJSON interface{}
	if len(instanceIDs) > 0 {
		b, _ := json.Marshal(instanceIDs)
		instJSON = string(b)
	}
	_, err := s.db.Exec(
		"UPDATE pre_auth_keys SET player_name = ?, max_devices = ?, is_active = ?, instance_ids = ? WHERE id = ?",
		playerName, maxDevices, isActive, instJSON, id,
	)
	return err
}

func (s *Store) UpdatePreAuthKeyLastUsed(id int) error {
	_, err := s.db.Exec("UPDATE pre_auth_keys SET last_used_at = ? WHERE id = ?", time.Now(), id)
	return err
}

func (s *Store) IncrementPreAuthKeyDevicesBound(id int) error {
	_, err := s.db.Exec("UPDATE pre_auth_keys SET devices_bound = devices_bound + 1 WHERE id = ?", id)
	return err
}

func (s *Store) DecrementPreAuthKeyDevicesBound(id int) error {
	_, err := s.db.Exec("UPDATE pre_auth_keys SET devices_bound = MAX(0, devices_bound - 1) WHERE id = ?", id)
	return err
}

func (s *Store) DeletePreAuthKey(id int) error {
	// Delete all associated bindings first to satisfy FK constraint
	if _, err := s.db.Exec("DELETE FROM device_bindings WHERE key_id = ?", id); err != nil {
		return err
	}
	_, err := s.db.Exec("DELETE FROM pre_auth_keys WHERE id = ?", id)
	return err
}

// --- Device Bindings ---

func (s *Store) CreateDeviceBinding(keyID int, deviceHash, bindingToken, deviceLabel string) (*DeviceBindingRow, error) {
	now := time.Now()
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Atomic slot check: increment only if devices_bound < max_devices
	result, err := tx.Exec(
		"UPDATE pre_auth_keys SET devices_bound = devices_bound + 1 WHERE id = ? AND devices_bound < max_devices",
		keyID,
	)
	if err != nil {
		return nil, err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, fmt.Errorf("no remaining device slots")
	}

	_, err = tx.Exec(
		"INSERT INTO device_bindings (key_id, device_hash, binding_token, device_label, first_bound_at, last_seen_at) VALUES (?, ?, ?, ?, ?, ?)",
		keyID, deviceHash, bindingToken, deviceLabel, now, now,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	var b DeviceBindingRow
	err = s.db.QueryRow(
		"SELECT id, key_id, device_hash, binding_token, device_label, first_bound_at, last_seen_at, is_active FROM device_bindings WHERE binding_token = ?",
		bindingToken,
	).Scan(&b.ID, &b.KeyID, &b.DeviceHash, &b.BindingToken, &b.DeviceLabel, &b.FirstBoundAt, &b.LastSeenAt, &b.IsActive)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *Store) GetDeviceBindingByID(id int) (*DeviceBindingRow, error) {
	var b DeviceBindingRow
	err := s.db.QueryRow(
		"SELECT id, key_id, device_hash, binding_token, device_label, first_bound_at, last_seen_at, is_active FROM device_bindings WHERE id = ?",
		id,
	).Scan(&b.ID, &b.KeyID, &b.DeviceHash, &b.BindingToken, &b.DeviceLabel, &b.FirstBoundAt, &b.LastSeenAt, &b.IsActive)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *Store) GetDeviceBindingByToken(bindingToken string) (*DeviceBindingRow, error) {
	var b DeviceBindingRow
	err := s.db.QueryRow(
		"SELECT id, key_id, device_hash, binding_token, device_label, first_bound_at, last_seen_at, is_active FROM device_bindings WHERE binding_token = ?",
		bindingToken,
	).Scan(&b.ID, &b.KeyID, &b.DeviceHash, &b.BindingToken, &b.DeviceLabel, &b.FirstBoundAt, &b.LastSeenAt, &b.IsActive)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *Store) ListDeviceBindingsByKey(keyID int) ([]*DeviceBindingRow, error) {
	rows, err := s.db.Query(
		"SELECT id, key_id, device_hash, binding_token, device_label, first_bound_at, last_seen_at, is_active FROM device_bindings WHERE key_id = ? ORDER BY first_bound_at DESC",
		keyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bindings []*DeviceBindingRow
	for rows.Next() {
		var b DeviceBindingRow
		if err := rows.Scan(&b.ID, &b.KeyID, &b.DeviceHash, &b.BindingToken, &b.DeviceLabel, &b.FirstBoundAt, &b.LastSeenAt, &b.IsActive); err != nil {
			return nil, err
		}
		bindings = append(bindings, &b)
	}
	return bindings, rows.Err()
}

func (s *Store) UpdateDeviceBindingLastSeen(id int) error {
	_, err := s.db.Exec("UPDATE device_bindings SET last_seen_at = ? WHERE id = ?", time.Now(), id)
	return err
}

func (s *Store) DeactivateDeviceBinding(id int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var keyID int
	var isActive bool
	err = tx.QueryRow("SELECT key_id, is_active FROM device_bindings WHERE id = ?", id).Scan(&keyID, &isActive)
	if err != nil {
		return err
	}

	if isActive {
		_, err = tx.Exec("UPDATE device_bindings SET is_active = 0 WHERE id = ?", id)
		if err != nil {
			return err
		}
		_, err = tx.Exec("UPDATE pre_auth_keys SET devices_bound = devices_bound - 1 WHERE id = ?", keyID)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ListLoginAttempts(limit int) ([]LoginAttempt, error) {
	rows, err := s.db.Query("SELECT id, ip, username, success, created_at FROM login_attempts ORDER BY created_at DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attempts []LoginAttempt
	for rows.Next() {
		var a LoginAttempt
		var successInt int
		if err := rows.Scan(&a.ID, &a.IP, &a.Username, &successInt, &a.CreatedAt); err != nil {
			continue
		}
		a.Success = successInt == 1
		attempts = append(attempts, a)
	}
	return attempts, nil
}
