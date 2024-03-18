package outbound

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	of "github.com/open-feature/go-sdk/openfeature"
)

const ofrepV1 = "/ofrep/v1/evaluate/flags/"

// HeaderCallback is a callback returning header name and header value
type HeaderCallback func() (name string, value string)

type Configuration struct {
	Callbacks []HeaderCallback
	BaseURI   string
}

// Outbound client for http communication
type Outbound struct {
	headerProvider []HeaderCallback
	baseURI        string

	client http.Client
}

func NewHttp(cfg Configuration) *Outbound {
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	return &Outbound{
		headerProvider: cfg.Callbacks,
		baseURI:        cfg.BaseURI,
		client:         client,
	}
}

func (h *Outbound) PostSingle(ctx context.Context, key string, payload []byte) (*http.Response, error) {
	path, err := url.JoinPath(h.baseURI, ofrepV1, key)
	if err != nil {
		return nil, fmt.Errorf("error building request path: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, bytes.NewReader(payload))
	if err != nil {
		resErr := of.NewGeneralResolutionError(fmt.Sprintf("request building error: %v", err))
		return nil, &resErr
	}

	for _, callback := range h.headerProvider {
		req.Header.Set(callback())
	}

	return h.client.Do(req)
}
