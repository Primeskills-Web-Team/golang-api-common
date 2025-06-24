package exception

import "net/http"

// HTTPException represents an HTTP error with a status code and message.
type HTTPException struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
}

// NewHTTPException creates a new HTTPException with the given status code and message.
func NewHTTPException(statusCode int, message string) *HTTPException {
	if statusCode == 0 {
		statusCode = http.StatusInternalServerError
	}
	if message == "" {
		message = http.StatusText(statusCode)
	}
	return &HTTPException{
		StatusCode: statusCode,
		Message:    message,
	}
}

// Error returns the error message of the HTTPException.
func (e *HTTPException) Error() string {
	return e.Message
}
