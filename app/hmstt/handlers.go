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

// StateResponse is the JSON representation of a single state entry.
type StateResponse struct {
	Type      string `json:"type"       example:"switch"`
	Key       string `json:"key"        example:"modem"`
	Value     string `json:"value"      example:"on"`
	UpdatedAt string `json:"updated_at" example:"2026-03-16T12:34:56Z"`
}

// SetStateRequest is the request body for setting a state value.
type SetStateRequest struct {
	Value string `json:"value" example:"on"`
}

func entryToResponse(e StateEntry) StateResponse {
	return StateResponse{
		Type:      e.Type,
		Key:       e.K,
		Value:     e.Value,
		UpdatedAt: e.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func RegisterHandlers(s *server.Server, svc *HmsttService) {
	h := &HmsttHandler{service: svc}

	v1 := s.GetRouter().PathPrefix("/v1").Subrouter()
	s.ApplyAuthMiddleware(v1)

	v1.HandleFunc("/states", h.listAllStates).Methods("GET")
	v1.HandleFunc("/states/{type}", h.listStatesByType).Methods("GET")
	v1.HandleFunc("/states/{type}/{key}", h.getState).Methods("GET")
	v1.HandleFunc("/states/{type}/{key}", h.setState).Methods("PUT")
}

// listAllStates godoc
//
//	@Summary		List all states
//	@Description	Returns all states across every type
//	@Tags			states
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	response.JsonResponse{data=[]StateResponse}	"List of states"
//	@Failure		401	{object}	response.JsonResponse						"Unauthorized"
//	@Failure		500	{object}	response.JsonResponse						"Internal error"
//	@Router			/states [get]
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

	data := make([]StateResponse, 0, len(entries))
	for _, e := range entries {
		data = append(data, entryToResponse(e))
	}
	response.SuccessResponse(w, data)
}

// listStatesByType godoc
//
//	@Summary		List states by type
//	@Description	Returns all states for a given type (e.g. switch)
//	@Tags			states
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			type	path		string									true	"State type"	example(switch)
//	@Success		200		{object}	response.JsonResponse{data=[]StateResponse}	"List of states"
//	@Failure		401		{object}	response.JsonResponse						"Unauthorized"
//	@Failure		404		{object}	response.JsonResponse						"No states found for type"
//	@Failure		500		{object}	response.JsonResponse						"Internal error"
//	@Router			/states/{type} [get]
func (h *HmsttHandler) listStatesByType(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := zerolog.Ctx(ctx)
	tipe := mux.Vars(r)["type"]

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

	data := make([]StateResponse, 0, len(entries))
	for _, e := range entries {
		data = append(data, entryToResponse(e))
	}
	response.SuccessResponse(w, data)
}

// getState godoc
//
//	@Summary		Get a single state
//	@Description	Returns the state for a specific type and key
//	@Tags			states
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			type	path		string									true	"State type"	example(switch)
//	@Param			key		path		string									true	"State key"		example(modem)
//	@Success		200		{object}	response.JsonResponse{data=StateResponse}	"State entry"
//	@Failure		401		{object}	response.JsonResponse						"Unauthorized"
//	@Failure		404		{object}	response.JsonResponse						"State not found"
//	@Router			/states/{type}/{key} [get]
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

// setState godoc
//
//	@Summary		Set a state value
//	@Description	Creates or updates the state for a specific type and key. Valid types: switch (values: on, off)
//	@Tags			states
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			type	path		string									true	"State type"	example(switch)
//	@Param			key		path		string									true	"State key"		example(modem)
//	@Param			body	body		SetStateRequest							true	"State value"
//	@Success		200		{object}	response.JsonResponse{data=StateResponse}	"Updated state"
//	@Failure		400		{object}	response.JsonResponse						"Invalid type/key/value"
//	@Failure		401		{object}	response.JsonResponse						"Unauthorized"
//	@Failure		500		{object}	response.JsonResponse						"Internal error"
//	@Router			/states/{type}/{key} [put]
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

	var body SetStateRequest
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
