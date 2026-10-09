// Package ravenna provides a typed client for the Ravenna REST API.
package ravenna

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// APIError is a non-2xx response from the Ravenna API. Ravenna returns a
// machine-readable Code alongside the human Message; Code is preserved so
// users can search the Ravenna docs for it.
type APIError struct {
	StatusCode int
	Code       string
	Message    string

	// retryAfter carries the server's Retry-After hint to the backoff
	// calculation. Unexported: it is transport detail, not part of the error
	// users see.
	retryAfter time.Duration
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("ravenna: %s (http %d, code %q)", e.Message, e.StatusCode, e.Code)
	}
	return fmt.Sprintf("ravenna: http %d: %s", e.StatusCode, e.Message)
}

// IsNotFound reports whether the resource is absent, which callers translate
// into removing the resource from Terraform state rather than erroring.
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// decodeError builds an APIError from a response body, falling back to the
// raw body when it is not the documented JSON envelope — gateways and proxies
// in front of the API return HTML.
func decodeError(statusCode int, body []byte) *APIError {
	apiErr := &APIError{StatusCode: statusCode}

	var env errorEnvelope
	if err := json.Unmarshal(body, &env); err == nil && env.Error.Message != "" {
		apiErr.Code = env.Error.Code
		apiErr.Message = env.Error.Message
		return apiErr
	}

	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = http.StatusText(statusCode)
	}
	if len(msg) > 512 {
		msg = msg[:512] + "…"
	}
	apiErr.Message = msg
	return apiErr
}
