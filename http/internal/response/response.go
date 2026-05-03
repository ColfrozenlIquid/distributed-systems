package response

import (
	"http/internal/headers"
	"strconv"
)

type Response struct {
}

type StatusCode int

const (
	OK                  StatusCode = 200
	Created             StatusCode = 201
	Accepted            StatusCode = 202
	NoContent           StatusCode = 204
	BadRequest          StatusCode = 400
	Unauthorized        StatusCode = 401
	Forbidden           StatusCode = 403
	NotFound            StatusCode = 404
	MethodNotAllowed    StatusCode = 405
	Conflict            StatusCode = 409
	InternalServerError StatusCode = 500
	NotImplemented      StatusCode = 501
)

func (s StatusCode) String() string {
	return string(rune(s))
}

func (s StatusCode) ReasonPhrase() string {
	reasonPhrase := ""
	switch s {
	case OK:
		reasonPhrase = "OK"
	case Created:
		reasonPhrase = "Created"
	case Accepted:
		reasonPhrase = "Accepted"
	case NoContent:
		reasonPhrase = "No Content"
	case BadRequest:
		reasonPhrase = "Bad Request"
	case Unauthorized:
		reasonPhrase = "Unauthorized"
	case Forbidden:
		reasonPhrase = "Forbidden"
	case NotFound:
		reasonPhrase = "Not Found"
	case MethodNotAllowed:
		reasonPhrase = "Method Not Allowed"
	case Conflict:
		reasonPhrase = "Conflict"
	case InternalServerError:
		reasonPhrase = "Internal Server Error"
	case NotImplemented:
		reasonPhrase = "Not Implemented"
	}
	return reasonPhrase
}

func GetDefaultHeaders(contentLength int) *headers.Headers {
	h := headers.NewHeaders()
	h.Set("Content-Length", strconv.Itoa(contentLength))
	h.Set("Connection", "close")
	h.Set("Content-Type", "text/plain")
	return h
}

type HandlerError struct {
	StatusCode StatusCode
	Message    string
}
