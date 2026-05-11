package middleware

import (
	"errors"
	"net/http"

	"xray2wg/backend/internal/domain"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ErrorResponse is the JSON envelope for API errors.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail is returned to clients (no stack traces or internal details on 5xx).
type ErrorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func loggerFor(c echo.Context) zerolog.Logger {
	lg := GetLogger(c)
	if lg == nil || lg.GetLevel() == zerolog.Disabled {
		return log.Logger
	}
	return *lg
}

type mappedError struct {
	status    int
	code      string
	clientMsg string
	logFull   bool
}

func mapError(err error) mappedError {
	var de *domain.Error
	if errors.As(err, &de) && de != nil {
		st, domCode := httpStatusAndCodeForDomain(de.Code)
		if st >= http.StatusInternalServerError {
			return mappedError{status: st, code: "INTERNAL", clientMsg: "internal server error", logFull: true}
		}
		return mappedError{status: st, code: string(domCode), clientMsg: de.Message, logFull: false}
	}

	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		return mappedError{http.StatusUnauthorized, "UNAUTHORIZED", domain.ErrUnauthorized.Error(), false}
	case errors.Is(err, domain.ErrNotFound):
		return mappedError{http.StatusNotFound, "NOT_FOUND", domain.ErrNotFound.Error(), false}
	case errors.Is(err, domain.ErrConflict):
		return mappedError{http.StatusConflict, "CONFLICT", domain.ErrConflict.Error(), false}
	case errors.Is(err, domain.ErrNodeInUse):
		return mappedError{http.StatusConflict, "NODE_IN_USE", domain.ErrNodeInUse.Error(), false}
	case errors.Is(err, domain.ErrValidation):
		return mappedError{http.StatusBadRequest, "VALIDATION", domain.ErrValidation.Error(), false}
	case errors.Is(err, domain.ErrRateLimited):
		return mappedError{http.StatusTooManyRequests, "RATE_LIMIT", domain.ErrRateLimited.Error(), false}
	case errors.Is(err, domain.ErrBootstrapDone):
		return mappedError{http.StatusBadRequest, "BOOTSTRAP_DONE", domain.ErrBootstrapDone.Error(), false}
	case errors.Is(err, domain.ErrInvalidPassword):
		return mappedError{http.StatusUnauthorized, "UNAUTHORIZED", domain.ErrInvalidPassword.Error(), false}
	}

	var he *echo.HTTPError
	if errors.As(err, &he) {
		code := he.Code
		if code >= http.StatusInternalServerError {
			return mappedError{
				status:    http.StatusInternalServerError,
				code:      "INTERNAL",
				clientMsg: "internal server error",
				logFull:   true,
			}
		}
		msg := httpStatusTextSafe(he)
		ec := "HTTP_ERROR"
		if code == http.StatusUnauthorized {
			ec = "UNAUTHORIZED"
		}
		return mappedError{status: code, code: ec, clientMsg: msg, logFull: false}
	}

	return mappedError{
		status:    http.StatusInternalServerError,
		code:      "INTERNAL",
		clientMsg: "internal server error",
		logFull:   true,
	}
}

func httpStatusAndCodeForDomain(code domain.ErrorCode) (int, domain.ErrorCode) {
	switch code {
	case domain.CodeNotFound:
		return http.StatusNotFound, domain.CodeNotFound
	case domain.CodeUnauthorized:
		return http.StatusUnauthorized, domain.CodeUnauthorized
	case domain.CodeForbidden:
		return http.StatusForbidden, domain.CodeForbidden
	case domain.CodeValidation:
		return http.StatusBadRequest, domain.CodeValidation
	case domain.CodeConflict:
		return http.StatusConflict, domain.CodeConflict
	case domain.CodeNodeInUse:
		return http.StatusConflict, domain.CodeNodeInUse
	case domain.CodeRateLimited:
		return http.StatusTooManyRequests, domain.CodeRateLimited
	case domain.CodeBootstrapDone:
		return http.StatusBadRequest, domain.CodeBootstrapDone
	default:
		return http.StatusInternalServerError, domain.CodeInternal
	}
}

func httpStatusTextSafe(he *echo.HTTPError) string {
	switch m := he.Message.(type) {
	case string:
		return m
	case error:
		return m.Error()
	case nil:
		return http.StatusText(he.Code)
	default:
		return http.StatusText(he.Code)
	}
}

// ErrorHandler is the global Echo HTTP error handler.
func ErrorHandler(err error, c echo.Context) {
	m := mapError(err)
	rid := GetRequestID(c)
	if rid == "" {
		rid = c.Response().Header().Get(echo.HeaderXRequestID)
	}
	if rid == "" {
		rid = c.Request().Header.Get(echo.HeaderXRequestID)
	}

	if m.logFull {
		lg := loggerFor(c)
		lg.WithLevel(zerolog.ErrorLevel).
			Err(err).
			Str("request_id", rid).
			Str("method", c.Request().Method).
			Str("path", c.Path()).
			Msg("request failed")
	}

	_ = c.JSON(m.status, ErrorResponse{
		Error: ErrorDetail{
			Code:      m.code,
			Message:   m.clientMsg,
			RequestID: rid,
		},
	})
}
