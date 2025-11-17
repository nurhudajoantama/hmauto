package hmalert

import (
	"encoding/json"
	"net/http"

	"github.com/nurhudajoantama/hmauto/app/server"
	"github.com/rs/zerolog"
)

type HmalertHandler struct {
	Service *HmalerService
}

func RegisterHandler(s *server.Server, svc *HmalerService) {
	h := &HmalertHandler{
		Service: svc,
	}

	r := s.GetRouter()
	hmalertGroup := r.PathPrefix("/hmalert").Subrouter()
	hmalertGroup.HandleFunc("/publish", h.PublishAlert).Methods("GET", "POST")

}

type publishAlertRequest struct {
	Level   string `json:"level"`
	Message string `json:"message"`
	Tipe    string `json:"tipe"`
}

func (h *HmalertHandler) PublishAlert(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := zerolog.Ctx(ctx)
	l.Info().Msg("Handling PublishAlert request")

	var level, message, tipe string

	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query()
		level = q.Get("level")
		message = q.Get("message")
		tipe = q.Get("tipe")
	case http.MethodPost:
		// parse from json
		body := r.Body
		defer body.Close()
		decoder := json.NewDecoder(body)
		var publishReq publishAlertRequest
		err := decoder.Decode(&publishReq)
		if err != nil {
			l.Error().Err(err).Msg("Failed to parse request body")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		level = publishReq.Level
		message = publishReq.Message
		tipe = publishReq.Tipe
	default:
		l.Error().Msg("Unsupported HTTP method")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	l.Debug().Str("level", level).Str("message", message).Str("type", tipe).Msg("Parsed PublishAlert request")

	err := h.Service.PublishAlert(ctx, tipe, level, message)
	if err != nil {
		l.Error().Err(err).Msg("Failed to publish alert")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to publish alert"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Alert published successfully"))

	l.Trace().Msgf("PublishAlert request handled successfully")
}
