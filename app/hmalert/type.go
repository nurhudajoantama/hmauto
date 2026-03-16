package hmalert

type alertEvent struct {
	Type      string `json:"type"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// PublishAlertRequest is the request body for publishing a single alert.
type PublishAlertRequest struct {
	Level   string `json:"level"   example:"info"`
	Message string `json:"message" example:"Internet is down"`
	Type    string `json:"type"    example:"internet"`
}

// PublishAlertBody is kept as an alias for internal compatibility.
type PublishAlertBody = PublishAlertRequest
