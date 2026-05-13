package text

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// RequestToCurl generates a basic curl command from an http.Request
func RequestToCurl(req *http.Request) (string, error) {
	if req == nil {
		return "", fmt.Errorf("request is nil")
	}

	var curlCmd strings.Builder
	curlCmd.WriteString(fmt.Sprintf("curl -X '%s'", req.Method))

	// Add headers
	for key, values := range req.Header {
		for _, value := range values {
			curlCmd.WriteString(fmt.Sprintf(" -H '%s: %s'", key, value))
		}
	}

	// Add body if it exists
	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read body: %w", err)
		}

		// Reconstruct the body so it can be read again by the HTTP client!
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		if len(bodyBytes) > 0 {
			// Escape single quotes for safe shell usage.
			escapedBody := strings.ReplaceAll(string(bodyBytes), "'", `'\''`)
			curlCmd.WriteString(fmt.Sprintf(" -d '%s'", escapedBody))
		}
	}

	// Add URL
	if req.URL != nil {
		curlCmd.WriteString(fmt.Sprintf(" '%s'", req.URL.String()))
	}

	return curlCmd.String(), nil
}

// MustRequestToCurl is like RequestToCurl but calls log.Fatal if an error occurs.
func MustRequestToCurl(req *http.Request) string {
	cmd, err := RequestToCurl(req)
	if err != nil {
		log.Fatalf("failed to generate curl command: %v", err)
	}
	return cmd
}
