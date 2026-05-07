package vshttp

import (
	"net/url"
	internal "vshttp/internal/request"
)

type Request struct {
	Method        Method
	Path          string
	Query         map[string][]string
	Headers       *Headers
	Body          []byte
	RequestTarget string
	HTTPVersion   string
	Params        map[string]string
}

func newRequest(raw *internal.Request) *Request {
	requestTarget := raw.RequestLine.RequestTarget
	path := requestTarget
	query := map[string][]string{}

	if parsed, err := url.ParseRequestURI(requestTarget); err == nil {
		path = parsed.Path
		query = parsed.Query()
	}
	if path == "" {
		path = "/"
	}

	return &Request{
		Method:        Method(raw.RequestLine.Method),
		Path:          path,
		Query:         query,
		Headers:       raw.Headers,
		Body:          raw.Body,
		RequestTarget: requestTarget,
		HTTPVersion:   raw.RequestLine.HttpVersion,
		Params:        map[string]string{},
	}
}

func (r *Request) Param(name string) string {
	if r == nil {
		return ""
	}
	return r.Params[name]
}

func (r *Request) QueryValue(name string) string {
	if r == nil {
		return ""
	}
	values := r.Query[name]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (r *Request) Header(name string) string {
	if r == nil || r.Headers == nil {
		return ""
	}
	value, _ := r.Headers.Get(name)
	return value
}
