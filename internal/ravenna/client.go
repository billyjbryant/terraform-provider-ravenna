package ravenna

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the public Ravenna API endpoint.
	DefaultBaseURL = "https://core.ravenna.ai/api"

	authHeader        = "x-ravenna-api-token"
	defaultMaxRetries = 4
	defaultTimeout    = 60 * time.Second
	maxBackoff        = 30 * time.Second
)

// Client is a Ravenna REST API client. It is safe for concurrent use.
type Client struct {
	baseURL    *url.URL
	token      string
	httpClient *http.Client
	userAgent  string
	maxRetries int
}

// Option customises a Client.
type Option func(*Client)

// WithHTTPClient replaces the underlying HTTP client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.httpClient = h }
}

// WithMaxRetries caps retry attempts for 429 and 5xx responses.
func WithMaxRetries(n int) Option {
	return func(c *Client) { c.maxRetries = n }
}

// WithUserAgent sets the User-Agent header.
func WithUserAgent(ua string) Option {
	return func(c *Client) { c.userAgent = ua }
}

// New builds a Client. baseURL may be empty, in which case DefaultBaseURL is
// used. token must be non-empty.
func New(baseURL, token string, opts ...Option) (*Client, error) {
	if token == "" {
		return nil, fmt.Errorf("ravenna: API token is required")
	}
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("ravenna: invalid base URL %q: %w", baseURL, err)
	}

	c := &Client{
		baseURL:    u,
		token:      token,
		httpClient: &http.Client{Timeout: defaultTimeout},
		userAgent:  "terraform-provider-ravenna",
		maxRetries: defaultMaxRetries,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.maxRetries < 0 {
		c.maxRetries = 0
	}
	return c, nil
}

// do issues a request and decodes the response. in is marshalled as the JSON
// body when non-nil; out receives the decoded response when non-nil.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, in, out any) error {
	var body []byte
	if in != nil {
		var err error
		body, err = json.Marshal(in)
		if err != nil {
			return fmt.Errorf("ravenna: encoding request body: %w", err)
		}
	}

	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + path
	if len(query) > 0 {
		endpoint.RawQuery = query.Encode()
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			if err := sleepCtx(ctx, backoff(attempt, lastErr)); err != nil {
				return err
			}
		}

		respBody, status, retryAfter, err := c.roundTrip(ctx, method, endpoint.String(), body)
		if err != nil {
			return err
		}

		if status >= 200 && status < 300 {
			if out == nil || len(respBody) == 0 {
				return nil
			}
			if err := json.Unmarshal(respBody, out); err != nil {
				return fmt.Errorf("ravenna: decoding response from %s %s: %w", method, path, err)
			}
			return nil
		}

		apiErr := decodeError(status, respBody)
		apiErr.retryAfter = retryAfter
		lastErr = apiErr

		if !isRetryable(status) {
			return apiErr
		}
	}

	return lastErr
}

func (c *Client) roundTrip(ctx context.Context, method, endpoint string, body []byte) ([]byte, int, time.Duration, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("ravenna: building request: %w", err)
	}
	req.Header.Set(authHeader, c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("ravenna: %s %s: %w", method, endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("ravenna: reading response body: %w", err)
	}

	return respBody, resp.StatusCode, parseRetryAfter(resp.Header.Get("Retry-After")), nil
}

func isRetryable(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}

func parseRetryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	return 0
}

// backoff returns exponential delay with jitter, preferring the server's
// Retry-After hint when it supplied one.
func backoff(attempt int, lastErr error) time.Duration {
	if apiErr, ok := lastErr.(*APIError); ok && apiErr.retryAfter > 0 {
		return apiErr.retryAfter
	}

	d := time.Duration(1<<uint(attempt-1)) * time.Second
	if d > maxBackoff {
		d = maxBackoff
	}
	jitter := time.Duration(rand.Int63n(int64(d/2) + 1))
	return d/2 + jitter
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
