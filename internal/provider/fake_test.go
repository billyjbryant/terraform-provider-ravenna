package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// fakeRavenna is an in-memory stand-in for the Ravenna API, sufficient for
// resource lifecycle tests. It stores channels keyed by id.
type fakeRavenna struct {
	mu       sync.Mutex
	nextID   int
	channels map[string]map[string]any
	tags     map[string]map[string]any
	statuses map[string]map[string]any
}

func newFakeRavenna(t *testing.T) *httptest.Server {
	t.Helper()

	f := &fakeRavenna{
		channels: map[string]map[string]any{},
		tags:     map[string]map[string]any{},
		statuses: map[string]map[string]any{},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/queues", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			f.createChannel(w, r)
		case http.MethodGet:
			f.listChannels(w)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/queues/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/queues/")
		switch r.Method {
		case http.MethodGet:
			f.getChannel(w, id)
		case http.MethodPut:
			f.updateChannel(w, r, id)
		case http.MethodDelete:
			f.deleteChannel(w, id)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/tags", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			f.createTag(w, r)
		case http.MethodGet:
			f.listTags(w)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/tags/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/tags/")
		switch r.Method {
		case http.MethodPut:
			f.updateTag(w, r, id)
		case http.MethodDelete:
			f.deleteTag(w, id)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/statuses", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			f.createStatus(w, r)
		case http.MethodGet:
			f.listStatuses(w)
		case http.MethodPut:
			f.updateStatus(w, r)
		case http.MethodDelete:
			f.deleteStatus(w, r.URL.Query().Get("id"))
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	srv := httptest.NewServer(authMiddleware(t, mux))
	t.Cleanup(srv.Close)
	return srv
}

// authMiddleware asserts every request carries the API token, so a regression
// that drops the header fails loudly instead of silently passing.
func authMiddleware(t *testing.T, next http.Handler) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-ravenna-api-token") == "" {
			t.Errorf("request to %s %s carried no x-ravenna-api-token header", r.Method, r.URL.Path)
			writeAPIError(w, http.StatusUnauthorized, "unauthorized", "missing token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeAPIError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"code": code, "message": msg},
	})
}

func (f *fakeRavenna) createChannel(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)

	f.mu.Lock()
	defer f.mu.Unlock()

	f.nextID++
	id := fmt.Sprintf("q_%d", f.nextID)

	ch := map[string]any{
		"id":          id,
		"name":        body["name"],
		"emoji":       body["emoji"],
		"prefix":      stringOrEmpty(body["prefix"]),
		"workspaceId": stringOrDefault(body["workspaceId"], "ws_default"),
		"type":        "DEFAULT",
		"system":      false,
	}
	f.channels[id] = ch

	writeJSON(w, ch)
}

func (f *fakeRavenna) getChannel(w http.ResponseWriter, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	ch, ok := f.channels[id]
	if !ok {
		writeAPIError(w, http.StatusNotFound, "not_found", "no such channel")
		return
	}
	writeJSON(w, ch)
}

func (f *fakeRavenna) updateChannel(w http.ResponseWriter, r *http.Request, id string) {
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)

	f.mu.Lock()
	defer f.mu.Unlock()

	ch, ok := f.channels[id]
	if !ok {
		writeAPIError(w, http.StatusNotFound, "not_found", "no such channel")
		return
	}
	for _, k := range []string{"name", "emoji", "prefix", "workspaceId"} {
		if v, present := body[k]; present {
			ch[k] = v
		}
	}
	writeJSON(w, ch)
}

func (f *fakeRavenna) deleteChannel(w http.ResponseWriter, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.channels[id]; !ok {
		writeAPIError(w, http.StatusNotFound, "not_found", "no such channel")
		return
	}
	delete(f.channels, id)
	w.WriteHeader(http.StatusOK)
}

