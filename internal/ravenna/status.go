package ravenna

import (
	"context"
	"net/http"
	"net/url"
)

// Status is a ticket status.
type Status struct {
	ID            string `json:"id"`
	Label         string `json:"label"`
	Order         int    `json:"order"`
	System        bool   `json:"system"`
	StatusGroupID string `json:"statusGroupId"`
}

// StatusGroup buckets statuses. Ravenna exposes no create, update or delete
// for groups; they are read-only and arrive alongside statuses.
type StatusGroup struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Order       int    `json:"order"`
	Color       string `json:"color"`
	WorkspaceID string `json:"workspaceId"`
}

// StatusCreateRequest is the body of POST /statuses.
type StatusCreateRequest struct {
	Label         string  `json:"label"`
	StatusGroupID string  `json:"statusGroupId"`
	RequestTypeID *string `json:"requestTypeId,omitempty"`
}

// StatusUpdateRequest is the body of PUT /statuses. The id travels in the
// body rather than the path, which is why it is a field here.
type StatusUpdateRequest struct {
	ID    string  `json:"id"`
	Label *string `json:"label,omitempty"`
	Order *int    `json:"order,omitempty"`
}

// statusListResponse is the shape of GET /statuses, which is not paginated
// and returns two arrays rather than the usual items envelope.
type statusListResponse struct {
	Statuses []Status      `json:"statuses"`
	Groups   []StatusGroup `json:"groups"`
}

// CreateStatus creates a ticket status.
func (c *Client) CreateStatus(ctx context.Context, req StatusCreateRequest) (*Status, error) {
	var out Status
	if err := c.do(ctx, http.MethodPost, "/statuses", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListStatuses returns every status and every status group.
func (c *Client) ListStatuses(ctx context.Context, workspaceID string) ([]Status, []StatusGroup, error) {
	q := url.Values{}
	if workspaceID != "" {
		q.Set("workspaceId", workspaceID)
	}

	var out statusListResponse
	if err := c.do(ctx, http.MethodGet, "/statuses", q, nil, &out); err != nil {
		return nil, nil, err
	}
	return out.Statuses, out.Groups, nil
}

// GetStatus returns a status by id, synthesising a 404 when absent.
func (c *Client) GetStatus(ctx context.Context, id, workspaceID string) (*Status, error) {
	statuses, _, err := c.ListStatuses(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	for i := range statuses {
		if statuses[i].ID == id {
			return &statuses[i], nil
		}
	}
	return nil, &APIError{
		StatusCode: http.StatusNotFound,
		Code:       "not_found",
		Message:    "status " + id + " was not found",
	}
}

// UpdateStatus updates a status. The API takes PUT on the collection with the
// id in the body rather than PUT /statuses/{id}.
func (c *Client) UpdateStatus(ctx context.Context, req StatusUpdateRequest) (*Status, error) {
	var out Status
	if err := c.do(ctx, http.MethodPut, "/statuses", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteStatus deletes a status. The API takes the id as a query parameter on
// the collection rather than DELETE /statuses/{id}. A non-empty targetStatusID
// moves the deleted status's tickets onto that status.
func (c *Client) DeleteStatus(ctx context.Context, id, targetStatusID string) error {
	q := url.Values{"id": []string{id}}
	if targetStatusID != "" {
		q.Set("targetStatusId", targetStatusID)
	}
	return c.do(ctx, http.MethodDelete, "/statuses", q, nil, nil)
}
