package ravenna

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetTag_FiltersFromList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tags" {
			t.Errorf("path = %q, want /tags (there is no GET /tags/{id})", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"items":[
			{"id":"t_1","name":"hardware","color":"blue","workspaceId":"ws_1","description":null},
			{"id":"t_2","name":"software","color":"green","workspaceId":"ws_1","description":"apps"}
		],"totalCount":2}`))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "test-token")

	tag, err := c.GetTag(context.Background(), "t_2", "ws_1")
	if err != nil {
		t.Fatalf("GetTag: %v", err)
	}
	if tag.Name != "software" {
		t.Errorf("Name = %q, want software", tag.Name)
	}
	if tag.Description == nil || *tag.Description != "apps" {
		t.Errorf("Description = %v, want \"apps\"", tag.Description)
	}
}

func TestGetTag_MissingReturnsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"items":[{"id":"t_1","name":"hardware","color":"blue","workspaceId":"ws_1"}],"totalCount":1}`))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "test-token")

	_, err := c.GetTag(context.Background(), "t_absent", "ws_1")

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if !apiErr.IsNotFound() {
		t.Errorf("IsNotFound() = false, want true so callers treat it as a real 404")
	}
}

func TestCreateTag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/tags" {
			t.Errorf("got %s %s, want POST /tags", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"t_1","name":"hardware","color":"blue","workspaceId":"ws_1","description":null}`))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "test-token")

	tag, err := c.CreateTag(context.Background(), TagCreateRequest{Name: "hardware", Color: "blue"})
	if err != nil {
		t.Fatalf("CreateTag: %v", err)
	}
	if tag.ID != "t_1" {
		t.Errorf("ID = %q, want t_1", tag.ID)
	}
}

func TestListTags_PassesWorkspaceID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("workspaceId"); got != "ws_1" {
			t.Errorf("workspaceId = %q, want ws_1", got)
		}
		_, _ = w.Write([]byte(`{"items":[{"id":"t_1","name":"hardware","color":"blue","workspaceId":"ws_1"}],"totalCount":1}`))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "test-token")
	tags, err := c.ListTags(context.Background(), "ws_1")
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if len(tags) != 1 || tags[0].ID != "t_1" {
		t.Errorf("tags = %+v, want one tag t_1", tags)
	}
}

func TestTagColors_CoversDocumentedPalette(t *testing.T) {
	if len(TagColors) != 26 {
		t.Errorf("len(TagColors) = %d, want 26", len(TagColors))
	}
	for _, want := range []string{"tomato", "gray", "iris", "bronze"} {
		var found bool
		for _, c := range TagColors {
			if c == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("TagColors is missing %q", want)
		}
	}
}
