package ravenna

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type testItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func TestListAll_FollowsCursor(t *testing.T) {
	var seenCursors []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cursor := r.URL.Query().Get("cursor")
		seenCursors = append(seenCursors, cursor)

		switch cursor {
		case "":
			_, _ = w.Write([]byte(`{"items":[{"id":"a","name":"A"}],"totalCount":3,"nextCursor":"c1"}`))
		case "c1":
			_, _ = w.Write([]byte(`{"items":[{"id":"b","name":"B"}],"totalCount":3,"nextCursor":"c2"}`))
		case "c2":
			_, _ = w.Write([]byte(`{"items":[{"id":"c","name":"C"}],"totalCount":3,"nextCursor":null}`))
		default:
			t.Errorf("unexpected cursor %q", cursor)
		}
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "test-token")

	items, err := listAll[testItem](context.Background(), c, "/tags", nil)
	if err != nil {
		t.Fatalf("listAll: %v", err)
	}

	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}
	want := []string{"a", "b", "c"}
	for i, id := range want {
		if items[i].ID != id {
			t.Errorf("items[%d].ID = %q, want %q", i, items[i].ID, id)
		}
	}
	if len(seenCursors) != 3 {
		t.Errorf("made %d requests, want 3", len(seenCursors))
	}
}

func TestListAll_SinglePageNoCursor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"items":[{"id":"only","name":"Only"}],"totalCount":1}`))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "test-token")

	items, err := listAll[testItem](context.Background(), c, "/tags", nil)
	if err != nil {
		t.Fatalf("listAll: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
}

func TestListAll_PreservesCallerQuery(t *testing.T) {
	var gotWorkspace string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotWorkspace = r.URL.Query().Get("workspaceId")
		_, _ = w.Write([]byte(`{"items":[],"totalCount":0}`))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "test-token")

	if _, err := listAll[testItem](context.Background(), c, "/tags", url.Values{"workspaceId": []string{"ws_9"}}); err != nil {
		t.Fatalf("listAll: %v", err)
	}
	if gotWorkspace != "ws_9" {
		t.Errorf("workspaceId = %q, want ws_9", gotWorkspace)
	}
}

func TestListAll_StopsOnRepeatedCursor(t *testing.T) {
	var calls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		// A server that always returns the same cursor would loop forever.
		_, _ = fmt.Fprintf(w, `{"items":[{"id":"x%d"}],"totalCount":99,"nextCursor":"stuck"}`, calls)
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "test-token")

	if _, err := listAll[testItem](context.Background(), c, "/tags", nil); err == nil {
		t.Fatal("listAll returned nil error on a non-advancing cursor, want failure")
	}
}
