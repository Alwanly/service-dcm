package config

import (
	"os"
	"strconv"
	"time"
)

type ControllerConfig struct {
	ServerAddr    string
	DatabasePath  string
	PollInterval  time.Duration
	AdminUsername string
	AdminPassword string
	AgentUsername string
	AgentPassword string
}

type WorkerConfig struct {
	ServerAddr     string
	RequestTimeout time.Duration
}

type AgentConfig struct {
	ControllerURL  string
	WorkerURL      string
	PollInterval   time.Duration
	RequestTimeout time.Duration
	AgentUsername  string
	AgentPassword  string
}

// LoadControllerConfig reads controller config from environment or returns defaults
func LoadControllerConfig() (*ControllerConfig, error) {
	poll := 5 * time.Second
	if v := os.Getenv("POLL_INTERVAL"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			poll = time.Duration(i) * time.Second
		}
	}

	return &ControllerConfig{
		ServerAddr:    envOrDefault("CONTROLLER_ADDR", ":8080"),
		DatabasePath:  envOrDefault("DATABASE_PATH", "./data.db"),
		PollInterval:  poll,
		AdminUsername: envOrDefault("ADMIN_USER", "admin"),
		AdminPassword: envOrDefault("ADMIN_PASSWORD", "password"),
		AgentUsername: envOrDefault("AGENT_USER", "agent"),
		AgentPassword: envOrDefault("AGENT_PASSWORD", "agentpass"),
	}, nil
}

// LoadWorkerConfig reads worker config from environment or returns defaults
func LoadWorkerConfig() (*WorkerConfig, error) {
	reqTimeout := 10 * time.Second
	if v := os.Getenv("REQUEST_TIMEOUT"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			reqTimeout = time.Duration(i) * time.Second
		}
	}

	return &WorkerConfig{
		ServerAddr:     envOrDefault("WORKER_ADDR", ":8082"),
		RequestTimeout: reqTimeout,
	}, nil
}

// LoadAgentConfig reads agent config from environment or returns defaults
func LoadAgentConfig() (*AgentConfig, error) {
	poll := 5 * time.Second
	if v := os.Getenv("POLL_INTERVAL"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			poll = time.Duration(i) * time.Second
		}
	}

	reqTimeout := 10 * time.Second
	if v := os.Getenv("REQUEST_TIMEOUT"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			reqTimeout = time.Duration(i) * time.Second
		}
	}

	return &AgentConfig{
		ControllerURL:  envOrDefault("CONTROLLER_URL", "http://localhost:8080"),
		WorkerURL:      envOrDefault("WORKER_URL", "http://localhost:8082"),
		PollInterval:   poll,
		RequestTimeout: reqTimeout,
		AgentUsername:  envOrDefault("AGENT_USER", "agent"),
		AgentPassword:  envOrDefault("AGENT_PASSWORD", "agentpass"),
	}, nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
