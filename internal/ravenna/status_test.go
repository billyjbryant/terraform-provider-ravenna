package ravenna

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

const statusListBody = `{
	"statuses":[
		{"id":"s_1","label":"Open","order":1,"system":true,"statusGroupId":"sg_open"},
		{"id":"s_2","label":"Waiting on vendor","order":2,"system":false,"statusGroupId":"sg_pending"}
	],
	"groups":[
		{"id":"sg_open","label":"Open","order":1,"color":"blue","workspaceId":"ws_1"},
		{"id":"sg_pending","label":"Pending","order":2,"color":"amber","workspaceId":"ws_1"}
	]
}`

func TestListStatuses_ReturnsStatusesAndGroups(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(statusListBody))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "test-token")

	statuses, groups, err := c.ListStatuses(context.Background(), "ws_1")
	if err != nil {
		t.Fatalf("ListStatuses: %v", err)
	}
	if len(statuses) != 2 {
		t.Errorf("len(statuses) = %d, want 2", len(statuses))
	}
	if len(groups) != 2 {
		t.Errorf("len(groups) = %d, want 2", len(groups))
	}
	if groups[1].Color != "amber" {
		t.Errorf("groups[1].Color = %q, want amber", groups[1].Color)
	}
}

func TestGetStatus_FiltersFromList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(statusListBody))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "test-token")

	st, err := c.GetStatus(context.Background(), "s_2", "ws_1")
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if st.Label != "Waiting on vendor" {
		t.Errorf("Label = %q, want \"Waiting on vendor\"", st.Label)
	}

	if _, err := c.GetStatus(context.Background(), "s_absent", "ws_1"); err != nil {
		var apiErr *APIError
		if !errors.As(err, &apiErr) || !apiErr.IsNotFound() {
			t.Errorf("err = %v, want a not-found *APIError", err)
		}
	} else {
		t.Error("GetStatus for an absent id returned nil error")
	}
}

func TestUpdateStatus_PutsToCollectionWithIDInBody(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"id":"s_2","label":"Renamed","order":2,"system":false,"statusGroupId":"sg_pending"}`))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "test-token")
	label := "Renamed"

	st, err := c.UpdateStatus(context.Background(), StatusUpdateRequest{ID: "s_2", Label: &label})
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	if gotMethod != http.MethodPut || gotPath != "/statuses" {
		t.Errorf("got %s %s, want PUT /statuses (the id goes in the body, not the path)", gotMethod, gotPath)
	}
	if gotBody["id"] != "s_2" {
		t.Errorf("body id = %v, want s_2", gotBody["id"])
	}
	if st.Label != "Renamed" {
		t.Errorf("Label = %q, want Renamed", st.Label)
	}
}

func TestDeleteStatus_UsesIDQueryParam(t *testing.T) {
	var gotPath, gotQuery, gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery, gotMethod = r.URL.Path, r.URL.RawQuery, r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "test-token")
	if err := c.DeleteStatus(context.Background(), "s_2", ""); err != nil {
		t.Fatalf("DeleteStatus: %v", err)
	}

	if gotMethod != http.MethodDelete || gotPath != "/statuses" {
		t.Errorf("got %s %s, want DELETE /statuses", gotMethod, gotPath)
	}
	if gotQuery != "id=s_2" {
		t.Errorf("query = %q, want id=s_2", gotQuery)
	}
}

func TestDeleteStatus_ForwardsTargetStatusID(t *testing.T) {
	var gotQuery url.Values

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "test-token")
	if err := c.DeleteStatus(context.Background(), "s_2", "s_1"); err != nil {
		t.Fatalf("DeleteStatus: %v", err)
	}

	if got := gotQuery.Get("id"); got != "s_2" {
		t.Errorf("id = %q, want s_2", got)
	}
	if got := gotQuery.Get("targetStatusId"); got != "s_1" {
		t.Errorf("targetStatusId = %q, want s_1", got)
	}
}
