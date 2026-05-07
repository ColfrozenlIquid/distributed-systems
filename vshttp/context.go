package vshttp

import (
	"encoding/json"
	"errors"
	"log"
)

type Context struct {
	Request *Request
	Writer  ResponseWriter
	params  map[string]string
}

type ContextHandlerFunc func(*Context) error

func (f ContextHandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
	ctx := &Context{
		Request: r,
		Writer:  w,
		params:  r.Params,
	}
	if err := f(ctx); err != nil {
		handleContextError(w, err)
	}
}

func (c *Context) Param(name string) string {
	if c == nil {
		return ""
	}
	return c.params[name]
}

func (c *Context) Query(name string) string {
	if c == nil || c.Request == nil {
		return ""
	}
	return c.Request.QueryValue(name)
}

func (c *Context) Header(name string) string {
	if c == nil || c.Request == nil {
		return ""
	}
	return c.Request.Header(name)
}

func (c *Context) JSONBody(v any) error {
	if c == nil || c.Request == nil {
		return errors.New("request is nil")
	}
	return json.Unmarshal(c.Request.Body, v)
}

func (c *Context) JSON(status StatusCode, v any) error {
	if c == nil || c.Writer == nil {
		return errors.New("response writer is nil")
	}
	return c.Writer.JSON(status, v)
}

func (c *Context) Text(status StatusCode, body string) error {
	if c == nil || c.Writer == nil {
		return errors.New("response writer is nil")
	}
	return c.Writer.Text(status, body)
}

func (c *Context) NoContent() error {
	if c == nil || c.Writer == nil {
		return errors.New("response writer is nil")
	}
	return c.Writer.NoContent()
}

func handleContextError(w ResponseWriter, err error) {
	if rw, ok := w.(*responseWriter); ok && rw.committed {
		log.Printf("Handler returned error after response was committed: %v", err)
		return
	}

	if httpErr, ok := errors.AsType[*HTTPError](err); ok {
		if httpErr.Message == "" {
			httpErr.Message = httpErr.Status.ReasonPhrase()
		}
		if rw, ok := w.(*responseWriter); ok {
			rw.writeError(httpErr.Status, httpErr.Message)
			return
		}
		_ = w.Text(httpErr.Status, httpErr.Message)
		return
	}

	log.Printf("Unexpected handler error: %v", err)
	if rw, ok := w.(*responseWriter); ok {
		rw.writeError(InternalServerError, "internal server error")
		return
	}
	_ = w.Text(InternalServerError, "internal server error")
}
