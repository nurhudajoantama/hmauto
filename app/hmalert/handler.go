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
	hmalertGroup.HandleFunc("/publishbatch", h.PublishAlertBatch).Methods("POST")
}

func (h *HmalertHandler) PublishAlert(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := zerolog.Ctx(ctx)
	l.Info().Msg("Handling PublishAlert request")

	var publishReq PublishAlertBody

	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query()
		publishReq.Level = q.Get("level")
		publishReq.Message = q.Get("message")
		publishReq.Tipe = q.Get("tipe")
	case http.MethodPost:
		// parse from json
		body := r.Body
		defer body.Close()
		decoder := json.NewDecoder(body)
		err := decoder.Decode(&publishReq)
		if err != nil {
			l.Error().Err(err).Msg("Failed to parse request body")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	default:
		l.Error().Msg("Unsupported HTTP method")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	l.Debug().Str("level", publishReq.Level).Str("message", publishReq.Message).Str("type", publishReq.Tipe).Msg("Parsed PublishAlert request")

	err := h.Service.PublishAlert(ctx, publishReq)
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

func (h *HmalertHandler) PublishAlertBatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := zerolog.Ctx(ctx)
	l.Info().Msg("Handling PublishAlert request")

	var publishReqs []PublishAlertBody

	// parse from json
	body := r.Body
	defer body.Close()
	decoder := json.NewDecoder(body)
	err := decoder.Decode(&publishReqs)
	if err != nil {
		l.Error().Err(err).Msg("Failed to parse request body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// TROBOS MANG
	for _, publishReq := range publishReqs {
		go func(publishReq PublishAlertBody) {
			l.Debug().Str("level", publishReq.Level).Str("message", publishReq.Message).Str("type", publishReq.Tipe).Msg("Parsed PublishAlert request")
			err := h.Service.PublishAlert(ctx, publishReq)
			if err != nil {
				l.Error().Err(err).Msg("Failed to publish alert")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Failed to publish alert"))
				return
			}
		}(publishReq)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Alert published successfully"))

	l.Trace().Msgf("PublishAlert request handled successfully")
}
