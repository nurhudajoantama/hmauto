package hmalert

type alertEvent struct {
	Type      string `json:"type"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

type PublishAlertBody struct {
	Level   string `json:"level"`
	Message string `json:"message"`
	Tipe    string `json:"tipe"`
}
