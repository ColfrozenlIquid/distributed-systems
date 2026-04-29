package request

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

type Request struct {
	RequestLine RequestLine
	Headers     map[string]string
	Body        []byte
}

type RequestLine struct {
	Method        string
	RequestTarget string
	HttpVersion   string
}

var SEPARATOR = "\r\n"
var ERROR_MALFORMED_REQUEST_LINE = fmt.Errorf("malformed request-line")

func RequestFromReader(reader io.Reader) (*Request, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("unable to io.ReadAll"), err)
	}

	str := string(data)
	requestLine, _, err := parseRequestLine(str)
	if err != nil {
		return nil, err
	}

	return &Request{
		RequestLine: *requestLine,
		Headers:     nil,
		Body:        nil,
	}, nil
}

func parseRequestLine(line string) (*RequestLine, string, error) {
	idx := strings.Index(line, SEPARATOR)
	if idx == -1 {
		return nil, line, nil
	}

	startLine := line[:idx]
	restOfMessage := line[idx+len(SEPARATOR):]

	parts := strings.Split(startLine, " ")
	if len(parts) != 3 {
		return nil, line, ERROR_MALFORMED_REQUEST_LINE
	}

	httpVersionParts := strings.Split(parts[2], "/")
	if len(httpVersionParts) != 2 || httpVersionParts[0] != "HTTP" {
		return nil, line, ERROR_MALFORMED_REQUEST_LINE
	}

	switch httpVersionParts[1] {
	case "1.1", "2", "3": // valid HTTP versions
	default:
		return nil, line, ERROR_MALFORMED_REQUEST_LINE
	}

	requestLine := &RequestLine{
		Method:        parts[0],
		RequestTarget: parts[1],
		HttpVersion:   httpVersionParts[1],
	}

	return requestLine, restOfMessage, nil
}
