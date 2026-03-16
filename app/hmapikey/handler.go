package hmapikey

import (
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/nurhudajoantama/hmauto/app/server"
	"github.com/nurhudajoantama/hmauto/internal/apikey"
	"github.com/nurhudajoantama/hmauto/internal/request"
	"github.com/nurhudajoantama/hmauto/internal/response"
	"github.com/rs/zerolog"
)

type Handler struct {
	svc *Service
}

func RegisterHandlers(s *server.Server, svc *Service) {
	h := &Handler{svc: svc}

	admin := s.GetRouter().PathPrefix("/v1/admin").Subrouter()
	s.ApplyAdminMiddleware(admin)

	admin.HandleFunc("/apikeys", h.listKeys).Methods("GET")
	admin.HandleFunc("/apikeys", h.createKey).Methods("POST")
	admin.HandleFunc("/apikeys/{key}", h.revokeKey).Methods("DELETE")
}

// listKeys godoc
//
//	@Summary		List API keys
//	@Description	Returns metadata for all active API keys. The full key value is never returned.
//	@Tags			admin
//	@Produce		json
//	@Security		AdminKeyAuth
//	@Success		200	{object}	response.JsonResponse{data=[]apikey.KeyMetadata}	"List of API key metadata"
//	@Failure		401	{object}	response.JsonResponse								"Unauthorized"
//	@Failure		500	{object}	response.JsonResponse								"Internal error"
//	@Router			/admin/apikeys [get]
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

// createKey godoc
//
//	@Summary		Create an API key
//	@Description	Creates a new API key. The full key is returned only in this response — store it securely.
//	@Tags			admin
//	@Accept			json
//	@Produce		json
//	@Security		AdminKeyAuth
//	@Param			body	body		CreateKeyRequest									true	"Key label"
//	@Success		201		{object}	response.JsonResponse{data=CreateKeyResponse}		"Created key (shown once)"
//	@Failure		400		{object}	response.JsonResponse								"Validation error"
//	@Failure		401		{object}	response.JsonResponse								"Unauthorized"
//	@Failure		500		{object}	response.JsonResponse								"Internal error"
//	@Router			/admin/apikeys [post]
func (h *Handler) createKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := zerolog.Ctx(ctx)
	l.Info().Msg("Handling createKey request")

	var body CreateKeyRequest
	if err := request.DecodeAndValidate(r, &body); err != nil {
		l.Error().Err(err).Msg("createKey: validation failed")
		response.ErrorResponse(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	key, err := h.svc.CreateKey(ctx, body.Label)
	if err != nil {
		l.Error().Err(err).Msg("createKey failed")
		response.ErrorResponse(w, http.StatusInternalServerError, "failed to create key", err)
		return
	}

	response.CreatedResponse(w, CreateKeyResponse{
		Key:       key,
		Label:     body.Label,
		CreatedAt: time.Now().UTC(),
	})
}

// revokeKey godoc
//
//	@Summary		Revoke an API key
//	@Description	Permanently revokes an API key by its full key value
//	@Tags			admin
//	@Produce		json
//	@Security		AdminKeyAuth
//	@Param			key	path		string					true	"Full API key to revoke"
//	@Success		200	{object}	response.JsonResponse	"Key revoked"
//	@Failure		401	{object}	response.JsonResponse	"Unauthorized"
//	@Failure		404	{object}	response.JsonResponse	"Key not found"
//	@Failure		500	{object}	response.JsonResponse	"Internal error"
//	@Router			/admin/apikeys/{key} [delete]
func (h *Handler) revokeKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := zerolog.Ctx(ctx)
	key := mux.Vars(r)["key"]

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
