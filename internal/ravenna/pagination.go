package ravenna

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// listEnvelope is the shape every paginated Ravenna list endpoint returns.
// Note this differs from the prose API overview, which claims bare arrays.
type listEnvelope[T any] struct {
	Items      []T     `json:"items"`
	TotalCount int     `json:"totalCount"`
	NextCursor *string `json:"nextCursor"`
}

// maxListPages bounds pagination so a server returning a non-advancing cursor
// fails loudly instead of looping forever.
const maxListPages = 1000

// listAll walks every page of a paginated list endpoint and returns all items.
func listAll[T any](ctx context.Context, c *Client, path string, query url.Values) ([]T, error) {
	q := url.Values{}
	for k, v := range query {
		q[k] = v
	}

	var all []T
	var lastCursor string

	for page := 0; page < maxListPages; page++ {
		var env listEnvelope[T]
		if err := c.do(ctx, http.MethodGet, path, q, nil, &env); err != nil {
			return nil, err
		}

		all = append(all, env.Items...)

		if env.NextCursor == nil || *env.NextCursor == "" {
			return all, nil
		}
		if *env.NextCursor == lastCursor {
			return nil, fmt.Errorf("ravenna: pagination of %s did not advance past cursor %q", path, lastCursor)
		}

		lastCursor = *env.NextCursor
		q.Set("cursor", lastCursor)
	}

	return nil, fmt.Errorf("ravenna: pagination of %s exceeded %d pages", path, maxListPages)
}
