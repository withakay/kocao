package controlplanecli

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrMissingToken  = errors.New("missing bearer token")
	ErrMissingAPIURL = errors.New("missing api url")
)

type APIError struct {
	StatusCode  int
	Message     string
	Body        string
	Method      string
	URL         string
	ContentType string
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Message) != "" {
		return fmt.Sprintf("api request failed (%d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("api request failed (%d)", e.StatusCode)
}

type DecodeError struct {
	Method      string
	URL         string
	StatusCode  int
	ContentType string
	BodyPreview string
	Cause       error
}

func (e *DecodeError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return fmt.Sprintf("decode response: %v", e.Cause)
	}
	return "decode response"
}

func (e *DecodeError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func explainCommandError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, ErrMissingToken) {
		return "missing bearer token: set KOCAO_TOKEN or pass --token"
	}
	if errors.Is(err, ErrMissingAPIURL) {
		return "missing api url: set KOCAO_API_URL or pass --api-url"
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 401:
			return "unauthorized: check bearer token and required scopes"
		case 403:
			return "forbidden: token does not include required scope for this operation"
		case 404:
			if strings.TrimSpace(apiErr.Message) != "" {
				return apiErr.Message
			}
			return "resource not found"
		default:
			if strings.TrimSpace(apiErr.Message) != "" {
				return apiErr.Error()
			}
		}
	}
	var decErr *DecodeError
	if errors.As(err, &decErr) {
		ct := strings.TrimSpace(decErr.ContentType)
		if ct == "" {
			ct = "(missing)"
		}
		msg := fmt.Sprintf("received non-JSON response from %s (%d, content-type: %s)", decErr.URL, decErr.StatusCode, ct)
		if strings.Contains(strings.ToLower(decErr.BodyPreview), "<html") {
			msg += "; this looks like an HTML page (wrong API URL or reverse proxy route)"
		}
		return msg
	}
	return err.Error()
}
