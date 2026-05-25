package vshttp

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"vshttp/internal/request"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseWriterTextWritesStatusHeadersAndBody(t *testing.T) {
	var buf bytes.Buffer
	w := newResponseWriter(&buf)

	err := w.Text(OK, "Hello")

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "HTTP/1.1 200 OK\r\n")
	assert.Contains(t, buf.String(), "content-type: text/plain; charset=utf-8\r\n")
	assert.Contains(t, buf.String(), "content-length: 5\r\n")
	assert.True(t, strings.HasSuffix(buf.String(), "\r\n\r\nHello"))
}

func TestResponseWriterJSONReturnsMarshalErrorsBeforeCommitting(t *testing.T) {
	var buf bytes.Buffer
	w := newResponseWriter(&buf)

	err := w.JSON(OK, func() {})

	require.Error(t, err)
	assert.Empty(t, buf.String())
}

func TestResponseWriterOnlyAcceptsFirstStatus(t *testing.T) {
	var buf bytes.Buffer
	w := newResponseWriter(&buf)

	w.WriteHeader(Created)
	w.WriteHeader(OK)
	_, err := w.Write([]byte("created"))

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "HTTP/1.1 201 Created\r\n")
	assert.NotContains(t, buf.String(), "HTTP/1.1 200 OK\r\n")
}

func TestResponseWriterNoContentWritesNoBody(t *testing.T) {
	var buf bytes.Buffer
	w := newResponseWriter(&buf)

	err := w.NoContent()

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "HTTP/1.1 204 No Content\r\n")
	assert.Contains(t, buf.String(), "content-length: 0\r\n")
	assert.True(t, strings.HasSuffix(buf.String(), "\r\n\r\n"))
}

func TestRequestNormalizesPathAndQuery(t *testing.T) {
	raw, err := request.FromReader(strings.NewReader("GET /coffee?size=large&shot=1&shot=2 HTTP/1.1\r\nHost: example.com\r\n\r\n"))
	require.NoError(t, err)

	req := newRequest(raw)

	assert.Equal(t, GET, req.Method)
	assert.Equal(t, "/coffee", req.Path)
	assert.Equal(t, "large", req.QueryValue("size"))
	assert.Equal(t, []string{"1", "2"}, req.Query["shot"])
	assert.Equal(t, "/coffee?size=large&shot=1&shot=2", req.RequestTarget)
}

func TestRouterMatchesExactRouteBeforeParamRoute(t *testing.T) {
	router := NewRouter()
	router.Get("/{name}", func(c *Context) error {
		return c.Text(OK, "param "+c.Param("name"))
	})
	router.Get("/health", func(c *Context) error {
		return c.Text(OK, "exact")
	})

	var buf bytes.Buffer
	w := newResponseWriter(&buf)
	router.ServeHTTP(w, &Request{
		Method:  GET,
		Path:    "/health",
		Headers: NewHeaders(),
		Query:   map[string][]string{},
		Params:  map[string]string{},
	})

	assert.Contains(t, buf.String(), "\r\n\r\nexact")
}

func TestRouterMatchesPathParams(t *testing.T) {
	router := NewRouter()
	router.Get("/users/{id}", func(c *Context) error {
		return c.JSON(OK, map[string]string{"id": c.Param("id")})
	})

	var buf bytes.Buffer
	w := newResponseWriter(&buf)
	router.ServeHTTP(w, &Request{
		Method:  GET,
		Path:    "/users/42",
		Headers: NewHeaders(),
		Query:   map[string][]string{},
		Params:  map[string]string{},
	})

	assert.Contains(t, buf.String(), "content-type: application/json\r\n")
	assert.Contains(t, buf.String(), "\r\n\r\n{\"id\":\"42\"}")
}

func TestRouterReturnsMethodNotAllowedWhenPathExists(t *testing.T) {
	router := NewRouter()
	router.Get("/users", func(c *Context) error { return nil })

	var buf bytes.Buffer
	w := newResponseWriter(&buf)
	router.ServeHTTP(w, &Request{
		Method:  POST,
		Path:    "/users",
		Headers: NewHeaders(),
		Query:   map[string][]string{},
		Params:  map[string]string{},
	})

	assert.Contains(t, buf.String(), "HTTP/1.1 405 Method Not Allowed\r\n")
	assert.Contains(t, buf.String(), "allow: GET\r\n")
	assert.Contains(t, buf.String(), "\r\n\r\nmethod not allowed")
}

func TestContextHandlerConvertsUnexpectedErrorsToInternalServerError(t *testing.T) {
	handler := ContextHandlerFunc(func(c *Context) error {
		return errors.New("database unavailable")
	})

	var buf bytes.Buffer
	w := newResponseWriter(&buf)
	handler.ServeHTTP(w, &Request{
		Method:  GET,
		Path:    "/",
		Headers: NewHeaders(),
		Query:   map[string][]string{},
		Params:  map[string]string{},
	})

	assert.Contains(t, buf.String(), "HTTP/1.1 500 Internal Server Error\r\n")
	assert.Contains(t, buf.String(), "\r\n\r\ninternal server error")
}

func TestContextHandlerErrorOverridesBufferedStatus(t *testing.T) {
	handler := ContextHandlerFunc(func(c *Context) error {
		c.Writer.WriteHeader(Created)
		return errors.New("database unavailable")
	})

	var buf bytes.Buffer
	w := newResponseWriter(&buf)
	handler.ServeHTTP(w, &Request{
		Method:  GET,
		Path:    "/",
		Headers: NewHeaders(),
		Query:   map[string][]string{},
		Params:  map[string]string{},
	})

	assert.Contains(t, buf.String(), "HTTP/1.1 500 Internal Server Error\r\n")
	assert.NotContains(t, buf.String(), "HTTP/1.1 201 Created\r\n")
}

func TestContextHandlerWritesHTTPErrorStatus(t *testing.T) {
	handler := ContextHandlerFunc(func(c *Context) error {
		return Error(BadRequest, "invalid payload")
	})

	var buf bytes.Buffer
	w := newResponseWriter(&buf)
	handler.ServeHTTP(w, &Request{
		Method:  POST,
		Path:    "/",
		Headers: NewHeaders(),
		Query:   map[string][]string{},
		Params:  map[string]string{},
	})

	assert.Contains(t, buf.String(), "HTTP/1.1 400 Bad Request\r\n")
	assert.Contains(t, buf.String(), "\r\n\r\ninvalid payload")
}
