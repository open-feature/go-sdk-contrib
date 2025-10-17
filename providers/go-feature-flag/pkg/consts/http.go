package consts

import (
	"net/http"
	"time"
)

const (
	ContentTypeHeader   = "Content-Type"
	IfNoneMatchHeader   = "If-None-Match"
	AuthorizationHeader = "Authorization"
	ApplicationJson     = "application/json"
	BearerPrefix        = "Bearer "
)

var DefaultHTTPClient = func() *http.Client {
	netTransport := &http.Transport{
		TLSHandshakeTimeout: 10000 * time.Millisecond,
		IdleConnTimeout:     90 * time.Second,
	}

	return &http.Client{
		Timeout:   10000 * time.Millisecond,
		Transport: netTransport,
	}
}()
