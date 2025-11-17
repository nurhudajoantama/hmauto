package hmalert

import (
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
	hmalertGroup.HandleFunc("/publish", h.PublishAlert).Methods("GET")

}

func (h *HmalertHandler) PublishAlert(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := zerolog.Ctx(ctx)
	l.Info().Msg("Handling PublishAlert request")

	q := r.URL.Query()
	level := q.Get("level")
	message := q.Get("message")
	tipe := q.Get("tipe")

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
