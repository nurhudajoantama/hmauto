package hmstt

import (
	"html/template"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/nurhudajoantama/stthmauto/app/server"
)

type hmsttHandler struct {
	service   *hmsttService
	templates *template.Template
}

func RegisterHandlers(s *server.Server, svc *hmsttService) {
	templates := template.Must(template.ParseGlob(HTML_TEMPLATE_PATTERN))

	h := &hmsttHandler{
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

func (h *hmsttHandler) handleGetState(w http.ResponseWriter, r *http.Request) {
	p := mux.Vars(r)
	key := p["key"]
	tipe := p["type"]

	result, err := h.service.GetState(r.Context(), tipe, key)
	if err != nil {
		returnErrorState(w)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(result))
}

// handleIndex serves the main (and only) HTML page
func (h *hmsttHandler) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Provide a Servers slice to the template so index.html can render switch placeholders dynamically.
	data := map[string]interface{}{
		"states": []hmsttState{},
	}

	states, err := h.service.GetAllStates(r.Context())
	if err == nil {
		data["states"] = states
	}

	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleState is another HTMX endpoint returning an HTML string
func (h *hmsttHandler) handleGetStateHTML(w http.ResponseWriter, r *http.Request) {
	p := mux.Vars(r)
	key := p["key"]
	tipe := p["type"]

	var state hmsttState
	results, err := h.service.GetStateDetail(r.Context(), tipe, key)
	if err != nil {
		state = hmsttState{Value: ERR_STRING}
	} else {
		state = results
	}

	h.returnStateHTML(w, state)
}

func (h *hmsttHandler) handleSetStateHTML(w http.ResponseWriter, r *http.Request) {
	p := mux.Vars(r)
	key := p["key"]
	tipe := p["type"]

	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("ERROR PARSE FORM"))
		return
	}

	value := r.FormValue("value")
	if value == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("STATE VALUE EMPTY"))
		return
	}

	if err := h.service.SetState(r.Context(), tipe, key, value); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("ERROR SET STATE"))
		return
	}

	var state hmsttState
	results, err := h.service.GetStateDetail(r.Context(), tipe, key)
	if err != nil {
		state = hmsttState{Value: ERR_STRING}
	} else {
		state = results
	}

	h.returnStateHTML(w, state)
}

func (h *hmsttHandler) returnStateHTML(w http.ResponseWriter, state hmsttState) {
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
