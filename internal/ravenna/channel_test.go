package ravenna

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateChannel(t *testing.T) {
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/queues" {
			t.Errorf("got %s %s, want POST /queues", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{
			"id":"q_1","name":"IT Helpdesk","prefix":"IT","emoji":"🎧",
			"workspaceId":"ws_1","type":"DEFAULT","system":false
		}`))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "test-token")
	prefix := "IT"

	ch, err := c.CreateChannel(context.Background(), ChannelCreateRequest{
		Name:   "IT Helpdesk",
		Prefix: &prefix,
		Emoji:  "🎧",
	})
	if err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}

	if ch.ID != "q_1" {
		t.Errorf("ID = %q, want q_1", ch.ID)
	}
	if ch.Type != "DEFAULT" {
		t.Errorf("Type = %q, want DEFAULT", ch.Type)
	}
	if gotBody["name"] != "IT Helpdesk" {
		t.Errorf("body name = %v, want IT Helpdesk", gotBody["name"])
	}
	if _, present := gotBody["workspaceId"]; present {
		t.Error("workspaceId present in body; a nil pointer must omit the key entirely")
	}
}

func TestGetChannel_NotFoundIsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"no such channel"}}`))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "test-token")
	_, err := c.GetChannel(context.Background(), "q_missing")

	var apiErr *APIError
	if !errors.As(err, &apiErr) || !apiErr.IsNotFound() {
		t.Fatalf("err = %v, want a not-found *APIError", err)
	}
}

func TestUpdateChannel_SendsOnlySetFields(t *testing.T) {
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/queues/q_1" {
			t.Errorf("got %s %s, want PUT /queues/q_1", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"id":"q_1","name":"Renamed","prefix":"IT","emoji":"🎧","workspaceId":"ws_1","type":"DEFAULT","system":false}`))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "test-token")
	name := "Renamed"

	ch, err := c.UpdateChannel(context.Background(), "q_1", ChannelUpdateRequest{Name: &name})
	if err != nil {
		t.Fatalf("UpdateChannel: %v", err)
	}
	if ch.Name != "Renamed" {
		t.Errorf("Name = %q, want Renamed", ch.Name)
	}
	if _, present := gotBody["emoji"]; present {
		t.Error("emoji present in body; unset fields must be omitted")
	}
}

func TestDeleteChannel(t *testing.T) {
	var called bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodDelete || r.URL.Path != "/queues/q_1" {
			t.Errorf("got %s %s, want DELETE /queues/q_1", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "test-token")
	if err := c.DeleteChannel(context.Background(), "q_1"); err != nil {
		t.Fatalf("DeleteChannel: %v", err)
	}
	if !called {
		t.Error("delete endpoint was not called")
	}
}

func TestListChannels_PassesWorkspaceID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("workspaceId"); got != "ws_1" {
			t.Errorf("workspaceId = %q, want ws_1", got)
		}
		_, _ = w.Write([]byte(`{"items":[{"id":"q_1","name":"IT"}],"totalCount":1}`))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "test-token")
	chans, err := c.ListChannels(context.Background(), "ws_1")
	if err != nil {
		t.Fatalf("ListChannels: %v", err)
	}
	if len(chans) != 1 || chans[0].ID != "q_1" {
		t.Errorf("channels = %+v, want one channel q_1", chans)
	}
}
