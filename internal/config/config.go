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
	Redis         *RedisConfig
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
	AgentAddr      string
	Redis          *RedisConfig
	// Registration retry configuration
	RegistrationMaxRetries        int
	RegistrationInitialBackoff    time.Duration
	RegistrationMaxBackoff        time.Duration
	RegistrationBackoffMultiplier float64
	// Hostname used for registration
	Hostname string
}

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// LoadControllerConfig reads controller config from environment or returns defaults
func LoadControllerConfig() (*ControllerConfig, error) {
	poll := 5 * time.Second
	if v := os.Getenv("POLL_INTERVAL"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			poll = time.Duration(i) * time.Second
		}
	}

	cfg := &ControllerConfig{
		ServerAddr:    envOrDefault("CONTROLLER_ADDR", ":8080"),
		DatabasePath:  envOrDefault("DATABASE_PATH", "./data/data.db"),
		PollInterval:  poll,
		AdminUsername: envOrDefault("ADMIN_USER", "admin"),
		AdminPassword: envOrDefault("ADMIN_PASSWORD", "password"),
		AgentUsername: envOrDefault("AGENT_USER", "agent"),
		AgentPassword: envOrDefault("AGENT_PASSWORD", "agentpass"),
	}

	cfg.Redis = LoadRedisConfig()

	return cfg, nil
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

	maxRetries := 5
	if v := os.Getenv("REGISTRATION_MAX_RETRIES"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			maxRetries = i
		}
	}

	initialBackoff := 1 * time.Second
	if v := os.Getenv("REGISTRATION_INITIAL_BACKOFF"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			initialBackoff = time.Duration(i) * time.Second
		}
	}

	maxBackoff := 30 * time.Second
	if v := os.Getenv("REGISTRATION_MAX_BACKOFF"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			maxBackoff = time.Duration(i) * time.Second
		}
	}

	multiplier := 2.0
	if v := os.Getenv("REGISTRATION_BACKOFF_MULTIPLIER"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			multiplier = f
		}
	}

	cfg := &AgentConfig{
		AgentAddr:                     envOrDefault("AGENT_ADDR", ":8081"),
		ControllerURL:                 envOrDefault("CONTROLLER_URL", "http://localhost:8080"),
		WorkerURL:                     envOrDefault("WORKER_URL", "http://localhost:8082"),
		PollInterval:                  poll,
		RequestTimeout:                reqTimeout,
		AgentUsername:                 envOrDefault("AGENT_USER", "agent"),
		AgentPassword:                 envOrDefault("AGENT_PASSWORD", "agentpass"),
		RegistrationMaxRetries:        maxRetries,
		RegistrationInitialBackoff:    initialBackoff,
		RegistrationMaxBackoff:        maxBackoff,
		RegistrationBackoffMultiplier: multiplier,
		Hostname:                      os.Getenv("AGENT_HOSTNAME"),
	}

	cfg.Redis = LoadRedisConfig()

	if cfg.Hostname == "" {
		if hn, err := os.Hostname(); err == nil {
			cfg.Hostname = hn
		} else {
			cfg.Hostname = "agent-hostname"
		}
	}

	return cfg, nil
}

// LoadRedisConfig loads Redis configuration from environment variables
func LoadRedisConfig() *RedisConfig {
	port := 6379
	if v := os.Getenv("REDIS_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			port = p
		}
	}

	db := 0
	if v := os.Getenv("REDIS_DB"); v != "" {
		if d, err := strconv.Atoi(v); err == nil {
			db = d
		}
	}

	return &RedisConfig{
		Host:     envOrDefault("REDIS_HOST", "localhost"),
		Port:     port,
		Password: envOrDefault("REDIS_PASSWORD", ""),
		DB:       db,
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
