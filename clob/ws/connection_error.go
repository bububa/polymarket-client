package ws

import (
	"errors"
	"fmt"
	"strings"

	"github.com/coder/websocket"
)

// ConnectionError reports a WebSocket transport failure or peer-initiated
// close. Status is -1 when the failure did not carry a close status.
type ConnectionError struct {
	// Operation identifies the failed transport phase.
	Operation string
	// Status is the peer close status, or -1 when no close frame was present.
	Status websocket.StatusCode
	// Err is the underlying transport error.
	Err error
}

// Error implements error.
func (e *ConnectionError) Error() string {
	if e == nil {
		return ""
	}
	operation := strings.TrimSpace(e.Operation)
	if operation == "" {
		operation = "transport"
	}
	if e.Status != -1 {
		return fmt.Sprintf("polymarket: websocket %s closed with status %d: %v", operation, e.Status, e.Err)
	}
	return fmt.Sprintf("polymarket: websocket %s: %v", operation, e.Err)
}

// Unwrap returns the underlying transport error.
func (e *ConnectionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// IsConnectionError reports whether err was emitted by the WebSocket
// transport lifecycle rather than by event decoding or application handling.
func IsConnectionError(err error) bool {
	var connectionErr *ConnectionError
	return errors.As(err, &connectionErr)
}

func newConnectionError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return &ConnectionError{
		Operation: operation,
		Status:    websocket.CloseStatus(err),
		Err:       err,
	}
}
