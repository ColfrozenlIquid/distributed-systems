package request

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"vshttp/internal/headers"
)

type Request struct {
	RequestLine RequestLine
	Headers     *headers.Headers
	Body        []byte
	state       parserState
}

type RequestLine struct {
	Method        string
	RequestTarget string
	HttpVersion   string
}

type parserState string

const (
	StateInit    parserState = "initialized"
	StateHeaders parserState = "headers"
	StateBody    parserState = "body"
	StateDone    parserState = "done"
	StateError   parserState = "error"
)

func (r *Request) hasBody() bool {
	length := r.Headers.GetInt(string(headers.ContentLength), 0)
	return length > 0
}

var SEPARATOR = []byte("\r\n")

var ErrorMalformedRequestLine = fmt.Errorf("malformed request-line")
var ErrorUnsupportedHttpVersion = fmt.Errorf("unsupported HTTP version")
var ErrorRequestInErrorState = fmt.Errorf("request in error state")

func FromReader(reader io.Reader) (*Request, error) {
	request := &Request{
		state:   StateInit,
		Headers: headers.New(),
		Body:    []byte(""),
	}

	buf := make([]byte, 1024)
	bufLen := 0
	for !(request.state == StateDone || request.state == StateError) {
		// read into buffer
		numBytesRead, err := reader.Read(buf[bufLen:])
		if err != nil {
			if errors.Is(err, io.EOF) {
				request.state = StateDone
				break
			}
			return nil, err
		}
		bufLen += numBytesRead

		// parse buffer
		numBytesParsed, err := request.parse(buf[:bufLen])
		if err != nil {
			return nil, err
		}

		copy(buf, buf[numBytesParsed:bufLen])
		bufLen -= numBytesParsed
	}
	return request, nil
}

func (r *Request) parse(data []byte) (int, error) {
	read := 0
outer:
	for {
		currentData := data[read:]
		if len(currentData) == 0 {
			break
		}
		switch r.state {
		case StateError:
			return 0, ErrorRequestInErrorState
		case StateInit:
			requestLine, n, err := parseRequestLine(currentData)
			if err != nil {
				r.state = StateError
				return 0, err
			}

			if n == 0 {
				break outer
			}
			r.RequestLine = *requestLine
			read += n
			r.state = StateHeaders

		case StateHeaders:
			n, done, err := r.Headers.Parse(currentData)
			if err != nil {
				r.state = StateError
				return 0, err
			}
			if n == 0 {
				break outer
			}
			read += n
			if done {
				if r.hasBody() {
					r.state = StateBody
				} else {
					r.state = StateDone
				}
			}
		case StateBody:
			length := r.Headers.GetInt("Content-Length", 0)
			if length == 0 {
				r.state = StateDone
				break
			}
			remaining := min(length-len(r.Body), len(currentData))
			r.Body = append(r.Body, currentData[:remaining]...)
			read += remaining
			if len(r.Body) == length {
				r.state = StateDone
			}
		case StateDone:
			break outer
		default:
			panic("invalid state")
		}
	}
	return read, nil
}

func parseRequestLine(line []byte) (*RequestLine, int, error) {
	idx := bytes.Index(line, SEPARATOR)
	if idx == -1 {
		return nil, 0, nil
	}
	startLine := line[:idx]
	read := idx + len(SEPARATOR)

	parts := bytes.Split(startLine, []byte(" "))
	if len(parts) != 3 {
		return nil, 0, ErrorMalformedRequestLine
	}

	httpVersionParts := bytes.Split(parts[2], []byte("/"))
	if len(httpVersionParts) != 2 || string(httpVersionParts[0]) != "HTTP" {
		return nil, 0, ErrorMalformedRequestLine
	}

	switch string(httpVersionParts[1]) {
	case "1.0", "1.1", "2", "3": // valid HTTP versions
	default:
		return nil, 0, ErrorUnsupportedHttpVersion
	}

	requestLine := &RequestLine{
		Method:        string(parts[0]),
		RequestTarget: string(parts[1]),
		HttpVersion:   string(httpVersionParts[1]),
	}

	return requestLine, read, nil
}
