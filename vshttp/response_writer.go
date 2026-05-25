package vshttp

import (
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"vshttp/internal/headers"
	"vshttp/internal/response"
)

var ErrResponseCommitted = errors.New("response already committed")

type ResponseWriter interface {
	Header() *Headers
	WriteHeader(status StatusCode)
	Write([]byte) (int, error)

	Text(status StatusCode, body string) error
	JSON(status StatusCode, value any) error
	NoContent() error
}

type responseWriter struct {
	raw       *response.Writer
	headers   *Headers
	status    StatusCode
	statusSet bool
	committed bool
	err       error
}

func newResponseWriter(writer io.Writer) *responseWriter {
	return &responseWriter{
		raw:     response.NewWriter(writer),
		headers: defaultHeaders(),
		status:  OK,
	}
}

func defaultHeaders() *Headers {
	h := NewHeaders()
	h.Replace("Connection", "close")
	return h
}

func (w *responseWriter) Header() *Headers {
	return w.headers
}

func (w *responseWriter) WriteHeader(status StatusCode) {
	if w.committed || w.statusSet {
		return
	}
	w.status = status
	w.statusSet = true
}

func (w *responseWriter) Write(data []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	if !w.statusSet {
		w.WriteHeader(OK)
	}
	if !w.committed {
		contentType := string(headers.ContentType)
		if _, ok := w.headers.Get(contentType); !ok && len(data) > 0 {
			w.headers.Replace(contentType, "application/octet-stream")
		}
		if err := w.commit(); err != nil {
			return 0, err
		}
	}
	return w.raw.WriteBody(data)
}

func (w *responseWriter) Text(status StatusCode, body string) error {
	if w.committed {
		return ErrResponseCommitted
	}
	data := []byte(body)
	w.headers.Replace(string(headers.ContentType), "text/plain; charset=utf-8")
	w.headers.Replace(string(headers.ContentLength), strconv.Itoa(len(data)))
	w.WriteHeader(status)
	_, err := w.Write(data)
	return err
}

func (w *responseWriter) JSON(status StatusCode, value any) error {
	if w.committed {
		return ErrResponseCommitted
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	w.headers.Replace(string(headers.ContentType), "application/json")
	w.headers.Replace(string(headers.ContentLength), strconv.Itoa(len(data)))
	w.WriteHeader(status)
	_, err = w.Write(data)
	return err
}

func (w *responseWriter) NoContent() error {
	if w.committed {
		return ErrResponseCommitted
	}
	w.headers.Replace(string(headers.ContentLength), "0")
	w.WriteHeader(NoContent)
	return w.commit()
}

func (w *responseWriter) finish() error {
	if w.err != nil {
		return w.err
	}
	if !w.committed {
		if _, ok := w.headers.Get(string(headers.ContentLength)); !ok {
			w.headers.Replace(string(headers.ContentLength), "0")
		}
		if !w.statusSet {
			w.WriteHeader(OK)
		}
		return w.commit()
	}
	return nil
}

func (w *responseWriter) writeError(status StatusCode, message string) {
	if w.committed {
		return
	}
	w.headers = defaultHeaders()
	w.status = OK
	w.statusSet = false
	_ = w.Text(status, message)
}

func (w *responseWriter) commit() error {
	if w.err != nil {
		return w.err
	}
	if w.committed {
		return nil
	}
	if !w.statusSet {
		w.WriteHeader(OK)
	}
	if err := w.raw.WriteStatusLine(w.status); err != nil {
		w.err = err
		return err
	}
	if err := w.raw.WriteHeaders(w.headers); err != nil {
		w.err = err
		return err
	}
	w.committed = true
	return nil
}
