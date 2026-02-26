package controlplaneapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

const maxJSONBodyBytes int64 = 1 << 20 // 1 MiB

type httpError struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, httpError{Error: msg})
}

type requestError struct {
	status int
	msg    string
	err    error
}

func (e *requestError) Error() string {
	if e.err == nil {
		return e.msg
	}
	return e.msg + ": " + e.err.Error()
}

func (e *requestError) Unwrap() error {
	return e.err
}

func jsonErrorResponse(err error) (int, string) {
	var re *requestError
	if errors.As(err, &re) {
		return re.status, re.msg
	}
	return http.StatusBadRequest, "invalid json"
}

func writeJSONError(w http.ResponseWriter, err error) {
	status, msg := jsonErrorResponse(err)
	writeError(w, status, msg)
}

func readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	ct := r.Header.Get("Content-Type")
	if ct != "" && !strings.HasPrefix(strings.ToLower(strings.TrimSpace(ct)), "application/json") {
		return &requestError{status: http.StatusUnsupportedMediaType, msg: "content-type must be application/json"}
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			return &requestError{status: http.StatusRequestEntityTooLarge, msg: "request body too large", err: err}
		}
		return &requestError{status: http.StatusBadRequest, msg: "invalid json", err: err}
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return &requestError{status: http.StatusBadRequest, msg: "invalid json"}
		}
		return &requestError{status: http.StatusBadRequest, msg: "invalid json", err: err}
	}
	return nil
}
