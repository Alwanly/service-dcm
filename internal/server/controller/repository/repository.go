package repository

import (
	"database/sql"
	"fmt"
	"time"
)

type Repository struct {
	DB *sql.DB
}

type IRepository interface {
	RegisterAgent(agentID string, info string) error
	UpdateConfig(config string) error
	GetConfigETag() (string, error)
	GetConfigIfChanged(currentETag string) (string, string, error)
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) RegisterAgent(agentID string, startup_time time.Time, status string) error {
	_, err := r.DB.Exec("INSERT OR REPLACE INTO agents (agent_id, startup_time,status) VALUES (?, ?,?)", agentID, startup_time, status)
	return err
}

func generateETag(config string) string {
	// Simple ETag generation using length and timestamp
	return fmt.Sprintf("%x-%d", len(config), time.Now().UnixNano())
}

func (r *Repository) UpdateConfig(config string) error {
	etag := generateETag(config)
	_, err := r.DB.Exec("INSERT INTO configurations (etag, config_data, created_at) VALUES (?, ?, ?)", etag, config, time.Now())
	return err
}

func (r *Repository) GetConfigETag() (string, error) {
	var etag string
	err := r.DB.QueryRow("SELECT etag FROM configurations ORDER BY created_at DESC LIMIT 1").Scan(&etag)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return etag, err
}

func (r *Repository) GetConfig(config string) (string, error) {
	var configData string
	err := r.DB.QueryRow("SELECT config_data FROM configurations ORDER BY created_at DESC LIMIT 1").Scan(&configData)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return configData, err
}

func (r *Repository) GetConfigIfChanged(currentETag string) (string, string, error) {
	var etag, configData string
	err := r.DB.QueryRow("SELECT etag, config_data FROM configurations ORDER BY created_at DESC LIMIT 1").Scan(&etag, &configData)
	if err == sql.ErrNoRows {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	if etag == currentETag {
		return "", "", nil
	}
	return configData, etag, nil
}
