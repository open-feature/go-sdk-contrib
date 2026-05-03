package vercel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/open-feature/go-sdk/openfeature"
)

var _ openfeature.FeatureProvider = (*Provider)(nil)
var _ openfeature.ContextAwareStateHandler = (*Provider)(nil)
var _ openfeature.EventHandler = (*Provider)(nil)

// Provider is an OpenFeature provider backed by Vercel Flags.
type Provider struct {
	options providerOptions
	sdkKey  string

	initMu sync.Mutex
	mu     sync.RWMutex

	data        *Datafile
	initialized bool
	status      openfeature.State

	events     chan openfeature.Event
	pollCancel context.CancelFunc
	pollWG     sync.WaitGroup
}

// NewProvider creates a Vercel Flags OpenFeature provider.
//
// If WithSDKKey or WithConnectionString is not provided, NewProvider reads the
// FLAGS environment variable, matching the TypeScript VercelProvider default.
func NewProvider(opts ...Option) (*Provider, error) {
	options := defaultOptions()
	for _, opt := range opts {
		opt(&options)
	}
	if options.host == "" {
		options.host = defaultHost
	}
	options.host = strings.TrimRight(options.host, "/")
	if options.httpClient == nil {
		options.httpClient = &http.Client{Timeout: defaultRequestTimeout}
	}
	if options.pollingEnabled && options.pollingInterval <= 0 {
		return nil, errors.New("@vercel/flags-core: Polling interval must be greater than 0")
	}

	var sdkKey string
	if options.sdkKeyOrConnectionString != "" {
		var ok bool
		sdkKey, ok = ParseSDKKey(options.sdkKeyOrConnectionString)
		if !ok {
			return nil, errors.New("@vercel/flags-core: Missing sdkKey")
		}
	}

	if sdkKey == "" && options.datafile == nil {
		return nil, errors.New("flags: Missing environment variable FLAGS")
	}
	if sdkKey == "" {
		options.pollingEnabled = false
	}

	provider := &Provider{
		options: options,
		sdkKey:  sdkKey,
		status:  openfeature.NotReadyState,
		events:  make(chan openfeature.Event, 5),
	}
	if options.datafile != nil {
		datafile := *options.datafile
		provider.data = &datafile
	}

	return provider, nil
}

// Metadata returns provider metadata.
func (p *Provider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{Name: providerName}
}

// Hooks returns provider hooks. The Vercel provider does not install hooks.
func (p *Provider) Hooks() []openfeature.Hook {
	return nil
}

// Init implements openfeature.StateHandler.
func (p *Provider) Init(evaluationContext openfeature.EvaluationContext) error {
	return p.InitWithContext(context.Background(), evaluationContext)
}

// InitWithContext implements openfeature.ContextAwareStateHandler.
func (p *Provider) InitWithContext(ctx context.Context, _ openfeature.EvaluationContext) error {
	return p.ensureInitialized(ctx)
}

// Shutdown implements openfeature.StateHandler.
func (p *Provider) Shutdown() {
	_ = p.ShutdownWithContext(context.Background())
}

// ShutdownWithContext implements openfeature.ContextAwareStateHandler.
func (p *Provider) ShutdownWithContext(ctx context.Context) error {
	p.mu.Lock()
	cancel := p.pollCancel
	p.pollCancel = nil
	p.initialized = false
	p.status = openfeature.NotReadyState
	p.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	done := make(chan struct{})
	go func() {
		p.pollWG.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Status returns the provider state.
func (p *Provider) Status() openfeature.State {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

// EventChannel implements openfeature.EventHandler.
func (p *Provider) EventChannel() <-chan openfeature.Event {
	return p.events
}

func (p *Provider) BooleanEvaluation(ctx context.Context, flag string, defaultValue bool, flatCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	result := p.resolve(ctx, flag, defaultValue, flatCtx)
	detail := providerResolutionDetail(result)
	if detail.Reason == openfeature.ErrorReason {
		return openfeature.BoolResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}

	value, ok := result.Value.(bool)
	if !ok {
		return openfeature.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: typeMismatchDetail(
				fmt.Sprintf(`Expected boolean value for flag "%s"`, flag),
			),
		}
	}

	return openfeature.BoolResolutionDetail{Value: value, ProviderResolutionDetail: detail}
}

func (p *Provider) StringEvaluation(ctx context.Context, flag string, defaultValue string, flatCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	result := p.resolve(ctx, flag, defaultValue, flatCtx)
	detail := providerResolutionDetail(result)
	if detail.Reason == openfeature.ErrorReason {
		return openfeature.StringResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}

	value, ok := result.Value.(string)
	if !ok {
		return openfeature.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: typeMismatchDetail(
				fmt.Sprintf(`Expected string value for flag "%s"`, flag),
			),
		}
	}

	return openfeature.StringResolutionDetail{Value: value, ProviderResolutionDetail: detail}
}

func (p *Provider) FloatEvaluation(ctx context.Context, flag string, defaultValue float64, flatCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	result := p.resolve(ctx, flag, defaultValue, flatCtx)
	detail := providerResolutionDetail(result)
	if detail.Reason == openfeature.ErrorReason {
		return openfeature.FloatResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}

	value, ok := numericFloat64(result.Value)
	if !ok {
		return openfeature.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: typeMismatchDetail(
				fmt.Sprintf(`Expected number value for flag "%s"`, flag),
			),
		}
	}

	return openfeature.FloatResolutionDetail{Value: value, ProviderResolutionDetail: detail}
}

