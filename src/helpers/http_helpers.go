package helpers_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"

	"gopkg.in/yaml.v3"
)

/**
 * MockServer is a simple HTTP server meant to be used in tests to mock responses.
 * It works by matching the path of the request to a response that was added to
 * the server using the AddResponse method.
 *
 * Example:
 *   func TestMockHttpServer(t *testing.T) {
 *  	server := NewMockServer()
 *  	defer server.Close()
 *
 *  	server.AddStringResponse("/string-path", "string body")
 *  	server.AddYamlResponse("/yaml-path", map[string]any{"key": "value"})
 *  	server.AddResponse(
 *  		"/generic-path",
 *  		http.Response{
 *  			StatusCode: 200,
 *  			Body:       io.NopCloser(strings.NewReader("generic body")),
 *  		},
 *  	)
 *
 *  	// Use server.URL() in your tests to make requests to the server.
 *  	// The server will respond with the response that was added for the path.
 *  	assert.HTTPBodyContains(t, server.ServeHTTP, "GET", server.URL()+"/string-path", nil, "string body")
 *  	assert.HTTPBodyContains(t, server.ServeHTTP, "GET", server.URL()+"/yaml-path", nil, "key: value")
 *  	assert.HTTPBodyContains(t, server.ServeHTTP, "GET", server.URL()+"/generic-path", nil, "generic body")
 *  	assert.HTTPStatusCode(t, server.ServeHTTP, "GET", server.URL()+"/not-found", nil, 404)
 *
 *  	assert.Equal(t, 1, server.GetHitCount("/string-path"))
 *  	assert.Equal(t, 1, server.GetHitCount("/yaml-path"))
 *  	assert.Equal(t, 1, server.GetHitCount("/generic-path"))
 *  	assert.Equal(t, 1, server.GetHitCount("/not-found"))
 *  	assert.Equal(t, 0, server.GetHitCount("/not-called"))
 *   }
 */
type MockServer struct {
	server    *httptest.Server
	responses map[string]http.Response
	hitCount  map[string]int
}

func NewMockServer() *MockServer {
	mockServer := &MockServer{
		responses: make(map[string]http.Response),
		hitCount:  make(map[string]int),
	}
	server := httptest.NewServer(mockServer)
	mockServer.server = server
	return mockServer
}

func (m *MockServer) Close() {
	m.server.Close()
}

func (m *MockServer) URL() string {
	return m.server.URL
}

func (m *MockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// this is weird, but works. `r.URL` doesn't include the protocol or host,
	// so we need to add figure those parts out ourselves. Luckily, if we end up
	// here, we know that the URL is this server's URL :win:
	url := m.URL() + r.URL.Path
	m.hitCount[url]++
	if response, ok := m.responses[url]; ok {
		w.WriteHeader(response.StatusCode)
		io.Copy(w, response.Body)
	} else {
		w.WriteHeader(http.StatusNotFound)
		slog.Warn("No response found for path", "path", url)
	}
}

func (m *MockServer) GetHitCount(path string) int {
	if !strings.HasPrefix(path, "http") {
		path = m.URL() + path
	}
	return m.hitCount[path]
}

func (m *MockServer) AddResponse(path string, response http.Response) {
	if !strings.HasPrefix(path, "http") {
		path = m.URL() + path
	}
	m.responses[path] = response
}

func (m *MockServer) AddYamlResponse(path string, body map[string]any) {
	m.AddResponse(path, NewYamlResponse(body))
}

func (m *MockServer) AddStringResponse(path string, body string) {
	m.AddResponse(path, NewStringResponse(body))
}

func (m *MockServer) Reset() {
	m.responses = make(map[string]http.Response)
	m.hitCount = make(map[string]int)
}

func NewYamlResponse(body map[string]any) http.Response {
	yamlBody, err := yaml.Marshal(body)
	if err != nil {
		slog.Error("Failed to marshal response body", "error", err)
		panic(err)
	}

	return http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(yamlBody)),
	}
}

func NewStringResponse(body string) http.Response {
	return http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
