package evaluator

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/api"
	"github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg/manager"
	"github.com/open-feature/go-sdk-contrib/providers/ofrep"
	"github.com/open-feature/go-sdk/openfeature"
)

const cacheableMetadataKey = "gofeatureflag_cacheable"

var _ Evaluator = &Remote{}

type Remote struct {
	ofrepProvider   *ofrep.Provider
	cache           *manager.Cache
	api             *api.GoFeatureFlagAPI
	pollingInterval time.Duration
	stopPolling     chan struct{}
	pollingDone     chan struct{}
	shutdownOnce    sync.Once
	etag            string
	mu              sync.Mutex
}

func NewRemoteEvaluator(
	baseUri string,
	httpClient *http.Client,
	apiKey string,
	headers map[string]string,
	flagCacheSize int,
	flagCacheTTL time.Duration,
	disableCache bool,
	pollingInterval time.Duration,
	goffAPI *api.GoFeatureFlagAPI) *Remote {
	ofrepOptions := []ofrep.Option{}
	if httpClient != nil {
		ofrepOptions = append(ofrepOptions, ofrep.WithClient(httpClient))
	}
	if apiKey != "" {
		ofrepOptions = append(ofrepOptions, ofrep.WithApiKeyAuth(apiKey))
	}
	for k, v := range headers {
		ofrepOptions = append(ofrepOptions, ofrep.WithHeader(k, v))
	}

	pollingDone := make(chan struct{})
	close(pollingDone) // pre-close so Shutdown is safe before Init
	return &Remote{
		ofrepProvider:   ofrep.NewProvider(baseUri, ofrepOptions...),
		cache:           manager.NewCache(flagCacheSize, flagCacheTTL, disableCache),
		api:             goffAPI,
		pollingInterval: pollingInterval,
		stopPolling:     make(chan struct{}),
		pollingDone:     pollingDone,
	}
}

func (r *Remote) Init(ctx context.Context) error {
	if !r.cache.IsEnabled() || r.api == nil {
		return nil
	}

	// Get initial ETag baseline
	resp, err := r.api.GetConfiguration(ctx, nil, "")
	if err == nil && resp != nil {
		r.mu.Lock()
		r.etag = resp.Etag
		r.mu.Unlock()
	}

	r.startPolling()
	return nil
}

// startPolling starts a background goroutine that periodically checks for flag
// configuration changes. When a change is detected (new ETag), the cache is purged
// so stale evaluations are not served.
func (r *Remote) startPolling() {
	interval := r.pollingInterval
	if interval <= 0 {
		interval = pollingIntervalDefault
	}
	r.stopPolling = make(chan struct{})
	r.pollingDone = make(chan struct{})
	r.shutdownOnce = sync.Once{}
	go func() {
		defer close(r.pollingDone)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-r.stopPolling:
				return
			case <-ticker.C:
				r.mu.Lock()
				currentEtag := r.etag
				r.mu.Unlock()

				resp, err := r.api.GetConfiguration(context.Background(), nil, currentEtag)
				if errors.Is(err, api.ErrNotModified) {
					continue
				}
				if err != nil || resp == nil {
					continue
				}
				// Config changed — purge stale cache entries
				r.cache.Purge()
				r.mu.Lock()
				r.etag = resp.Etag
				r.mu.Unlock()
			}
		}
	}()
}

func (r *Remote) Shutdown(_ context.Context) error {
	r.shutdownOnce.Do(func() { close(r.stopPolling) })
	<-r.pollingDone
	return nil
}

func (r *Remote) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, flatCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	if cacheValue, err := r.cache.GetBool(flag, flatCtx); err == nil && cacheValue != nil {
		cacheValue.Reason = openfeature.CachedReason
		return *cacheValue
	}
	evalResp := r.ofrepProvider.BooleanEvaluation(ctx, flag, defaultValue, flatCtx)
	if cachable, err := evalResp.FlagMetadata.GetBool(cacheableMetadataKey); err == nil && cachable {
		_ = r.cache.Set(flag, flatCtx, evalResp)
	}
	return evalResp
}

func (r *Remote) StringEvaluation(ctx context.Context, flag string, defaultValue string, flatCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	if cacheValue, err := r.cache.GetString(flag, flatCtx); err == nil && cacheValue != nil {
		cacheValue.Reason = openfeature.CachedReason
		return *cacheValue
	}
	evalResp := r.ofrepProvider.StringEvaluation(ctx, flag, defaultValue, flatCtx)
	if cachable, err := evalResp.FlagMetadata.GetBool(cacheableMetadataKey); err == nil && cachable {
		_ = r.cache.Set(flag, flatCtx, evalResp)
	}
	return evalResp
}

func (r *Remote) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, flatCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	if cacheValue, err := r.cache.GetFloat(flag, flatCtx); err == nil && cacheValue != nil {
		cacheValue.Reason = openfeature.CachedReason
		return *cacheValue
	}
	evalResp := r.ofrepProvider.FloatEvaluation(ctx, flag, defaultValue, flatCtx)
	if cachable, err := evalResp.FlagMetadata.GetBool(cacheableMetadataKey); err == nil && cachable {
		_ = r.cache.Set(flag, flatCtx, evalResp)
	}
	return evalResp
}

func (r *Remote) IntEvaluation(ctx context.Context, flag string, defaultValue int64, flatCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	if cacheValue, err := r.cache.GetInt(flag, flatCtx); err == nil && cacheValue != nil {
		cacheValue.Reason = openfeature.CachedReason
		return *cacheValue
	}
	evalResp := r.ofrepProvider.IntEvaluation(ctx, flag, defaultValue, flatCtx)
	if cachable, err := evalResp.FlagMetadata.GetBool(cacheableMetadataKey); err == nil && cachable {
		_ = r.cache.Set(flag, flatCtx, evalResp)
	}
	return evalResp
}

func (r *Remote) ObjectEvaluation(ctx context.Context, flag string, defaultValue any, flatCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	if cacheValue, err := r.cache.GetInterface(flag, flatCtx); err == nil && cacheValue != nil {
		cacheValue.Reason = openfeature.CachedReason
		return *cacheValue
	}
	evalResp := r.ofrepProvider.ObjectEvaluation(ctx, flag, defaultValue, flatCtx)
	if cachable, err := evalResp.FlagMetadata.GetBool(cacheableMetadataKey); err == nil && cachable {
		_ = r.cache.Set(flag, flatCtx, evalResp)
	}
	return evalResp
}
