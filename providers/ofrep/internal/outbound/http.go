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
	BaseURI   string
	Callbacks []HeaderCallback
	Client    *http.Client
}

// Outbound client for http communication
type Outbound struct {
	baseURI        string
	client         *http.Client
	headerProvider []HeaderCallback
}

func NewHttp(cfg Configuration) *Outbound {
	if cfg.Client == nil {
		cfg.Client = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	return &Outbound{
		headerProvider: cfg.Callbacks,
		baseURI:        cfg.BaseURI,
		client:         cfg.Client,
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
