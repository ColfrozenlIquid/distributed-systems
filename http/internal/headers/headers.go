package headers

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

var SEPARATOR = []byte("\r\n")

type Headers struct {
	headers map[string]string
}

func NewHeaders() *Headers {
	return &Headers{
		headers: make(map[string]string),
	}
}

func (h *Headers) Get(key string) (string, bool) {
	str, ok := h.headers[strings.ToLower(key)]
	return str, ok
}

func (h *Headers) Set(key, value string) {
	key = strings.ToLower(key)
	if prev, ok := h.headers[key]; ok {
		h.headers[key] = fmt.Sprintf("%s,%s", prev, value)
	} else {
		h.headers[key] = value
	}
}

func (h *Headers) Replace(key, value string) {
	key = strings.ToLower(key)
	h.headers[key] = value
}

func (h *Headers) GetInt(key string, defaultValue int) int {
	valueStr, exists := h.Get(key)
	if !exists {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func (h *Headers) ForEach(cb func(key, value string)) {
	for key, value := range h.headers {
		cb(key, value)
	}
}

func (h *Headers) Parse(data []byte) (n int, done bool, err error) {
	n = 0
	done = false
	for {
		idx := bytes.Index(data[n:], SEPARATOR)
		if idx == -1 {
			break
		}

		// Empty header
		if idx == 0 {
			done = true
			n += len(SEPARATOR)
			break
		}

		key, value, err := parseHeader(data[n : n+idx])
		if err != nil {
			return 0, false, err
		}
		if !isToken(key) {
			return 0, false, fmt.Errorf("invalid header name: %s", key)
		}

		n += idx + len(SEPARATOR)
		h.Set(key, value)
	}
	return
}

func parseHeader(fieldLine []byte) (key string, value string, err error) {
	parts := bytes.SplitN(fieldLine, []byte(":"), 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("malformed field-line")
	}

	name := parts[0]
	if bytes.HasSuffix(name, []byte(" ")) {
		return "", "", fmt.Errorf("malformed field-name: %s", name)
	}

	key = string(name)
	value = string(bytes.TrimSpace(parts[1]))

	return
}

func isToken(str string) bool {
	for _, c := range str {
		found := false
		if 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' {
			found = true
		}
		switch c {
		case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
			found = true
		}
		if !found {
			return false
		}
	}
	return true
}