func (p *Provider) IntEvaluation(ctx context.Context, flag string, defaultValue int64, flatCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	result := p.resolve(ctx, flag, defaultValue, flatCtx)
	detail := providerResolutionDetail(result)
	if detail.Reason == openfeature.ErrorReason {
		return openfeature.IntResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}

	value, ok := numericInt64(result.Value)
	if !ok {
		return openfeature.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: typeMismatchDetail(
				fmt.Sprintf(`Expected integer value for flag "%s"`, flag),
			),
		}
	}

	return openfeature.IntResolutionDetail{Value: value, ProviderResolutionDetail: detail}
}

func (p *Provider) ObjectEvaluation(ctx context.Context, flag string, defaultValue any, flatCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	result := p.resolve(ctx, flag, defaultValue, flatCtx)
	detail := providerResolutionDetail(result)
	if detail.Reason == openfeature.ErrorReason {
		return openfeature.InterfaceResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}

	return openfeature.InterfaceResolutionDetail{Value: result.Value, ProviderResolutionDetail: detail}
}

func (p *Provider) resolve(ctx context.Context, flag string, defaultValue any, flatCtx openfeature.FlattenedContext) evaluationResult {
	if err := p.ensureInitialized(ctx); err != nil {
		return evaluationResult{
			Value:        defaultValue,
			Reason:       reasonError,
			ErrorMessage: err.Error(),
		}
	}

	p.mu.RLock()
	data := p.data
	p.mu.RUnlock()

	return evaluateDatafile(data, flag, defaultValue, entitiesFromContext(flatCtx))
}

func (p *Provider) ensureInitialized(ctx context.Context) error {
	p.mu.RLock()
	if p.initialized {
		p.mu.RUnlock()
		return nil
	}
	p.mu.RUnlock()

	p.initMu.Lock()
	defer p.initMu.Unlock()

	p.mu.RLock()
	if p.initialized {
		p.mu.RUnlock()
		return nil
	}
	hasData := p.data != nil
	p.mu.RUnlock()

	var data *Datafile
	if !hasData {
		fetched, err := p.fetchDatafile(ctx)
		if err != nil {
			p.setStatus(openfeature.ErrorState)
			p.emit(openfeature.ProviderError, err.Error(), nil)
			return &openfeature.ProviderInitError{
				ErrorCode: openfeature.GeneralCode,
				Message:   err.Error(),
			}
		}
		data = &fetched
	}

	p.mu.Lock()
	if data != nil {
		p.data = data
	}
	p.initialized = true
	p.status = openfeature.ReadyState
	p.startPollingLocked()
	p.mu.Unlock()

	p.emit(openfeature.ProviderReady, "", nil)
	return nil
}

func (p *Provider) fetchDatafile(ctx context.Context) (Datafile, error) {
	if p.sdkKey == "" {
		return Datafile{}, errors.New("@vercel/flags-core: Missing sdkKey")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.options.host+"/v1/datafile", nil)
	if err != nil {
		return Datafile{}, err
	}
	req.Header.Set("Authorization", "Bearer "+p.sdkKey)
	req.Header.Set("User-Agent", "VercelFlagsGo/0.1")
	if vercelEnv := os.Getenv("VERCEL_ENV"); vercelEnv != "" {
		req.Header.Set("X-Vercel-Env", vercelEnv)
	}

	res, err := p.options.httpClient.Do(req)
	if err != nil {
		return Datafile{}, err
	}
	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return Datafile{}, fmt.Errorf("failed to fetch data: %s", res.Status)
	}

	decoder := json.NewDecoder(res.Body)
	decoder.UseNumber()

	var datafile Datafile
	if err := decoder.Decode(&datafile); err != nil {
		return Datafile{}, err
	}
	if datafile.Definitions == nil {
		return Datafile{}, errors.New("@vercel/flags-core: Invalid datafile: missing definitions")
	}
	if datafile.Environment == "" {
		return Datafile{}, errors.New("@vercel/flags-core: Invalid datafile: missing environment")
	}
	return datafile, nil
}

