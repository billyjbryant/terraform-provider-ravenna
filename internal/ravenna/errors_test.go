package ravenna

import (
	"errors"
	"net/http"
	"testing"
)

func TestDecodeError_Envelope(t *testing.T) {
	body := []byte(`{"error":{"code":"queue_not_found","message":"Channel does not exist"}}`)

	err := decodeError(http.StatusNotFound, body)

	if err.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want 404", err.StatusCode)
	}
	if err.Code != "queue_not_found" {
		t.Errorf("Code = %q, want %q", err.Code, "queue_not_found")
	}
	if err.Message != "Channel does not exist" {
		t.Errorf("Message = %q, want %q", err.Message, "Channel does not exist")
	}
	if !err.IsNotFound() {
		t.Error("IsNotFound() = false, want true")
	}
}

func TestDecodeError_NonJSONBody(t *testing.T) {
	err := decodeError(http.StatusBadGateway, []byte("<html>502 Bad Gateway</html>"))

	if err.StatusCode != http.StatusBadGateway {
		t.Errorf("StatusCode = %d, want 502", err.StatusCode)
	}
	if err.Message == "" {
		t.Error("Message is empty; a non-JSON body must still produce a usable message")
	}
	if err.IsNotFound() {
		t.Error("IsNotFound() = true, want false")
	}
}

func TestAPIError_IsErrorsAs(t *testing.T) {
	var err error = decodeError(http.StatusForbidden, []byte(`{"error":{"code":"forbidden","message":"nope"}}`))

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("errors.As failed; APIError must be usable as a target")
	}
	if apiErr.Code != "forbidden" {
		t.Errorf("Code = %q, want %q", apiErr.Code, "forbidden")
	}
}
