package issue

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Meta struct {
	Number           int       `json:"number"`
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	State            string    `json:"state"`
	URL              string    `json:"url"`
	IsPullRequest    bool      `json:"is_pull_request"`
	UpdatedAt        time.Time `json:"updated_at"`
	LastSyncedAt     time.Time `json:"last_synced_at"`
	IssueMDSHA256    string    `json:"issue_md_sha256"`
	RemoteBodySHA256 string    `json:"remote_body_sha256"`
}

func LoadMeta(path string) (Meta, error) {
	var m Meta
	data, err := os.ReadFile(path)
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return m, err
	}
	return m, nil
}

func SaveMeta(path string, m Meta) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func SHA256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
