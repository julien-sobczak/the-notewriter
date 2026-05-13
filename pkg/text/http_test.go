package text

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMustRequestToCurl(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "https://example.com/api", nil)
		require.NoError(t, err)

		cmd := MustRequestToCurl(req)
		assert.Equal(t, "curl -X 'GET' 'https://example.com/api'", cmd)
	})
}

func TestRequestToCurl(t *testing.T) {
	t.Run("GET request", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "https://www.googleapis.com/books/v1/volumes?q=good+inside", nil)
		require.NoError(t, err)

		cmd, err := RequestToCurl(req)
		require.NoError(t, err)
		assert.Equal(t, "curl -X 'GET' 'https://www.googleapis.com/books/v1/volumes?q=good+inside'", cmd)
	})

	t.Run("GET request with header", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "https://en.wikipedia.org/w/api.php?action=query", nil)
		require.NoError(t, err)
		req.Header.Set("Accept", "application/json")

		cmd, err := RequestToCurl(req)
		require.NoError(t, err)
		assert.Equal(t, "curl -X 'GET' -H 'Accept: application/json' 'https://en.wikipedia.org/w/api.php?action=query'", cmd)
	})

	t.Run("POST request with body", func(t *testing.T) {
		body := strings.NewReader("0525538836")
		req, err := http.NewRequest(http.MethodPost, "http://localhost:1969/search", body)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "text/plain")

		cmd, err := RequestToCurl(req)
		require.NoError(t, err)
		assert.Equal(t, "curl -X 'POST' -H 'Content-Type: text/plain' -d '0525538836' 'http://localhost:1969/search'", cmd)

		// Ensure the body is still readable after calling RequestToCurl
		remaining, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.Equal(t, "0525538836", string(remaining))
	})

	t.Run("POST request with body containing single quotes", func(t *testing.T) {
		body := strings.NewReader("it's a query")
		req, err := http.NewRequest(http.MethodPost, "http://localhost:1969/search", body)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "text/plain")

		cmd, err := RequestToCurl(req)
		require.NoError(t, err)
		assert.Equal(t, "curl -X 'POST' -H 'Content-Type: text/plain' -d 'it'\\''s a query' 'http://localhost:1969/search'", cmd)
	})

	t.Run("nil request", func(t *testing.T) {
		_, err := RequestToCurl(nil)
		require.Error(t, err)
	})

	t.Run("request with nil URL", func(t *testing.T) {
		req := &http.Request{
			Method: http.MethodGet,
			URL:    nil,
			Header: http.Header{},
		}

		cmd, err := RequestToCurl(req)
		require.NoError(t, err)
		assert.Equal(t, "curl -X 'GET'", cmd)
	})

	t.Run("request with nil body", func(t *testing.T) {
		u, _ := url.Parse("https://example.com/api")
		req := &http.Request{
			Method: http.MethodPost,
			URL:    u,
			Header: http.Header{},
			Body:   nil,
		}

		cmd, err := RequestToCurl(req)
		require.NoError(t, err)
		assert.Equal(t, "curl -X 'POST' 'https://example.com/api'", cmd)
	})
}
