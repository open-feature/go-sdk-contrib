package http_service

import (
	"context"
	"io"
	"net/http"
)

type iHTTPClient interface {
	Request(ctx context.Context, method string, url string, body io.Reader) (*http.Response, error)
}

type httpClient struct {
	Client *http.Client
}

func (c *httpClient) instance() http.Client {
	if c.Client == nil {
		c.Client = &http.Client{}
	}
	return *c.Client
}

func (c *httpClient) Request(ctx context.Context, method string, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	client := c.instance()
	return client.Do(req)
}
