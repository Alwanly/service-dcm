package handler

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type RegistrationStatus string

const (
	StatusRegistering        RegistrationStatus = "registering"
	StatusRegistered         RegistrationStatus = "registered"
	StatusRegistrationFailed RegistrationStatus = "registration_failed"
)

type HealthStatus struct {
	mu sync.RWMutex

	Status               RegistrationStatus `json:"status"`
	AgentID              string             `json:"agent_id,omitempty"`
	Hostname             string             `json:"hostname,omitempty"`
	Version              string             `json:"version,omitempty"`
	StartTime            time.Time          `json:"start_time"`
	RegistrationTime     *time.Time         `json:"registration_time,omitempty"`
	Uptime               string             `json:"uptime"`
	RegistrationError    string             `json:"registration_error,omitempty"`
	RegistrationAttempts int                `json:"registration_attempts"`
}

type Handler struct {
	health *HealthStatus
}

func NewHandler(hostname, version string, startTime time.Time) *Handler {
	return &Handler{
		health: &HealthStatus{
			Status:               StatusRegistering,
			Hostname:             hostname,
			Version:              version,
			StartTime:            startTime,
			RegistrationAttempts: 0,
		},
	}
}

func (h *Handler) SetRegistered(agentID string) {
	h.health.mu.Lock()
	defer h.health.mu.Unlock()

	now := time.Now()
	h.health.Status = StatusRegistered
	h.health.AgentID = agentID
	h.health.RegistrationTime = &now
	h.health.RegistrationError = ""
}

func (h *Handler) SetRegistrationFailed(err error, attempts int) {
	h.health.mu.Lock()
	defer h.health.mu.Unlock()

	h.health.Status = StatusRegistrationFailed
	if err != nil {
		h.health.RegistrationError = err.Error()
	}
	h.health.RegistrationAttempts = attempts
}

func (h *Handler) IncrementAttempts() {
	h.health.mu.Lock()
	defer h.health.mu.Unlock()

	h.health.RegistrationAttempts++
}

func (h *Handler) Health(c *fiber.Ctx) error {
	h.health.mu.RLock()
	defer h.health.mu.RUnlock()

	response := *h.health
	response.Uptime = time.Since(h.health.StartTime).String()

	statusCode := fiber.StatusOK
	if response.Status == StatusRegistrationFailed {
		statusCode = fiber.StatusServiceUnavailable
	} else if response.Status == StatusRegistering {
		statusCode = fiber.StatusAccepted
	}

	return c.Status(statusCode).JSON(response)
}
