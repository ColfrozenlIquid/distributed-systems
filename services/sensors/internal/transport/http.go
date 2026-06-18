package transport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Handle HTTP transport layer
type HTTPClient struct {
	httpClient *http.Client
	BaseURL    string
	Timeout    time.Duration
}

type Response struct {
	StatusCode int
	Body       []byte
	Header     http.Header
}

func NewHTTPClient(cfg TransportConfig) *HTTPClient {
	return &HTTPClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		BaseURL:    cfg.HttpAddress,
		Timeout:    cfg.HttpTimeout,
	}
}

func (c *HTTPClient) send(method string, path string, body any) (*Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Body:       respBody,
		Header:     resp.Header,
	}, nil
}

func (c *HTTPClient) Post(path string, body any) (*Response, error) {
	return c.send(http.MethodPost, path, body)
}
