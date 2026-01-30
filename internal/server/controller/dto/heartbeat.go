package dto

import "time"

type HeartbeatRequest struct {
	ConfigVersion string `json:"config_version" validate:"required"`
	Status        string `json:"status"`
}

type HeartbeatResponse struct {
	LatestConfigVersion string    `json:"latest_config_version"`
	ReceivedAt          time.Time `json:"received_at"`
}
