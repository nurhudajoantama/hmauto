package hmapikey

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/nurhudajoantama/hmauto/app/server"
	"github.com/nurhudajoantama/hmauto/internal/apikey"
	"github.com/nurhudajoantama/hmauto/internal/response"
	"github.com/rs/zerolog"
)

type Handler struct {
	svc *Service
}

func RegisterHandlers(s *server.Server, svc *Service) {
	h := &Handler{svc: svc}

	admin := s.GetRouter().PathPrefix("/admin").Subrouter()
	s.ApplyAdminMiddleware(admin)

	admin.HandleFunc("/apikeys", h.listKeys).Methods("GET")
	admin.HandleFunc("/apikeys", h.createKey).Methods("POST")
	admin.HandleFunc("/apikeys/{key}", h.revokeKey).Methods("DELETE")
}

func (h *Handler) listKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := zerolog.Ctx(ctx)
	l.Info().Msg("Handling listKeys request")

	keys, err := h.svc.ListKeys(ctx)
	if err != nil {
		l.Error().Err(err).Msg("listKeys failed")
		response.ErrorResponse(w, http.StatusInternalServerError, "failed to list keys", err)
		return
	}

	response.SuccessResponse(w, keys)
}

func (h *Handler) createKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := zerolog.Ctx(ctx)
	l.Info().Msg("Handling createKey request")

	var body struct {
		Label string `json:"label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		l.Error().Err(err).Msg("createKey: failed to decode body")
		response.ErrorResponse(w, http.StatusBadRequest, "invalid request body", err)
		return
	}
	if body.Label == "" {
		response.ErrorResponse(w, http.StatusBadRequest, "label is required", errors.New("label is required"))
		return
	}

	key, err := h.svc.CreateKey(ctx, body.Label)
	if err != nil {
		l.Error().Err(err).Msg("createKey failed")
		response.ErrorResponse(w, http.StatusInternalServerError, "failed to create key", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"key":        key,
			"label":      body.Label,
			"created_at": time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		},
	})
}

func (h *Handler) revokeKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := zerolog.Ctx(ctx)
	p := mux.Vars(r)
	key := p["key"]

	l.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("apikey_hint", key[:min(4, len(key))])
	})
	l.Info().Msg("Handling revokeKey request")

	if err := h.svc.RevokeKey(ctx, key); err != nil {
		if errors.Is(err, apikey.ErrKeyNotFound) {
			response.ErrorResponse(w, http.StatusNotFound, "api key not found", err)
			return
		}
		l.Error().Err(err).Msg("revokeKey failed")
		response.ErrorResponse(w, http.StatusInternalServerError, "failed to revoke key", err)
		return
	}

	response.SuccessResponse(w, nil)
}

