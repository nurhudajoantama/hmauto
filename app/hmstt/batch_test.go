package hmstt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nurhudajoantama/hmauto/app/server"
)

type fakeStateStore struct {
	states map[string][]StateEntry
}

func (f *fakeStateStore) GetState(_ context.Context, tipe, k string) (StateEntry, error) {
	for _, entry := range f.states[tipe] {
		if entry.K == k {
			return entry, nil
		}
	}
	return StateEntry{}, ErrStateNotFound
}

func (f *fakeStateStore) SetState(context.Context, string, string, string, string) error {
	return nil
}

func (f *fakeStateStore) GetAllByType(_ context.Context, tipe string) ([]StateEntry, error) {
	entries := f.states[tipe]
	result := make([]StateEntry, len(entries))
	copy(result, entries)
	return result, nil
}

func (f *fakeStateStore) GetAll(_ context.Context) ([]StateEntry, error) {
	var result []StateEntry
	for _, entries := range f.states {
		result = append(result, entries...)
	}
	return result, nil
}

func TestGetStatesByKeysPreservesRequestOrder(t *testing.T) {
	store := &fakeStateStore{states: map[string][]StateEntry{
		"switch": {
			{Type: "switch", K: "server_1", Value: "on", UpdatedAt: time.Now().UTC()},
			{Type: "switch", K: "server_2", Value: "off", UpdatedAt: time.Now().UTC()},
			{Type: "switch", K: "server_3", Value: "on", UpdatedAt: time.Now().UTC()},
		},
	}}

	svc := NewService(store, nil)
	entries, err := svc.GetStatesByKeys(context.Background(), "switch", []string{"server_3", "missing", "server_1"})
	if err != nil {
		t.Fatalf("GetStatesByKeys() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}
	if entries[0].K != "server_3" || entries[1].K != "server_1" {
		t.Fatalf("unexpected key order = [%s, %s]", entries[0].K, entries[1].K)
	}
}

func TestGetStatesByKeysHandler(t *testing.T) {
	store := &fakeStateStore{states: map[string][]StateEntry{
		"switch": {
			{Type: "switch", K: "server_1", Value: "on", Description: "Server 1", UpdatedAt: time.Date(2026, 4, 10, 12, 34, 56, 0, time.UTC)},
			{Type: "switch", K: "server_2", Value: "off", Description: "Server 2", UpdatedAt: time.Date(2026, 4, 10, 12, 35, 56, 0, time.UTC)},
		},
	}}

	svc := NewService(store, nil)
	srv := server.NewWithConfig(":0", &server.ServerConfig{BearerToken: "test-token"})
	RegisterHandlers(srv, svc)

	t.Run("authorized batch sync returns matching states", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/states/switch/batch?key=server_2&key=missing&key=server_1", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		rr := httptest.NewRecorder()

		srv.GetRouter().ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}

		var resp struct {
			Message string          `json:"message"`
			Data    []StateResponse `json:"data"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if len(resp.Data) != 2 {
			t.Fatalf("len(resp.Data) = %d, want 2", len(resp.Data))
		}
		if resp.Data[0].Key != "server_2" || resp.Data[1].Key != "server_1" {
			t.Fatalf("unexpected response order = [%s, %s]", resp.Data[0].Key, resp.Data[1].Key)
		}
	})

	t.Run("missing auth returns unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/states/switch/batch?key=server_1", nil)
		rr := httptest.NewRecorder()

		srv.GetRouter().ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
		}
	})

	t.Run("missing key query returns bad request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/states/switch/batch", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		rr := httptest.NewRecorder()

		srv.GetRouter().ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})
}
