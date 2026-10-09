package ravenna

import (
	"context"
	"net/http"
	"net/url"
)

// Channel is a Ravenna channel, called a queue throughout the REST API.
type Channel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Prefix      string `json:"prefix"`
	Emoji       string `json:"emoji"`
	WorkspaceID string `json:"workspaceId"`
	Type        string `json:"type"`
	System      bool   `json:"system"`
}

// ChannelCreateRequest is the body of POST /queues. Pointer fields are
// omitted from the payload when nil.
type ChannelCreateRequest struct {
	Name             string  `json:"name"`
	Prefix           *string `json:"prefix,omitempty"`
	Emoji            string  `json:"emoji"`
	WorkspaceID      *string `json:"workspaceId,omitempty"`
	RequestChannelID *string `json:"requestChannelId,omitempty"`
	TriageChannelID  *string `json:"triageChannelId,omitempty"`
}

// ChannelUpdateRequest is the body of PUT /queues/{id}.
type ChannelUpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	Prefix      *string `json:"prefix,omitempty"`
	Emoji       *string `json:"emoji,omitempty"`
	WorkspaceID *string `json:"workspaceId,omitempty"`
}

// CreateChannel creates a channel.
func (c *Client) CreateChannel(ctx context.Context, req ChannelCreateRequest) (*Channel, error) {
	var out Channel
	if err := c.do(ctx, http.MethodPost, "/queues", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetChannel fetches a channel by id.
func (c *Client) GetChannel(ctx context.Context, id string) (*Channel, error) {
	var out Channel
	if err := c.do(ctx, http.MethodGet, "/queues/"+url.PathEscape(id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateChannel updates a channel in place.
func (c *Client) UpdateChannel(ctx context.Context, id string, req ChannelUpdateRequest) (*Channel, error) {
	var out Channel
	if err := c.do(ctx, http.MethodPut, "/queues/"+url.PathEscape(id), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteChannel deletes a channel.
func (c *Client) DeleteChannel(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/queues/"+url.PathEscape(id), nil, nil, nil)
}

// ListChannels returns every channel, optionally scoped to a workspace.
func (c *Client) ListChannels(ctx context.Context, workspaceID string) ([]Channel, error) {
	q := url.Values{}
	if workspaceID != "" {
		q.Set("workspaceId", workspaceID)
	}
	return listAll[Channel](ctx, c, "/queues", q)
}
