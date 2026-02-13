package hmalert

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/nurhudajoantama/hmauto/app/server"
	"github.com/nurhudajoantama/hmauto/internal/validation"
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
	
	// Apply authentication if enabled
	s.ApplyAuthMiddleware(hmalertGroup)
	
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
		publishReq.Level = validation.SanitizeString(q.Get("level"))
		publishReq.Message = validation.SanitizeString(q.Get("message"))
		publishReq.Tipe = validation.SanitizeString(q.Get("tipe"))
	case http.MethodPost:
		// parse from json
		body := r.Body
		defer body.Close()
		decoder := json.NewDecoder(body)
		err := decoder.Decode(&publishReq)
		if err != nil {
			l.Error().Err(err).Msg("Failed to parse request body")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid request body"))
			return
		}
		// Sanitize inputs
		publishReq.Level = validation.SanitizeString(publishReq.Level)
		publishReq.Message = validation.SanitizeString(publishReq.Message)
		publishReq.Tipe = validation.SanitizeString(publishReq.Tipe)
	default:
		l.Error().Msg("Unsupported HTTP method")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Validate inputs
	allowedLevels := []string{LEVEL_INFO, LEVEL_WARNING, LEVEL_ERROR}
	if err := validation.ValidateAlertLevel(publishReq.Level, allowedLevels); err != nil {
		l.Warn().Err(err).Str("level", publishReq.Level).Msg("Invalid alert level")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid alert level"))
		return
	}

	if err := validation.ValidateAlertMessage(publishReq.Message); err != nil {
		l.Warn().Err(err).Msg("Invalid alert message")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid alert message"))
		return
	}

	if err := validation.ValidateAlertType(publishReq.Tipe); err != nil {
		l.Warn().Err(err).Msg("Invalid alert type")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid alert type"))
		return
	}

	l.Debug().Str("level", publishReq.Level).Str("message", publishReq.Message).Str("type", publishReq.Tipe).Msg("Validated PublishAlert request")

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
	l.Info().Msg("Handling PublishAlertBatch request")

	var publishReqs []PublishAlertBody

	// parse from json
	body := r.Body
	defer body.Close()
	decoder := json.NewDecoder(body)
	err := decoder.Decode(&publishReqs)
	if err != nil {
		l.Error().Err(err).Msg("Failed to parse request body")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid request body"))
		return
	}

	// Limit batch size
	const maxBatchSize = 100
	if len(publishReqs) > maxBatchSize {
		l.Warn().Int("size", len(publishReqs)).Msg("Batch size exceeds limit")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Batch size exceeds limit"))
		return
	}

	// Validate all requests first
	allowedLevels := []string{LEVEL_INFO, LEVEL_WARNING, LEVEL_ERROR}
	for i := range publishReqs {
		// Sanitize inputs
		publishReqs[i].Level = validation.SanitizeString(publishReqs[i].Level)
		publishReqs[i].Message = validation.SanitizeString(publishReqs[i].Message)
		publishReqs[i].Tipe = validation.SanitizeString(publishReqs[i].Tipe)

		// Validate
		if err := validation.ValidateAlertLevel(publishReqs[i].Level, allowedLevels); err != nil {
			l.Warn().Err(err).Int("index", i).Msg("Invalid alert level in batch")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid alert level in batch"))
			return
		}
		if err := validation.ValidateAlertMessage(publishReqs[i].Message); err != nil {
			l.Warn().Err(err).Int("index", i).Msg("Invalid alert message in batch")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid alert message in batch"))
			return
		}
		if err := validation.ValidateAlertType(publishReqs[i].Tipe); err != nil {
			l.Warn().Err(err).Int("index", i).Msg("Invalid alert type in batch")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid alert type in batch"))
			return
		}
	}

	// Process batch asynchronously using background context to avoid cancellation
	// when request completes, but wait for all to finish before responding
	var wg sync.WaitGroup
	errChan := make(chan error, len(publishReqs))
	
	// Create a background context with timeout instead of using request context
	bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, publishReq := range publishReqs {
		wg.Add(1)
		go func(req PublishAlertBody) {
			defer wg.Done()
			l.Debug().Str("level", req.Level).Str("message", req.Message).Str("type", req.Tipe).Msg("Processing batch alert")
			if err := h.Service.PublishAlert(bgCtx, req); err != nil {
				l.Error().Err(err).Msg("Failed to publish alert in batch")
				errChan <- err
			}
		}(publishReq)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Check if any errors occurred
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		l.Error().Int("error_count", len(errs)).Msg("Some alerts failed to publish")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Some alerts failed to publish"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("All alerts published successfully"))

	l.Trace().Int("count", len(publishReqs)).Msg("PublishAlertBatch request handled successfully")
}
