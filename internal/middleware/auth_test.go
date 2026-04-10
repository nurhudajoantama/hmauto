package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBearerTokenAuth(t *testing.T) {
	h := BearerTokenAuth("http-token")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/states", nil)
	req.Header.Set("Authorization", "Bearer http-token")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
}

func TestQueryTokenAuth(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		authorization string
		wantStatus    int
	}{
		{name: "missing token", url: "/mcp", wantStatus: http.StatusUnauthorized},
		{name: "wrong token", url: "/mcp?token=wrong", wantStatus: http.StatusUnauthorized},
		{name: "correct token", url: "/mcp?token=mcp-token", wantStatus: http.StatusNoContent},
		{name: "header only is rejected", url: "/mcp", authorization: "Bearer http-token", wantStatus: http.StatusUnauthorized},
	}

	h := QueryTokenAuth("mcp-token")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.url, nil)
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}
