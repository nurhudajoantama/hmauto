package hmstt

// StateResponse is the JSON representation of a single state entry.
type StateResponse struct {
	Type      string `json:"type"       example:"switch"`
	Key       string `json:"key"        example:"modem"`
	Value     string `json:"value"      example:"on"`
	UpdatedAt string `json:"updated_at" example:"2026-03-16T12:34:56Z"`
}

// SetStateRequest is the request body for setting a state value.
type SetStateRequest struct {
	Value string `json:"value" validate:"required" example:"on"`
}

// CreateStateRequest is the request body for creating a new state entry.
type CreateStateRequest struct {
	Type  string `json:"type"  validate:"required" example:"switch"`
	Key   string `json:"key"   validate:"required" example:"modem"`
	Value string `json:"value" validate:"required" example:"on"`
}
