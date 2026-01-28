package dto

type HitRequest struct{}

type HitResponse struct {
	ETag string      `json:"etag" example:"v1.0.0"`
	URL  string      `json:"url" example:"http://example.com/api"`
	Data interface{} `json:"data"`
}
