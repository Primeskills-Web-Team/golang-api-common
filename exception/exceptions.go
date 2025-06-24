package exception

import "net/http"

// BadRequestException represents a bad request error.
func BadRequest(message string) *HTTPException {
	return NewHTTPException(http.StatusBadRequest, message)
}

// UnauthorizedException represents an unauthorized error.
func Unauthorized(message string) *HTTPException {
	return NewHTTPException(http.StatusUnauthorized, message)
}

// ForbiddenException represents a forbidden error.
func Forbidden(message string) *HTTPException {
	return NewHTTPException(http.StatusForbidden, message)
}

// NotFoundException represents a not found error.
func NotFound(message string) *HTTPException {
	return NewHTTPException(http.StatusNotFound, message)
}

// InternalServerErrorException represents an internal server error.
func InternalServerError(message string) *HTTPException {
	return NewHTTPException(http.StatusInternalServerError, message)
}

// NotImplementedException represents a not implemented error.
func NotImplemented(message string) *HTTPException {
	return NewHTTPException(http.StatusNotImplemented, message)
}

// ServiceUnavailableException represents a service unavailable error.
func ServiceUnavailable(message string) *HTTPException {
	return NewHTTPException(http.StatusServiceUnavailable, message)
}

// GatewayTimeoutException represents a gateway timeout error.
func GatewayTimeout(message string) *HTTPException {
	return NewHTTPException(http.StatusGatewayTimeout, message)
}

// ConflictException represents a conflict error.
func Conflict(message string) *HTTPException {
	return NewHTTPException(http.StatusConflict, message)
}