func (p *Provider) startPollingLocked() {
	if !p.options.pollingEnabled || p.sdkKey == "" || p.pollCancel != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.pollCancel = cancel
	p.pollWG.Add(1)
	go p.poll(ctx)
}

func (p *Provider) poll(ctx context.Context) {
	defer p.pollWG.Done()

	ticker := time.NewTicker(p.options.pollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			datafile, err := p.fetchDatafile(ctx)
			if err != nil {
				p.setStatus(openfeature.StaleState)
				p.emit(openfeature.ProviderError, err.Error(), nil)
				continue
			}

			p.setStatus(openfeature.ReadyState)
			if p.replaceDataIfNewer(&datafile) {
				p.emit(openfeature.ProviderConfigChange, "", nil)
			}
		}
	}
}

func (p *Provider) replaceDataIfNewer(data *Datafile) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !isNewerData(p.data, data) {
		return false
	}
	p.data = data
	return true
}

func isNewerData(current, incoming *Datafile) bool {
	if current == nil {
		return true
	}
	if incoming == nil {
		return false
	}
	if current.Revision != 0 || incoming.Revision != 0 {
		return incoming.Revision > current.Revision
	}
	if current.Digest != "" && incoming.Digest != "" {
		return current.Digest != incoming.Digest
	}

	currentTime, currentOK := toFloat64(current.ConfigUpdatedAt)
	incomingTime, incomingOK := toFloat64(incoming.ConfigUpdatedAt)
	if currentOK && incomingOK {
		return incomingTime > currentTime
	}

	return true
}

func (p *Provider) setStatus(status openfeature.State) {
	p.mu.Lock()
	p.status = status
	p.mu.Unlock()
}

func (p *Provider) emit(eventType openfeature.EventType, message string, metadata map[string]any) {
	event := openfeature.Event{
		ProviderName: providerName,
		EventType:    eventType,
		ProviderEventDetails: openfeature.ProviderEventDetails{
			Message:       message,
			EventMetadata: metadata,
		},
	}

	select {
	case p.events <- event:
	default:
	}
}

func entitiesFromContext(flatCtx openfeature.FlattenedContext) map[string]any {
	entities := make(map[string]any, len(flatCtx))
	for key, value := range flatCtx {
		entities[key] = value
	}
	return entities
}

func providerResolutionDetail(result evaluationResult) openfeature.ProviderResolutionDetail {
	if result.Reason == reasonError {
		return openfeature.ProviderResolutionDetail{
			Reason:          openfeature.ErrorReason,
			ResolutionError: resolutionError(result),
		}
	}

	return openfeature.ProviderResolutionDetail{
		Reason:  mapReason(result.Reason),
		Variant: result.Variant,
	}
}

func mapReason(reason vercelReason) openfeature.Reason {
	switch reason {
	case reasonPaused:
		return openfeature.StaticReason
	case reasonFallthrough:
		return openfeature.DefaultReason
	case reasonTargetMatch, reasonRuleMatch:
		return openfeature.TargetingMatchReason
	case reasonError:
		return openfeature.ErrorReason
	default:
		return openfeature.UnknownReason
	}
}

func resolutionError(result evaluationResult) openfeature.ResolutionError {
	switch result.ErrorCode {
	case "FLAG_NOT_FOUND":
		return openfeature.NewFlagNotFoundResolutionError(result.ErrorMessage)
	default:
		return openfeature.NewGeneralResolutionError(result.ErrorMessage)
	}
}

func typeMismatchDetail(message string) openfeature.ProviderResolutionDetail {
	return openfeature.ProviderResolutionDetail{
		Reason:          openfeature.ErrorReason,
		ResolutionError: openfeature.NewTypeMismatchResolutionError(message),
	}
}

func numericFloat64(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		f, err := typed.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func numericInt64(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int64:
		return typed, true
	case float64:
		if typed == float64(int64(typed)) {
			return int64(typed), true
		}
	case json.Number:
		if i, err := typed.Int64(); err == nil {
			return i, true
		}
		if f, err := typed.Float64(); err == nil && f == float64(int64(f)) {
			return int64(f), true
		}
	}
	return 0, false
}
