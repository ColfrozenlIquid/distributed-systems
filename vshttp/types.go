package vshttp

import (
	"vshttp/internal/headers"
	"vshttp/internal/response"
)

type Headers = headers.Headers
type StatusCode = response.StatusCode

const (
	OK                  StatusCode = response.OK
	Created             StatusCode = response.Created
	Accepted            StatusCode = response.Accepted
	NoContent           StatusCode = response.NoContent
	BadRequest          StatusCode = response.BadRequest
	Unauthorized        StatusCode = response.Unauthorized
	Forbidden           StatusCode = response.Forbidden
	NotFound            StatusCode = response.NotFound
	MethodNotAllowed    StatusCode = response.MethodNotAllowed
	Conflict            StatusCode = response.Conflict
	InternalServerError StatusCode = response.InternalServerError
	NotImplemented      StatusCode = response.NotImplemented
)

type Method string

const (
	GET     Method = "GET"
	HEAD    Method = "HEAD"
	POST    Method = "POST"
	PUT     Method = "PUT"
	PATCH   Method = "PATCH"
	DELETE  Method = "DELETE"
	OPTIONS Method = "OPTIONS"
)

func NewHeaders() *Headers {
	return headers.New()
}

func (m Method) String() string {
	return string(m)
}

func (m Method) Valid() bool {
	switch m {
	case GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS:
		return true
	default:
		return false
	}
}
