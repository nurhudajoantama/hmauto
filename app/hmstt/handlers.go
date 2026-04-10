package hmstt

import (
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/nurhudajoantama/hmauto/app/server"
	"github.com/nurhudajoantama/hmauto/internal/request"
	"github.com/nurhudajoantama/hmauto/internal/response"
	"github.com/rs/zerolog"
)

type HmsttHandler struct {
	service *HmsttService
}

func entryToResponse(e StateEntry) StateResponse {
	return StateResponse{
		Type:        e.Type,
		Key:         e.K,
		Value:       e.Value,
		Description: e.Description,
		UpdatedAt:   e.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func RegisterHandlers(s *server.Server, svc *HmsttService) {
	h := &HmsttHandler{service: svc}

	v1 := s.GetRouter().PathPrefix("/v1").Subrouter()
	s.ApplyAuthMiddleware(v1)

	v1.HandleFunc("/states", h.listAllStates).Methods("GET")
	v1.HandleFunc("/states", h.createState).Methods("POST")
	v1.HandleFunc("/states/{type}", h.listStatesByType).Methods("GET")
	v1.HandleFunc("/states/{type}/batch", h.getStatesByKeys).Methods("GET")
	v1.HandleFunc("/states/{type}/{key}", h.getState).Methods("GET")
	v1.HandleFunc("/states/{type}/{key}", h.setState).Methods("PUT")
	v1.HandleFunc("/states/{type}/{key}", h.patchState).Methods("PATCH")
}

// listAllStates godoc
//
//	@Summary		List all states
//	@Description	Returns all states across every type
//	@Tags			states
//	@Produce		json
//	@Security		BearerAuth
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
//	@Security		BearerAuth
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
//	@Security		BearerAuth
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

func (h *HmsttHandler) getStatesByKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := zerolog.Ctx(ctx)
	tipe := mux.Vars(r)["type"]
	keys := r.URL.Query()["key"]

	l.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("hmstt_type", tipe)
	})
	l.Info().Msg("Handling getStatesByKeys request")

	if len(keys) == 0 {
		response.ErrorResponse(w, http.StatusBadRequest, "at least one key query parameter is required", nil)
		return
	}

	entries, err := h.service.GetStatesByKeys(ctx, tipe, keys)
	if err != nil {
		l.Error().Err(err).Msg("getStatesByKeys failed")
		response.ErrorResponse(w, http.StatusInternalServerError, "failed to get states", err)
		return
	}

	data := make([]StateResponse, 0, len(entries))
	for _, entry := range entries {
		data = append(data, entryToResponse(entry))
	}

	response.SuccessResponse(w, data)
}

// createState godoc
//
//	@Summary		Create a state entry
//	@Description	Creates a new state for the given type and key. Returns 409 if the key already exists.
//	@Tags			states
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateStateRequest							true	"State to create"
//	@Success		201		{object}	response.JsonResponse{data=StateResponse}	"Created state"
//	@Failure		400		{object}	response.JsonResponse						"Invalid type/key/value/description"
//	@Failure		401		{object}	response.JsonResponse						"Unauthorized"
//	@Failure		409		{object}	response.JsonResponse						"State already exists"
//	@Failure		500		{object}	response.JsonResponse						"Internal error"
//	@Router			/states [post]
func (h *HmsttHandler) createState(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := zerolog.Ctx(ctx)
	l.Info().Msg("Handling createState request")

	var body CreateStateRequest
	if err := request.DecodeAndValidate(r, &body); err != nil {
		l.Error().Err(err).Msg("createState: validation failed")
		response.ErrorResponse(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	l.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("hmstt_type", body.Type).Str("hmstt_key", body.Key)
	})

	if err := h.service.CreateState(ctx, body.Type, body.Key, body.Value, body.Description); err != nil {
		if errors.Is(err, ErrStateAlreadyExists) {
			response.ErrorResponse(w, http.StatusConflict, "state already exists", err)
			return
		}
		l.Error().Err(err).Msg("createState failed")
		response.ErrorResponse(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	entry, err := h.service.GetState(ctx, body.Type, body.Key)
	if err != nil {
		l.Error().Err(err).Msg("createState: get state after create failed")
		response.ErrorResponse(w, http.StatusInternalServerError, "failed to retrieve created state", err)
		return
	}

	response.CreatedResponse(w, entryToResponse(entry))
}

// setState godoc
//
//	@Summary		Set a state value
//	@Description	Updates the value (and optionally description) of an existing state. Fires MQTT event only if value changes.
//	@Tags			states
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
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
	if err := request.DecodeAndValidate(r, &body); err != nil {
		l.Error().Err(err).Msg("setState: validation failed")
		response.ErrorResponse(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	if err := h.service.SetState(ctx, tipe, key, body.Value, body.Description); err != nil {
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

// patchState godoc
//
//	@Summary		Partially update a state entry
//	@Description	Updates value and/or description independently. Fields not provided are left unchanged. MQTT event fired only if value changes.
//	@Tags			states
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			type	path		string									true	"State type"	example(switch)
//	@Param			key		path		string									true	"State key"		example(modem)
//	@Param			body	body		PatchStateRequest						true	"Fields to update"
//	@Success		200		{object}	response.JsonResponse{data=StateResponse}	"Updated state"
//	@Failure		400		{object}	response.JsonResponse						"Invalid input or nothing to update"
//	@Failure		401		{object}	response.JsonResponse						"Unauthorized"
//	@Failure		404		{object}	response.JsonResponse						"State not found"
//	@Failure		500		{object}	response.JsonResponse						"Internal error"
//	@Router			/states/{type}/{key} [patch]
func (h *HmsttHandler) patchState(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := zerolog.Ctx(ctx)
	p := mux.Vars(r)
	tipe := p["type"]
	key := p["key"]

	l.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("hmstt_type", tipe).Str("hmstt_key", key)
	})
	l.Info().Msg("Handling patchState request")

	var body PatchStateRequest
	if err := request.DecodeAndValidate(r, &body); err != nil {
		l.Error().Err(err).Msg("patchState: validation failed")
		response.ErrorResponse(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	if err := h.service.PatchState(ctx, tipe, key, body.Value, body.Description); err != nil {
		if errors.Is(err, ErrStateNotFound) {
			response.ErrorResponse(w, http.StatusNotFound, "state not found", err)
			return
		}
		if errors.Is(err, ErrNothingToUpdate) {
			response.ErrorResponse(w, http.StatusBadRequest, "nothing to update", err)
			return
		}
		l.Error().Err(err).Msg("patchState failed")
		response.ErrorResponse(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	entry, err := h.service.GetState(ctx, tipe, key)
	if err != nil {
		l.Error().Err(err).Msg("patchState: get state after patch failed")
		response.ErrorResponse(w, http.StatusInternalServerError, "failed to retrieve updated state", err)
		return
	}

	response.SuccessResponse(w, entryToResponse(entry))
}
