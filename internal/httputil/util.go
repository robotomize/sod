package httputil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-sod/sod/internal/logging"
)

func DecodeErr(ctx context.Context, w http.ResponseWriter, err error) {
	var (
		syntaxErr      *json.SyntaxError
		unmarshalError *json.UnmarshalTypeError
	)
	switch {
	case errors.As(err, &syntaxErr):
		RespBadRequestErrorf(ctx, w, `{"error": "malformed json at position %v"}`, syntaxErr.Offset)
	case errors.Is(err, io.ErrUnexpectedEOF):
		RespBadRequestErrorf(ctx, w, `{"error": "malformed json"}`)
	case errors.As(err, &unmarshalError):
		RespBadRequestErrorf(ctx, w, `{"error": "invalid value %v at position %v"}`, unmarshalError.Field, unmarshalError.Offset)
	case strings.HasPrefix(err.Error(), "json: unknown field"):
		fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
		RespBadRequestErrorf(ctx, w, `{"error": "unknown field %s"}`, fieldName)
	case errors.Is(err, io.EOF):
		RespBadRequestErrorf(ctx, w, `{"error": "body must not be empty"}`)
	case err.Error() == "http: request body too large":
		w.WriteHeader(http.StatusRequestEntityTooLarge)
	default:
		RespInternalErrorf(ctx, w, "failed to decode json: %v", err)
	}
}

func RespBadRequestErrorf(ctx context.Context, w http.ResponseWriter, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	logging.FromContext(ctx).Debug(msg)
	http.Error(w, msg, http.StatusBadRequest)
}

func RespInternalErrorf(ctx context.Context, w http.ResponseWriter, format string, args ...interface{}) {
	logging.FromContext(ctx).Errorf(format, args...)
	http.Error(w, "Internal error", http.StatusInternalServerError)
}
