package chariot

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// httpResponseValue is transport metadata created only by httpResponse().
// Ordinary maps remain ordinary JSON response bodies.
type httpResponseValue struct {
	StatusCode int
	Headers    http.Header
	Body       Value
}

func buildHTTPResponseValue(args ...Value) (Value, error) {
	if len(args) != 2 && len(args) != 3 {
		return nil, errors.New("httpResponse requires status and body, or status, headers, and body")
	}
	for index, argument := range args {
		if entry, ok := argument.(ScopeEntry); ok {
			args[index] = entry.Value
		}
	}
	statusNumber, ok := args[0].(Number)
	if !ok || float64(int(statusNumber)) != float64(statusNumber) {
		return nil, fmt.Errorf("HTTP status must be a whole number, got %T", args[0])
	}
	status := int(statusNumber)
	if status < 100 || status > 599 {
		return nil, fmt.Errorf("HTTP status must be between 100 and 599")
	}

	response := &httpResponseValue{StatusCode: status, Headers: make(http.Header)}
	if len(args) == 2 {
		response.Body = args[1]
		return response, nil
	}

	headers, ok := args[1].(*MapValue)
	if !ok {
		return nil, fmt.Errorf("HTTP headers must be a map, got %T", args[1])
	}
	for name, rawValue := range headers.Values {
		name = strings.TrimSpace(name)
		if !validHTTPHeaderName(name) {
			return nil, fmt.Errorf("invalid HTTP header name %q", name)
		}
		switch value := rawValue.(type) {
		case Str:
			if strings.ContainsAny(string(value), "\r\n") {
				return nil, fmt.Errorf("HTTP header %q contains a newline", name)
			}
			response.Headers.Set(name, string(value))
		case *ArrayValue:
			if value.Length() == 0 {
				return nil, fmt.Errorf("HTTP header %q values cannot be empty", name)
			}
			for _, item := range value.Elements {
				text, ok := item.(Str)
				if !ok || strings.ContainsAny(string(text), "\r\n") {
					return nil, fmt.Errorf("HTTP header %q values must be newline-free strings", name)
				}
				response.Headers.Add(name, string(text))
			}
		default:
			return nil, fmt.Errorf("HTTP header %q must be a string or array of strings, got %T", name, rawValue)
		}
	}
	response.Body = args[2]
	return response, nil
}

func validHTTPHeaderName(name string) bool {
	if name == "" {
		return false
	}
	const separators = "()<>@,;:\\\"/[]?={} \t"
	for _, character := range name {
		if character < 33 || character > 126 || strings.ContainsRune(separators, character) {
			return false
		}
	}
	return true
}
