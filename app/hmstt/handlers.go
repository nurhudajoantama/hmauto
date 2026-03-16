package hmstt

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/nurhudajoantama/hmauto/app/server"
	"github.com/nurhudajoantama/hmauto/internal/response"
	"github.com/rs/zerolog"
)

type HmsttHandler struct {
	service *HmsttService
}

type stateResponse struct {
	Type      string `json:"type"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	UpdatedAt string `json:"updated_at"`
}

func entryToResponse(e StateEntry) stateResponse {
	return stateResponse{
		Type:      e.Type,
		Key:       e.K,
		Value:     e.Value,
		UpdatedAt: e.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func RegisterHandlers(s *server.Server, svc *HmsttService) {
	h := &HmsttHandler{service: svc}

	protected := s.GetRouter().PathPrefix("/hmstt").Subrouter()
	s.ApplyAuthMiddleware(protected)

	protected.HandleFunc("/states", h.listAllStates).Methods("GET")
	protected.HandleFunc("/states/{type}", h.listStatesByType).Methods("GET")
	protected.HandleFunc("/state/{type}/{key}", h.getState).Methods("GET")
	protected.HandleFunc("/state/{type}/{key}", h.setState).Methods("POST")
}

func (h *HmsttHandler) listAllStates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := zerolog.Ctx(ctx)
	l.Info().Msg("Handling listAllStates request")

	entries, err := h.service.GetAllStates(ctx)
	if err != nil {
		l.Error().Err(err).Msg("listAllStates failed")
		response.ErrorResponse(w, http.StatusInternalServerError, "failed to get states", err)
		return
	}

	data := make([]stateResponse, 0, len(entries))
	for _, e := range entries {
		data = append(data, entryToResponse(e))
	}
	response.SuccessResponse(w, data)
}

func (h *HmsttHandler) listStatesByType(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := zerolog.Ctx(ctx)
	p := mux.Vars(r)
	tipe := p["type"]

	l.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("hmstt_type", tipe)
	})
	l.Info().Msg("Handling listStatesByType request")

	entries, err := h.service.GetAllByType(ctx, tipe)
	if err != nil {
		l.Error().Err(err).Msg("listStatesByType failed")
		response.ErrorResponse(w, http.StatusInternalServerError, "failed to get states", err)
		return
	}
	if len(entries) == 0 {
		response.ErrorResponse(w, http.StatusNotFound, "no states found for type", nil)
		return
	}

	data := make([]stateResponse, 0, len(entries))
	for _, e := range entries {
		data = append(data, entryToResponse(e))
	}
	response.SuccessResponse(w, data)
}

func (h *HmsttHandler) getState(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := zerolog.Ctx(ctx)
	p := mux.Vars(r)
	tipe := p["type"]
	key := p["key"]

	l.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("hmstt_type", tipe).Str("hmstt_key", key)
	})
	l.Info().Msg("Handling getState request")

	entry, err := h.service.GetState(ctx, tipe, key)
	if err != nil {
		l.Error().Err(err).Msg("getState failed")
		response.ErrorResponse(w, http.StatusNotFound, "state not found", err)
		return
	}

	response.SuccessResponse(w, entryToResponse(entry))
}

func (h *HmsttHandler) setState(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := zerolog.Ctx(ctx)
	p := mux.Vars(r)
	tipe := p["type"]
	key := p["key"]

	l.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("hmstt_type", tipe).Str("hmstt_key", key)
	})
	l.Info().Msg("Handling setState request")

	var body struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		l.Error().Err(err).Msg("setState: failed to decode body")
		response.ErrorResponse(w, http.StatusBadRequest, "invalid request body", err)
		return
	}
	if body.Value == "" {
		response.ErrorResponse(w, http.StatusBadRequest, "value is required", nil)
		return
	}

	if err := h.service.SetState(ctx, tipe, key, body.Value); err != nil {
		l.Error().Err(err).Msg("setState failed")
		response.ErrorResponse(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	entry, err := h.service.GetState(ctx, tipe, key)
	if err != nil {
		l.Error().Err(err).Msg("setState: get state after set failed")
		response.ErrorResponse(w, http.StatusInternalServerError, "failed to retrieve updated state", err)
		return
	}

	response.SuccessResponse(w, entryToResponse(entry))
}
