package helpers_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockHttpServer(t *testing.T) {
	server := NewMockServer()
	defer server.Close()

	server.AddStringResponse("/string-path", "string body")
	server.AddYamlResponse("/yaml-path", map[string]any{"key": "value"})
	server.AddResponse(
		"/generic-path",
		http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("generic body")),
		},
	)

	// Use server.URL() in your tests to make requests to the server.
	// The server will respond with the response that was added for the path.
	assert.HTTPBodyContains(t, server.ServeHTTP, "GET", server.URL()+"/string-path", nil, "string body")
	assert.HTTPBodyContains(t, server.ServeHTTP, "GET", server.URL()+"/yaml-path", nil, "key: value")
	assert.HTTPBodyContains(t, server.ServeHTTP, "GET", server.URL()+"/generic-path", nil, "generic body")
	assert.HTTPStatusCode(t, server.ServeHTTP, "GET", server.URL()+"/not-found", nil, 404)

	assert.Equal(t, 1, server.GetHitCount("/string-path"))
	assert.Equal(t, 1, server.GetHitCount("/yaml-path"))
	assert.Equal(t, 1, server.GetHitCount("/generic-path"))
	assert.Equal(t, 1, server.GetHitCount("/not-found"))
	assert.Equal(t, 0, server.GetHitCount("/not-called"))
}
