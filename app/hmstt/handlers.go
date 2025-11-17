package hmstt

import (
	"html/template"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/nurhudajoantama/hmauto/app/server"
	"github.com/rs/zerolog"
)

type HmsttHandler struct {
	service   *HmsttService
	templates *template.Template
}

func RegisterHandlers(s *server.Server, svc *HmsttService) {
	templates := template.Must(template.ParseGlob(HTML_TEMPLATE_PATTERN))

	h := &HmsttHandler{
		service:   svc,
		templates: templates,
	}
	srv := s.GetRouter()
	srv.HandleFunc("/", h.handleIndex).Methods("GET")

	hmsttGroup := srv.PathPrefix("/hmstt").Subrouter()
	hmsttGroup.HandleFunc("/", h.handleIndex).Methods("GET")
	hmsttGroup.HandleFunc("/statehtml/{type}/{key}", h.handleGetStateHTML).Methods("GET")
	hmsttGroup.HandleFunc("/setstatehtml/{type}/{key}", h.handleSetStateHTML).Methods("POST")

	hmsttGroup.HandleFunc("/getstatevalue/{type}/{key}", h.handleGetState).Methods("GET")
}

func (h *HmsttHandler) handleGetState(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	p := mux.Vars(r)
	key := p["key"]
	tipe := p["type"]

	l := zerolog.Ctx(ctx)
	l.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("hmstt_type", tipe).Str("hmstt_key", key)
	})
	l.Info().Msg("Handling GetState request")

	result, err := h.service.GetState(ctx, tipe, key)
	if err != nil {
		returnErrorState(w)
		return
	}
	l.Debug().Str("state_value", result).Send()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(result))

	l.Trace().Msgf("GetState request handled successfully")
}

// handleIndex serves the main (and only) HTML page
func (h *HmsttHandler) handleIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := zerolog.Ctx(ctx)
	l.Info().Msg("Handling Hmstt index page request")

	data := map[string]interface{}{
		"states": []hmsttState{},
	}

	states, err := h.service.GetAllStates(ctx)
	if err == nil {
		data["states"] = states
	}
	l.Debug().Interface("data", data).Msg("Rendering index.html")

	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		l.Error().Err(err).Msg("Failed to execute template")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleState is another HTMX endpoint returning an HTML string
func (h *HmsttHandler) handleGetStateHTML(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := zerolog.Ctx(ctx)
	l.Info().Msg("Handling GetStateHTML request")

	p := mux.Vars(r)
	key := p["key"]
	tipe := p["type"]

	l.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("hmstt_type", tipe).Str("hmstt_key", key)
	})

	var state hmsttState
	results, err := h.service.GetStateDetail(ctx, tipe, key)
	if err != nil {
		l.Error().Err(err).Msg("Failed to get state detail")
		state = hmsttState{Value: ERR_STRING}
	} else {
		state = results
		l.Debug().Interface("state", state).Msg("Retrieved state detail successfully")
	}

	h.returnStateHTML(w, state)
}

func (h *HmsttHandler) handleSetStateHTML(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := zerolog.Ctx(ctx)
	l.Info().Msg("Handling SetStateHTML request")

	p := mux.Vars(r)
	key := p["key"]
	tipe := p["type"]

	l.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("hmstt_type", tipe).Str("hmstt_key", key)
	})

	if err := r.ParseForm(); err != nil {
		l.Error().Err(err).Msg("Failed to parse form")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("ERROR PARSE FORM"))
		return
	}

	value := r.FormValue("value")
	if value == "" {
		l.Error().Msg("State value is empty")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("STATE VALUE EMPTY"))
		return
	}

	if err := h.service.SetState(ctx, tipe, key, value); err != nil {
		l.Error().Err(err).Msg("Failed to set state")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("ERROR SET STATE"))
		return
	}

	var state hmsttState
	results, err := h.service.GetStateDetail(ctx, tipe, key)
	if err != nil {
		l.Error().Err(err).Msg("Failed to get state detail after setting state")
		state = hmsttState{Value: ERR_STRING}
	} else {
		state = results
	}

	h.returnStateHTML(w, state)
}

func (h *HmsttHandler) returnStateHTML(w http.ResponseWriter, state hmsttState) {
	var templateData = state

	templateFileName, ok := TYPE_TEMPLATES[state.Type]
	if !ok {
		templateFileName = HTML_TEMPLATE_NOTFOUND_TYPE
	}

	if err := h.templates.ExecuteTemplate(w, templateFileName, templateData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func returnErrorState(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(ERR_STRING))
}
