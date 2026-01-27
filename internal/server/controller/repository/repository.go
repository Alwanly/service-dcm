package repository

import "database/sql"

type Repository struct {
	DB *sql.DB
}

type IRepository interface {
	RegisterAgent(agentID string, info string) error
	CreateConfig(config string) error
	GetConfigETag() (string, error)
	GetConfigIfChanged(currentETag string) (string, string, error)
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) RegisterAgent(agentID string, info string) error {
	_, err := r.DB.Exec("INSERT OR REPLACE INTO agents (agent_id, info) VALUES (?, ?)", agentID, info)
	return err
}

func (r *Repository) CreateConfig(config string) error {
	_, err := r.DB.Exec("INSERT INTO configurations (config_data) VALUES (?)", config)
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
