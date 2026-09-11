package ravenna

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

func TestClient_Do_SetsAuthHeaderAndDecodes(t *testing.T) {
	var gotToken, gotPath, gotQuery, gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("x-ravenna-api-token")
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"q_1","name":"IT"}`))
	}))
	defer srv.Close()

	c, err := New(srv.URL, "test-token")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var out struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	q := url.Values{"workspaceId": []string{"ws_1"}}
	if err := c.do(context.Background(), http.MethodGet, "/queues/q_1", q, nil, &out); err != nil {
		t.Fatalf("do: %v", err)
	}

	if gotToken != "test-token" {
		t.Errorf("auth header = %q, want %q", gotToken, "test-token")
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/queues/q_1" {
		t.Errorf("path = %q, want /queues/q_1", gotPath)
	}
	if gotQuery != "workspaceId=ws_1" {
		t.Errorf("query = %q, want workspaceId=ws_1", gotQuery)
	}
	if out.Name != "IT" {
		t.Errorf("decoded name = %q, want IT", out.Name)
	}
}

func TestClient_Do_SendsJSONBody(t *testing.T) {
	var got map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "test-token")
	in := map[string]string{"name": "IT", "emoji": "🎧"}
	if err := c.do(context.Background(), http.MethodPost, "/queues", nil, in, nil); err != nil {
		t.Fatalf("do: %v", err)
	}

	if got["name"] != "IT" {
		t.Errorf("body name = %v, want IT", got["name"])
	}
}

func TestClient_Do_ReturnsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"gone"}}`))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "test-token")
	err := c.do(context.Background(), http.MethodGet, "/queues/nope", nil, nil, nil)

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if !apiErr.IsNotFound() {
		t.Errorf("IsNotFound() = false, want true")
	}
}

func TestClient_Do_RetriesOn429ThenSucceeds(t *testing.T) {
	var calls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"code":"rate_limited","message":"slow down"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"q_1"}`))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "test-token", WithMaxRetries(3))

	var out struct {
		ID string `json:"id"`
	}
	if err := c.do(context.Background(), http.MethodGet, "/queues/q_1", nil, nil, &out); err != nil {
		t.Fatalf("do: %v", err)
	}

	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("calls = %d, want 2 (one 429 then one success)", got)
	}
	if out.ID != "q_1" {
		t.Errorf("id = %q, want q_1", out.ID)
	}
}

func TestClient_Do_DoesNotRetry4xx(t *testing.T) {
	var calls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"bad","message":"nope"}}`))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "test-token", WithMaxRetries(3))
	_ = c.do(context.Background(), http.MethodGet, "/queues", nil, nil, nil)

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("calls = %d, want 1 (4xx other than 429 must not retry)", got)
	}
}

func TestClient_Do_HonoursContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "test-token")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	if err := c.do(ctx, http.MethodGet, "/queues", nil, nil, nil); err == nil {
		t.Fatal("do returned nil, want a context deadline error")
	}
}

func TestNew_RejectsEmptyToken(t *testing.T) {
	if _, err := New("https://example.com", ""); err == nil {
		t.Fatal("New with empty token returned nil error, want failure")
	}
}
