package response

import (
	"encoding/json"
	"net/http"
)

func SuccessResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(JsonResponse{
		Message: "success",
		Data:    data,
	})
}

func CreatedResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(JsonResponse{
		Message: "created",
		Data:    data,
	})
}

func ErrorResponse(w http.ResponseWriter, statusCode int, message string, err error) {
	r := JsonResponse{Message: message}
	if err != nil {
		r.Error = err.Error()
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(r)
}
