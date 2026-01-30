package dto

import "time"

// HeartbeatRequest represents agent health heartbeat
type HeartbeatRequest struct {
	ConfigVersion string `json:"config_version" validate:"required"`
	Status        string `json:"status"`
}

// HeartbeatResponse returned to agent after heartbeat
type HeartbeatResponse struct {
	LatestConfigVersion string    `json:"latest_config_version"`
	ReceivedAt          time.Time `json:"received_at"`
}
