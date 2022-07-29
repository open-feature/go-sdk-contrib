package http_service

import (
	"io"
	"net/http"
)

type IHTTPClient interface {
	Request(method string, url string, body io.Reader) (*http.Response, error)
}

type HTTPClient struct {
	client *http.Client
}

func (c *HTTPClient) GetInstance() http.Client {
	if c.client == nil {
		c.client = &http.Client{}
	}
	return *c.client
}

func (c *HTTPClient) Request(method string, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	client := c.GetInstance()
	return client.Do(req)
}
