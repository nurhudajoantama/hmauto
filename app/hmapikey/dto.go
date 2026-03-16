package hmapikey

import "time"

// CreateKeyRequest is the request body for creating an API key.
type CreateKeyRequest struct {
	Label string `json:"label" validate:"required,max=100" example:"iot-device-1"`
}

// CreateKeyResponse is returned once on key creation. The key is never shown again.
type CreateKeyResponse struct {
	Key       string    `json:"key"        example:"a1b2c3d4e5f6..."`
	Label     string    `json:"label"      example:"iot-device-1"`
	CreatedAt time.Time `json:"created_at"`
}
