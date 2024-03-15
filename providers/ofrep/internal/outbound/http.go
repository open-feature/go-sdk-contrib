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

type AuthCallback func() (key string, value string)

// Outbound client for http communication
type Outbound struct {
	auth    AuthCallback
	baseURI string

	client http.Client
}

func NewOutbound(baseUri string, callback AuthCallback) *Outbound {
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	return &Outbound{
		baseURI: baseUri,
		client:  client,
		auth:    callback,
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

	if h.auth != nil {
		// set authentication headers
		req.Header.Set(h.auth())
	}

	return h.client.Do(req)
}
