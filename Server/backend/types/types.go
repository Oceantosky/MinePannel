package types

import "time"

// NodeConfig represents a configured node in the cluster.
type NodeConfig struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	IP          string `json:"ip"`
	Flag        string `json:"flag"`
}

// Config is the server configuration.
type Config struct {
	NodeName       string       `json:"node_name"`
	Role           string       `json:"role"`
	SecretKey      string       `json:"secret_key"`
	PublicIP       string       `json:"public_ip"`
	DistPolicy     string       `json:"dist_policy"`
	JWTSecret      string       `json:"jwt_secret"`
	PreAuthSecret  string       `json:"pre_auth_secret"`
	PreAuthEnabled bool         `json:"pre_auth_enabled"`
	TrustedProxy   bool         `json:"trusted_proxy"`
	ManagePort     int          `json:"manage_port"`
	BusinessPort   int          `json:"business_port"`
	Nodes          []NodeConfig `json:"nodes"`
}

// ReleaseEntry represents a versioned release of an instance.
type ReleaseEntry struct {
	VersionID string    `json:"version_id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	FileName  string    `json:"file_name"`
	Size      int64     `json:"size"`
	Time      time.Time `json:"time"`
}

// InstanceMetadata holds mutable metadata for an instance.
type InstanceMetadata struct {
	DisplayName   string         `json:"display_name"`
	CDNLink       string         `json:"cdn_link"`
	IsPaused      bool           `json:"is_paused"`
	ActiveVersion string         `json:"active_version"`
	McVersion     string         `json:"mc_version"`
	ModLoader     string         `json:"mod_loader"`
	Releases      []ReleaseEntry `json:"releases"`
}

// Instance is the public-facing instance representation.
type Instance struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"display_name"`
	Status      string   `json:"status"`
	FullPackURL string   `json:"full_pack_url"`
	McVersion   string   `json:"mc_version"`
	Modloader   string   `json:"modloader"`
	VersionID   string   `json:"version_id"`
	ModList     []string `json:"mod_list"`
}
