package pisigo

import (
	"errors"
	"fmt"
	"net/http"
)

type HTTPError struct {
	Code    int            `json:"-"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
	Internal error         `json:"-"`
}

func (e *HTTPError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Internal)
	}
	return e.Message
}

func (e *HTTPError) Unwrap() error {
	return e.Internal
}

func (e *HTTPError) Status() int {
	if e.Code == 0 {
		return http.StatusInternalServerError
	}
	return e.Code
}

func NewHTTPError(code int, message string) *HTTPError {
	return &HTTPError{Code: code, Message: message}
}

func NewHTTPErrorWithInternal(code int, message string, internal error) *HTTPError {
	return &HTTPError{Code: code, Message: message, Internal: internal}
}

func (e *HTTPError) WithDetails(details map[string]any) *HTTPError {
	e.Details = details
	return e
}

func AsHTTPError(err error) (*HTTPError, bool) {
	var he *HTTPError
	if errors.As(err, &he) {
		return he, true
	}
	return nil, false
}

var (
	ErrBadRequest          = NewHTTPError(http.StatusBadRequest, "bad request")
	ErrUnauthorized        = NewHTTPError(http.StatusUnauthorized, "unauthorized")
	ErrForbidden           = NewHTTPError(http.StatusForbidden, "forbidden")
	ErrNotFound            = NewHTTPError(http.StatusNotFound, "not found")
	ErrMethodNotAllowed    = NewHTTPError(http.StatusMethodNotAllowed, "method not allowed")
	ErrConflict            = NewHTTPError(http.StatusConflict, "conflict")
	ErrTooManyRequests     = NewHTTPError(http.StatusTooManyRequests, "too many requests")
	ErrInternalServer      = NewHTTPError(http.StatusInternalServerError, "internal server error")
	ErrServiceUnavailable  = NewHTTPError(http.StatusServiceUnavailable, "service unavailable")
	ErrGatewayTimeout      = NewHTTPError(http.StatusGatewayTimeout, "gateway timeout")
	ErrRequestEntityTooLarge = NewHTTPError(http.StatusRequestEntityTooLarge, "request entity too large")
)

type ErrorHandler func(c *Context, err error)

func DefaultErrorHandler(c *Context, err error) {
	if c.Written() {
		return
	}
	if he, ok := AsHTTPError(err); ok {
		body := map[string]any{
			"message": he.Message,
		}
		if he.Details != nil {
			body["details"] = he.Details
		}
		_ = c.JSON(he.Status(), body)
		return
	}
	c.Log().Error("unhandled error", "error", err)
	_ = c.JSON(http.StatusInternalServerError, map[string]any{
		"message": "internal server error",
	})
}
