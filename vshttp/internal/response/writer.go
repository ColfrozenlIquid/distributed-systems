package response

import (
	"fmt"
	"io"
	"vshttp/internal/headers"
)

type Writer struct {
	writer io.Writer
}

func NewWriter(writer io.Writer) *Writer {
	return &Writer{writer: writer}
}

func (w *Writer) WriteStatusLine(statusCode StatusCode) error {
	statusLine := "HTTP/1.1 " + statusCode.String() + " " + statusCode.ReasonPhrase() + "\r\n"
	_, err := w.writer.Write([]byte(statusLine))
	return err
}
func (w *Writer) WriteHeaders(h *headers.Headers) error {
	var bytes []byte
	h.ForEach(func(key, value string) {
		bytes = fmt.Appendf(bytes, "%s: %s\r\n", key, value)
	})
	bytes = fmt.Append(bytes, "\r\n")
	_, err := w.writer.Write(bytes)
	return err
}
func (w *Writer) WriteBody(p []byte) (int, error) {
	return w.writer.Write(p)
}
