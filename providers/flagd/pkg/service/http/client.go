package http_service

import (
	"io"
	"net/http"
)

type iHTTPClient interface {
	Request(method string, url string, body io.Reader) (*http.Response, error)
}

type httpClient struct {
	client *http.Client
}

func (c *httpClient) instance() http.Client {
	if c.client == nil {
		c.client = &http.Client{}
	}
	return *c.client
}

func (c *httpClient) Request(method string, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	client := c.instance()
	return client.Do(req)
}