func (f *fakeRavenna) listChannels(w http.ResponseWriter) {
	f.mu.Lock()
	defer f.mu.Unlock()

	items := make([]map[string]any, 0, len(f.channels))
	for _, ch := range f.channels {
		items = append(items, ch)
	}
	writeJSON(w, map[string]any{"items": items, "totalCount": len(items)})
}

func (f *fakeRavenna) createTag(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)

	f.mu.Lock()
	defer f.mu.Unlock()

	f.nextID++
	id := fmt.Sprintf("t_%d", f.nextID)

	tag := map[string]any{
		"id":          id,
		"name":        body["name"],
		"color":       body["color"],
		"description": body["description"],
		"workspaceId": "ws_default",
	}
	f.tags[id] = tag
	writeJSON(w, tag)
}

func (f *fakeRavenna) updateTag(w http.ResponseWriter, r *http.Request, id string) {
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)

	f.mu.Lock()
	defer f.mu.Unlock()

	tag, ok := f.tags[id]
	if !ok {
		writeAPIError(w, http.StatusNotFound, "not_found", "no such tag")
		return
	}
	for _, k := range []string{"name", "color", "description"} {
		if v, present := body[k]; present {
			tag[k] = v
		}
	}
	writeJSON(w, tag)
}

func (f *fakeRavenna) deleteTag(w http.ResponseWriter, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.tags[id]; !ok {
		writeAPIError(w, http.StatusNotFound, "not_found", "no such tag")
		return
	}
	delete(f.tags, id)
	w.WriteHeader(http.StatusOK)
}

func (f *fakeRavenna) listTags(w http.ResponseWriter) {
	f.mu.Lock()
	defer f.mu.Unlock()

	items := make([]map[string]any, 0, len(f.tags))
	for _, tag := range f.tags {
		items = append(items, tag)
	}
	writeJSON(w, map[string]any{"items": items, "totalCount": len(items)})
}

func (f *fakeRavenna) createStatus(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)

	f.mu.Lock()
	defer f.mu.Unlock()

	f.nextID++
	id := fmt.Sprintf("s_%d", f.nextID)

	st := map[string]any{
		"id":            id,
		"label":         body["label"],
		"statusGroupId": body["statusGroupId"],
		"order":         len(f.statuses) + 1,
		"system":        false,
	}
	f.statuses[id] = st
	writeJSON(w, st)
}

func (f *fakeRavenna) updateStatus(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)

	id, _ := body["id"].(string)

	f.mu.Lock()
	defer f.mu.Unlock()

	st, ok := f.statuses[id]
	if !ok {
		writeAPIError(w, http.StatusNotFound, "not_found", "no such status")
		return
	}
	for _, k := range []string{"label", "order"} {
		if v, present := body[k]; present {
			st[k] = v
		}
	}
	writeJSON(w, st)
}

func (f *fakeRavenna) deleteStatus(w http.ResponseWriter, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.statuses[id]; !ok {
		writeAPIError(w, http.StatusNotFound, "not_found", "no such status")
		return
	}
	delete(f.statuses, id)
	w.WriteHeader(http.StatusOK)
}

// listStatuses returns the {statuses, groups} shape. The groups are fixed
// because Ravenna exposes no way to create them.
func (f *fakeRavenna) listStatuses(w http.ResponseWriter) {
	f.mu.Lock()
	defer f.mu.Unlock()

	items := make([]map[string]any, 0, len(f.statuses))
	for _, st := range f.statuses {
		items = append(items, st)
	}
	writeJSON(w, map[string]any{
		"statuses": items,
		"groups": []map[string]any{
			{"id": "sg_open", "label": "Open", "order": 1, "color": "blue", "workspaceId": "ws_default"},
			{"id": "sg_pending", "label": "Pending", "order": 2, "color": "amber", "workspaceId": "ws_default"},
		},
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func stringOrEmpty(v any) string {
	s, _ := v.(string)
	return s
}

func stringOrDefault(v any, def string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return def
}
