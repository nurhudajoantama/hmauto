package hmalert

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/nurhudajoantama/hmauto/app/server"
	"github.com/nurhudajoantama/hmauto/internal/response"
	"github.com/nurhudajoantama/hmauto/internal/validation"
	"github.com/rs/zerolog"
)

const (
	batchProcessingTimeout = 30 * time.Second
	maxBatchSize           = 100
)

type HmalertHandler struct {
	Service *HmalerService
}

func RegisterHandler(s *server.Server, svc *HmalerService) {
	h := &HmalertHandler{
		Service: svc,
	}

	v1 := s.GetRouter().PathPrefix("/v1").Subrouter()
	s.ApplyAuthMiddleware(v1)

	v1.HandleFunc("/alerts", h.publishAlert).Methods("POST")
	v1.HandleFunc("/alerts/batch", h.publishAlertBatch).Methods("POST")
}

// publishAlert godoc
//
//	@Summary		Publish an alert
//	@Description	Publishes a single alert to the alert pipeline (RabbitMQ → Discord)
//	@Tags			alerts
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			body	body		PublishAlertRequest					true	"Alert payload"
//	@Success		200		{object}	response.JsonResponse				"Alert published"
//	@Failure		400		{object}	response.JsonResponse				"Validation error"
//	@Failure		401		{object}	response.JsonResponse				"Unauthorized"
//	@Failure		500		{object}	response.JsonResponse				"Internal error"
//	@Router			/alerts [post]
func (h *HmalertHandler) publishAlert(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := zerolog.Ctx(ctx)
	l.Info().Msg("Handling publishAlert request")

	var req PublishAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		l.Error().Err(err).Msg("publishAlert: failed to decode body")
		response.ErrorResponse(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	req.Level = validation.SanitizeString(req.Level)
	req.Message = validation.SanitizeString(req.Message)
	req.Type = validation.SanitizeString(req.Type)

	allowedLevels := []string{LEVEL_INFO, LEVEL_WARNING, LEVEL_ERROR}
	if err := validation.ValidateAlertLevel(req.Level, allowedLevels); err != nil {
		l.Warn().Err(err).Str("level", req.Level).Msg("publishAlert: invalid level")
		response.ErrorResponse(w, http.StatusBadRequest, "invalid alert level", err)
		return
	}
	if err := validation.ValidateAlertMessage(req.Message); err != nil {
		l.Warn().Err(err).Msg("publishAlert: invalid message")
		response.ErrorResponse(w, http.StatusBadRequest, "invalid alert message", err)
		return
	}
	if err := validation.ValidateAlertType(req.Type); err != nil {
		l.Warn().Err(err).Msg("publishAlert: invalid type")
		response.ErrorResponse(w, http.StatusBadRequest, "invalid alert type", err)
		return
	}

	if err := h.Service.PublishAlert(ctx, req); err != nil {
		l.Error().Err(err).Msg("publishAlert: failed to publish")
		response.ErrorResponse(w, http.StatusInternalServerError, "failed to publish alert", err)
		return
	}

	response.SuccessResponse(w, nil)
}

// publishAlertBatch godoc
//
//	@Summary		Publish alerts in batch
//	@Description	Publishes up to 100 alerts concurrently to the alert pipeline
//	@Tags			alerts
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			body	body		[]PublishAlertRequest				true	"Array of alert payloads (max 100)"
//	@Success		200		{object}	response.JsonResponse				"All alerts published"
//	@Failure		400		{object}	response.JsonResponse				"Validation error or batch too large"
//	@Failure		401		{object}	response.JsonResponse				"Unauthorized"
//	@Failure		500		{object}	response.JsonResponse				"One or more alerts failed"
//	@Router			/alerts/batch [post]
func (h *HmalertHandler) publishAlertBatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := zerolog.Ctx(ctx)
	l.Info().Msg("Handling publishAlertBatch request")

	var reqs []PublishAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&reqs); err != nil {
		l.Error().Err(err).Msg("publishAlertBatch: failed to decode body")
		response.ErrorResponse(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if len(reqs) > maxBatchSize {
		l.Warn().Int("size", len(reqs)).Msg("publishAlertBatch: batch too large")
		response.ErrorResponse(w, http.StatusBadRequest, "batch size exceeds limit of 100", nil)
		return
	}

	allowedLevels := []string{LEVEL_INFO, LEVEL_WARNING, LEVEL_ERROR}
	for i := range reqs {
		reqs[i].Level = validation.SanitizeString(reqs[i].Level)
		reqs[i].Message = validation.SanitizeString(reqs[i].Message)
		reqs[i].Type = validation.SanitizeString(reqs[i].Type)

		if err := validation.ValidateAlertLevel(reqs[i].Level, allowedLevels); err != nil {
			l.Warn().Err(err).Int("index", i).Msg("publishAlertBatch: invalid level")
			response.ErrorResponse(w, http.StatusBadRequest, "invalid alert level in batch", err)
			return
		}
		if err := validation.ValidateAlertMessage(reqs[i].Message); err != nil {
			l.Warn().Err(err).Int("index", i).Msg("publishAlertBatch: invalid message")
			response.ErrorResponse(w, http.StatusBadRequest, "invalid alert message in batch", err)
			return
		}
		if err := validation.ValidateAlertType(reqs[i].Type); err != nil {
			l.Warn().Err(err).Int("index", i).Msg("publishAlertBatch: invalid type")
			response.ErrorResponse(w, http.StatusBadRequest, "invalid alert type in batch", err)
			return
		}
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(reqs))

	bgCtx, cancel := context.WithTimeout(context.Background(), batchProcessingTimeout)
	defer cancel()

	for _, req := range reqs {
		wg.Add(1)
		go func(r PublishAlertRequest) {
			defer wg.Done()
			if err := h.Service.PublishAlert(bgCtx, r); err != nil {
				l.Error().Err(err).Msg("publishAlertBatch: failed to publish one alert")
				errChan <- err
			}
		}(req)
	}

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		l.Error().Int("error_count", len(errs)).Msg("publishAlertBatch: some alerts failed")
		response.ErrorResponse(w, http.StatusInternalServerError, "some alerts failed to publish", nil)
		return
	}

	response.SuccessResponse(w, nil)
}
