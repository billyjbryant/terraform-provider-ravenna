package ravenna

import (
	"context"
	"net/http"
	"net/url"
)

// TagColors is the closed colour palette the Ravenna API accepts for tags.
var TagColors = []string{
	"tomato", "red", "ruby", "crimson", "pink", "plum", "purple", "violet",
	"iris", "indigo", "blue", "cyan", "teal", "jade", "green", "grass",
	"brown", "orange", "sky", "mint", "lime", "yellow", "amber", "gold",
	"bronze", "gray",
}

// Tag labels tickets.
type Tag struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Color       string  `json:"color"`
	WorkspaceID string  `json:"workspaceId"`
}

// TagCreateRequest is the body of POST /tags.
type TagCreateRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Color       string  `json:"color"`
}

// TagUpdateRequest is the body of PUT /tags/{id}. Description omits
// omitempty deliberately: a nil value must marshal as "description": null
// so clearing it reaches the server, rather than being silently dropped.
type TagUpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description"`
	Color       *string `json:"color,omitempty"`
}

// CreateTag creates a tag.
func (c *Client) CreateTag(ctx context.Context, req TagCreateRequest) (*Tag, error) {
	var out Tag
	if err := c.do(ctx, http.MethodPost, "/tags", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetTag returns a tag by id. The API exposes no GET /tags/{id}, so this lists
// and filters, synthesising a 404 when the id is absent.
func (c *Client) GetTag(ctx context.Context, id, workspaceID string) (*Tag, error) {
	tags, err := c.ListTags(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	for i := range tags {
		if tags[i].ID == id {
			return &tags[i], nil
		}
	}
	return nil, &APIError{
		StatusCode: http.StatusNotFound,
		Code:       "not_found",
		Message:    "tag " + id + " was not found",
	}
}

// UpdateTag updates a tag.
func (c *Client) UpdateTag(ctx context.Context, id string, req TagUpdateRequest) (*Tag, error) {
	var out Tag
	if err := c.do(ctx, http.MethodPut, "/tags/"+url.PathEscape(id), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteTag deletes a tag.
func (c *Client) DeleteTag(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/tags/"+url.PathEscape(id), nil, nil, nil)
}

// ListTags returns every tag, optionally scoped to a workspace.
func (c *Client) ListTags(ctx context.Context, workspaceID string) ([]Tag, error) {
	q := url.Values{}
	if workspaceID != "" {
		q.Set("workspaceId", workspaceID)
	}
	return listAll[Tag](ctx, c, "/tags", q)
}
