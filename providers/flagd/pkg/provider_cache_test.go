package flagd

import (
	schemav1 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/schema/v1"
	"context"
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	flagdModels "github.com/open-feature/flagd/pkg/model"
	flagdService "github.com/open-feature/flagd/pkg/service"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/logger"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/mock"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/constant"
	of "github.com/open-feature/go-sdk/pkg/openfeature"
	"google.golang.org/protobuf/types/known/structpb"
	"testing"
)

func TestBooleanEvaluationCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := map[string]struct {
		flagKey string
		mockOut *schemav1.ResolveBooleanResponse
		setup   func(
			t *testing.T, ctx context.Context, provider Provider, mockSvc *mock.MockIService,
			flagKey string, mockOut *schemav1.ResolveBooleanResponse,
		)
		expectedRes of.BoolResolutionDetail
	}{
		"cache when static": {
			flagKey: "foo",
			mockOut: &schemav1.ResolveBooleanResponse{
				Value:   true,
				Variant: "on",
				Reason:  flagdModels.StaticReason,
			},
			setup: func(
				t *testing.T, ctx context.Context, provider Provider, mockSvc *mock.MockIService,
				flagKey string, mockOut *schemav1.ResolveBooleanResponse,
			) {
				mockSvc.EXPECT().ResolveBoolean(gomock.Any(), flagKey, gomock.Any()).Return(mockOut, nil)
				mockSvc.EXPECT().IsEventStreamAlive().Return(true).AnyTimes()
				provider.BooleanEvaluation(ctx, flagKey, false, of.FlattenedContext{}) // store flag in cache
			},
			expectedRes: of.BoolResolutionDetail{
				Value: true,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:  constant.ReasonCached,
					Variant: "on",
				},
			},
		},
		"don't cache when not static": {
			flagKey: "foo",
			mockOut: &schemav1.ResolveBooleanResponse{
				Value:   true,
				Variant: "on",
				Reason:  flagdModels.DefaultReason,
			},
			setup: func(
				t *testing.T, ctx context.Context, provider Provider, mockSvc *mock.MockIService,
				flagKey string, mockOut *schemav1.ResolveBooleanResponse,
			) {
				mockSvc.EXPECT().ResolveBoolean(gomock.Any(), flagKey, gomock.Any()).Return(mockOut, nil).Times(2)
				mockSvc.EXPECT().IsEventStreamAlive().Return(true).AnyTimes()
				provider.BooleanEvaluation(ctx, flagKey, false, of.FlattenedContext{}) // shouldn't store flag in cache
			},
			expectedRes: of.BoolResolutionDetail{
				Value: true,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:  flagdModels.DefaultReason,
					Variant: "on",
				},
			},
		},
		"don't cache when event stream isn't alive": {
			flagKey: "foo",
			mockOut: &schemav1.ResolveBooleanResponse{
				Value:   true,
				Variant: "on",
				Reason:  flagdModels.StaticReason,
			},
			setup: func(
				t *testing.T, ctx context.Context, provider Provider, mockSvc *mock.MockIService,
				flagKey string, mockOut *schemav1.ResolveBooleanResponse,
			) {
				mockSvc.EXPECT().ResolveBoolean(gomock.Any(), flagKey, gomock.Any()).Return(mockOut, nil).Times(2)
				mockSvc.EXPECT().IsEventStreamAlive().Return(false).AnyTimes()
				provider.BooleanEvaluation(ctx, flagKey, false, of.FlattenedContext{}) // shouldn't store flag in cache
			},
			expectedRes: of.BoolResolutionDetail{
				Value: true,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:  flagdModels.StaticReason,
					Variant: "on",
				},
			},
		},
	}

	cacheImplementations := []struct {
		name  string
		apply func(provider *Provider)
	}{
		{
			name: "in memory",
			apply: func(provider *Provider) {
				WithBasicInMemoryCache()(provider)
			},
		},
		{
			name: "lru",
			apply: func(provider *Provider) {
				WithLRUCache(100)(provider)
			},
		},
	}

	for _, cacheImplementation := range cacheImplementations {
		t.Run(cacheImplementation.name, func(t *testing.T) {
			for name, tt := range tests {
				t.Run(name, func(t *testing.T) {
					ctx := context.Background()
					mockSvc := mock.NewMockIService(ctrl)

					provider := Provider{
						service: mockSvc,
					}
					cacheImplementation.apply(&provider)

					tt.setup(t, ctx, provider, mockSvc, tt.flagKey, tt.mockOut)

					got := provider.BooleanEvaluation(ctx, tt.flagKey, false, of.FlattenedContext{})

					if diff := cmp.Diff(
						tt.expectedRes, got,
						cmpopts.IgnoreFields(of.ProviderResolutionDetail{}, "ResolutionError"),
					); diff != "" {
						t.Errorf("mismatch (-expected +got):\n%s", diff)
					}
				})
			}
		})
	}
}

func TestStringEvaluationCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := map[string]struct {
		flagKey string
		mockOut *schemav1.ResolveStringResponse
		setup   func(
			t *testing.T, ctx context.Context, provider Provider, mockSvc *mock.MockIService,
			flagKey string, mockOut *schemav1.ResolveStringResponse,
		)
		expectedRes of.StringResolutionDetail
	}{
		"cache when static": {
			flagKey: "foo",
			mockOut: &schemav1.ResolveStringResponse{
				Value:   "bar",
				Variant: "on",
				Reason:  flagdModels.StaticReason,
			},
			setup: func(
				t *testing.T, ctx context.Context, provider Provider, mockSvc *mock.MockIService,
				flagKey string, mockOut *schemav1.ResolveStringResponse,
			) {
				mockSvc.EXPECT().ResolveString(gomock.Any(), flagKey, gomock.Any()).Return(mockOut, nil)
				mockSvc.EXPECT().IsEventStreamAlive().Return(true).AnyTimes()
				provider.StringEvaluation(ctx, flagKey, "", of.FlattenedContext{}) // store flag in cache
			},
			expectedRes: of.StringResolutionDetail{
				Value: "bar",
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:  constant.ReasonCached,
					Variant: "on",
				},
			},
		},
		"don't cache when not static": {
			flagKey: "foo",
			mockOut: &schemav1.ResolveStringResponse{
				Value:   "bar",
				Variant: "on",
				Reason:  flagdModels.DefaultReason,
			},
			setup: func(
				t *testing.T, ctx context.Context, provider Provider, mockSvc *mock.MockIService,
				flagKey string, mockOut *schemav1.ResolveStringResponse,
			) {
				mockSvc.EXPECT().ResolveString(gomock.Any(), flagKey, gomock.Any()).Return(mockOut, nil).Times(2)
				mockSvc.EXPECT().IsEventStreamAlive().Return(true).AnyTimes()
				provider.StringEvaluation(ctx, flagKey, "", of.FlattenedContext{}) // shouldn't store flag in cache
			},
			expectedRes: of.StringResolutionDetail{
				Value: "bar",
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:  flagdModels.DefaultReason,
					Variant: "on",
				},
			},
		},
		"don't cache when event stream isn't alive": {
			flagKey: "foo",
			mockOut: &schemav1.ResolveStringResponse{
				Value:   "bar",
				Variant: "on",
				Reason:  flagdModels.StaticReason,
			},
			setup: func(
				t *testing.T, ctx context.Context, provider Provider, mockSvc *mock.MockIService,
				flagKey string, mockOut *schemav1.ResolveStringResponse,
			) {
				mockSvc.EXPECT().ResolveString(gomock.Any(), flagKey, gomock.Any()).Return(mockOut, nil).Times(2)
				mockSvc.EXPECT().IsEventStreamAlive().Return(false).AnyTimes()
				provider.StringEvaluation(ctx, flagKey, "", of.FlattenedContext{}) // shouldn't store flag in cache
			},
			expectedRes: of.StringResolutionDetail{
				Value: "bar",
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:  flagdModels.StaticReason,
					Variant: "on",
				},
			},
		},
	}

	cacheImplementations := []struct {
		name  string
		apply func(provider *Provider)
	}{
		{
			name: "in memory",
			apply: func(provider *Provider) {
				WithBasicInMemoryCache()(provider)
			},
		},
		{
			name: "lru",
			apply: func(provider *Provider) {
				WithLRUCache(100)(provider)
			},
		},
	}

	for _, cacheImplementation := range cacheImplementations {
		t.Run(cacheImplementation.name, func(t *testing.T) {
			for name, tt := range tests {
				t.Run(name, func(t *testing.T) {
					ctx := context.Background()
					mockSvc := mock.NewMockIService(ctrl)

					provider := Provider{
						service: mockSvc,
					}
					cacheImplementation.apply(&provider)

					tt.setup(t, ctx, provider, mockSvc, tt.flagKey, tt.mockOut)

					got := provider.StringEvaluation(ctx, tt.flagKey, "", of.FlattenedContext{})

					if diff := cmp.Diff(
						tt.expectedRes, got,
						cmpopts.IgnoreFields(of.ProviderResolutionDetail{}, "ResolutionError"),
					); diff != "" {
						t.Errorf("mismatch (-expected +got):\n%s", diff)
					}
				})
			}
		})
	}
}

func TestFloatEvaluationCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := map[string]struct {
		flagKey string
		mockOut *schemav1.ResolveFloatResponse
		setup   func(
			t *testing.T, ctx context.Context, provider Provider, mockSvc *mock.MockIService,
			flagKey string, mockOut *schemav1.ResolveFloatResponse,
		)
		expectedRes of.FloatResolutionDetail
	}{
		"cache when static": {
			flagKey: "foo",
			mockOut: &schemav1.ResolveFloatResponse{
				Value:   9,
				Variant: "on",
				Reason:  flagdModels.StaticReason,
			},
			setup: func(
				t *testing.T, ctx context.Context, provider Provider, mockSvc *mock.MockIService,
				flagKey string, mockOut *schemav1.ResolveFloatResponse,
			) {
				mockSvc.EXPECT().ResolveFloat(gomock.Any(), flagKey, gomock.Any()).Return(mockOut, nil)
				mockSvc.EXPECT().IsEventStreamAlive().Return(true).AnyTimes()
				provider.FloatEvaluation(ctx, flagKey, 9, of.FlattenedContext{}) // store flag in cache
			},
			expectedRes: of.FloatResolutionDetail{
				Value: 9,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:  constant.ReasonCached,
					Variant: "on",
				},
			},
		},
		"don't cache when not static": {
			flagKey: "foo",
			mockOut: &schemav1.ResolveFloatResponse{
				Value:   9,
				Variant: "on",
				Reason:  flagdModels.DefaultReason,
			},
			setup: func(
				t *testing.T, ctx context.Context, provider Provider, mockSvc *mock.MockIService,
				flagKey string, mockOut *schemav1.ResolveFloatResponse,
			) {
				mockSvc.EXPECT().ResolveFloat(gomock.Any(), flagKey, gomock.Any()).Return(mockOut, nil).Times(2)
				mockSvc.EXPECT().IsEventStreamAlive().Return(true).AnyTimes()
				provider.FloatEvaluation(ctx, flagKey, 9, of.FlattenedContext{}) // shouldn't store flag in cache
			},
			expectedRes: of.FloatResolutionDetail{
				Value: 9,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:  flagdModels.DefaultReason,
					Variant: "on",
				},
			},
		},
		"don't cache when event stream isn't alive": {
			flagKey: "foo",
			mockOut: &schemav1.ResolveFloatResponse{
				Value:   9,
				Variant: "on",
				Reason:  flagdModels.StaticReason,
			},
			setup: func(
				t *testing.T, ctx context.Context, provider Provider, mockSvc *mock.MockIService,
				flagKey string, mockOut *schemav1.ResolveFloatResponse,
			) {
				mockSvc.EXPECT().ResolveFloat(gomock.Any(), flagKey, gomock.Any()).Return(mockOut, nil).Times(2)
				mockSvc.EXPECT().IsEventStreamAlive().Return(false).AnyTimes()
				provider.FloatEvaluation(ctx, flagKey, 9, of.FlattenedContext{}) // shouldn't store flag in cache
			},
			expectedRes: of.FloatResolutionDetail{
				Value: 9,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:  flagdModels.StaticReason,
					Variant: "on",
				},
			},
		},
	}

	cacheImplementations := []struct {
		name  string
		apply func(provider *Provider)
	}{
		{
			name: "in memory",
			apply: func(provider *Provider) {
				WithBasicInMemoryCache()(provider)
			},
		},
		{
			name: "lru",
			apply: func(provider *Provider) {
				WithLRUCache(100)(provider)
			},
		},
	}

	for _, cacheImplementation := range cacheImplementations {
		t.Run(cacheImplementation.name, func(t *testing.T) {
			for name, tt := range tests {
				t.Run(name, func(t *testing.T) {
					ctx := context.Background()
					mockSvc := mock.NewMockIService(ctrl)

					provider := Provider{
						service: mockSvc,
					}
					cacheImplementation.apply(&provider)

					tt.setup(t, ctx, provider, mockSvc, tt.flagKey, tt.mockOut)

					got := provider.FloatEvaluation(ctx, tt.flagKey, 0, of.FlattenedContext{})

					if diff := cmp.Diff(
						tt.expectedRes, got,
						cmpopts.IgnoreFields(of.ProviderResolutionDetail{}, "ResolutionError"),
					); diff != "" {
						t.Errorf("mismatch (-expected +got):\n%s", diff)
					}
				})
			}
		})
	}
}

func TestIntEvaluationCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := map[string]struct {
		flagKey string
		mockOut *schemav1.ResolveIntResponse
		setup   func(
			t *testing.T, ctx context.Context, provider Provider, mockSvc *mock.MockIService,
			flagKey string, mockOut *schemav1.ResolveIntResponse,
		)
		expectedRes of.IntResolutionDetail
	}{
		"cache when static": {
			flagKey: "foo",
			mockOut: &schemav1.ResolveIntResponse{
				Value:   9,
				Variant: "on",
				Reason:  flagdModels.StaticReason,
			},
			setup: func(
				t *testing.T, ctx context.Context, provider Provider, mockSvc *mock.MockIService,
				flagKey string, mockOut *schemav1.ResolveIntResponse,
			) {
				mockSvc.EXPECT().ResolveInt(gomock.Any(), flagKey, gomock.Any()).Return(mockOut, nil)
				mockSvc.EXPECT().IsEventStreamAlive().Return(true).AnyTimes()
				provider.IntEvaluation(ctx, flagKey, 9, of.FlattenedContext{}) // store flag in cache
			},
			expectedRes: of.IntResolutionDetail{
				Value: 9,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:  constant.ReasonCached,
					Variant: "on",
				},
			},
		},
		"don't cache when not static": {
			flagKey: "foo",
			mockOut: &schemav1.ResolveIntResponse{
				Value:   9,
				Variant: "on",
				Reason:  flagdModels.DefaultReason,
			},
			setup: func(
				t *testing.T, ctx context.Context, provider Provider, mockSvc *mock.MockIService,
				flagKey string, mockOut *schemav1.ResolveIntResponse,
			) {
				mockSvc.EXPECT().ResolveInt(gomock.Any(), flagKey, gomock.Any()).Return(mockOut, nil).Times(2)
				mockSvc.EXPECT().IsEventStreamAlive().Return(true).AnyTimes()
				provider.IntEvaluation(ctx, flagKey, 9, of.FlattenedContext{}) // shouldn't store flag in cache
			},
			expectedRes: of.IntResolutionDetail{
				Value: 9,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:  flagdModels.DefaultReason,
					Variant: "on",
				},
			},
		},
		"don't cache when event stream isn't alive": {
			flagKey: "foo",
			mockOut: &schemav1.ResolveIntResponse{
				Value:   9,
				Variant: "on",
				Reason:  flagdModels.StaticReason,
			},
			setup: func(
				t *testing.T, ctx context.Context, provider Provider, mockSvc *mock.MockIService,
				flagKey string, mockOut *schemav1.ResolveIntResponse,
			) {
				mockSvc.EXPECT().ResolveInt(gomock.Any(), flagKey, gomock.Any()).Return(mockOut, nil).Times(2)
				mockSvc.EXPECT().IsEventStreamAlive().Return(false).AnyTimes()
				provider.IntEvaluation(ctx, flagKey, 9, of.FlattenedContext{}) // shouldn't store flag in cache
			},
			expectedRes: of.IntResolutionDetail{
				Value: 9,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:  flagdModels.StaticReason,
					Variant: "on",
				},
			},
		},
	}

	cacheImplementations := []struct {
		name  string
		apply func(provider *Provider)
	}{
		{
			name: "in memory",
			apply: func(provider *Provider) {
				WithBasicInMemoryCache()(provider)
			},
		},
		{
			name: "lru",
			apply: func(provider *Provider) {
				WithLRUCache(100)(provider)
			},
		},
	}

	for _, cacheImplementation := range cacheImplementations {
		t.Run(cacheImplementation.name, func(t *testing.T) {
			for name, tt := range tests {
				t.Run(name, func(t *testing.T) {
					ctx := context.Background()
					mockSvc := mock.NewMockIService(ctrl)

					provider := Provider{
						service: mockSvc,
					}
					cacheImplementation.apply(&provider)

					tt.setup(t, ctx, provider, mockSvc, tt.flagKey, tt.mockOut)

					got := provider.IntEvaluation(ctx, tt.flagKey, 0, of.FlattenedContext{})

					if diff := cmp.Diff(
						tt.expectedRes, got,
						cmpopts.IgnoreFields(of.ProviderResolutionDetail{}, "ResolutionError"),
					); diff != "" {
						t.Errorf("mismatch (-expected +got):\n%s", diff)
					}
				})
			}
		})
	}
}

func TestObjectEvaluationCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := map[string]struct {
		flagKey string
		mockOut *schemav1.ResolveObjectResponse
		setup   func(
			t *testing.T, ctx context.Context, provider Provider, mockSvc *mock.MockIService,
			flagKey string, mockOut *schemav1.ResolveObjectResponse,
		)
		expectedRes of.InterfaceResolutionDetail
	}{
		"cache when static": {
			flagKey: "foo",
			mockOut: &schemav1.ResolveObjectResponse{
				Variant: "on",
				Reason:  flagdModels.StaticReason,
			},
			setup: func(
				t *testing.T, ctx context.Context, provider Provider, mockSvc *mock.MockIService,
				flagKey string, mockOut *schemav1.ResolveObjectResponse,
			) {
				mockSvc.EXPECT().ResolveObject(gomock.Any(), flagKey, gomock.Any()).Return(mockOut, nil)
				mockSvc.EXPECT().IsEventStreamAlive().Return(true).AnyTimes()
				provider.ObjectEvaluation(ctx, flagKey, "", of.FlattenedContext{}) // store flag in cache
			},
			expectedRes: of.InterfaceResolutionDetail{
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:  constant.ReasonCached,
					Variant: "on",
				},
			},
		},
		"don't cache when not static": {
			flagKey: "foo",
			mockOut: &schemav1.ResolveObjectResponse{
				Variant: "on",
				Reason:  flagdModels.DefaultReason,
			},
			setup: func(
				t *testing.T, ctx context.Context, provider Provider, mockSvc *mock.MockIService,
				flagKey string, mockOut *schemav1.ResolveObjectResponse,
			) {
				mockSvc.EXPECT().ResolveObject(gomock.Any(), flagKey, gomock.Any()).Return(mockOut, nil).Times(2)
				mockSvc.EXPECT().IsEventStreamAlive().Return(true).AnyTimes()
				provider.ObjectEvaluation(ctx, flagKey, "", of.FlattenedContext{}) // shouldn't store flag in cache
			},
			expectedRes: of.InterfaceResolutionDetail{
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:  flagdModels.DefaultReason,
					Variant: "on",
				},
			},
		},
		"don't cache when event stream isn't alive": {
			flagKey: "foo",
			mockOut: &schemav1.ResolveObjectResponse{
				Variant: "on",
				Reason:  flagdModels.StaticReason,
			},
			setup: func(
				t *testing.T, ctx context.Context, provider Provider, mockSvc *mock.MockIService,
				flagKey string, mockOut *schemav1.ResolveObjectResponse,
			) {
				mockSvc.EXPECT().ResolveObject(gomock.Any(), flagKey, gomock.Any()).Return(mockOut, nil).Times(2)
				mockSvc.EXPECT().IsEventStreamAlive().Return(false).AnyTimes()
				provider.ObjectEvaluation(ctx, flagKey, "", of.FlattenedContext{}) // shouldn't store flag in cache
			},
			expectedRes: of.InterfaceResolutionDetail{
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:  flagdModels.StaticReason,
					Variant: "on",
				},
			},
		},
	}

	cacheImplementations := []struct {
		name  string
		apply func(provider *Provider)
	}{
		{
			name: "in memory",
			apply: func(provider *Provider) {
				WithBasicInMemoryCache()(provider)
			},
		},
		{
			name: "lru",
			apply: func(provider *Provider) {
				WithLRUCache(100)(provider)
			},
		},
	}

	for _, cacheImplementation := range cacheImplementations {
		t.Run(cacheImplementation.name, func(t *testing.T) {
			for name, tt := range tests {
				t.Run(name, func(t *testing.T) {
					ctx := context.Background()
					mockSvc := mock.NewMockIService(ctrl)

					provider := Provider{
						service: mockSvc,
					}
					cacheImplementation.apply(&provider)

					tt.setup(t, ctx, provider, mockSvc, tt.flagKey, tt.mockOut)

					got := provider.ObjectEvaluation(ctx, tt.flagKey, 0, of.FlattenedContext{})

					if diff := cmp.Diff(
						tt.expectedRes, got,
						cmpopts.IgnoreFields(of.ProviderResolutionDetail{}, "ResolutionError"),
						cmpopts.IgnoreFields(of.InterfaceResolutionDetail{}, "Value"),
					); diff != "" {
						t.Errorf("mismatch (-expected +got):\n%s", diff)
					}
				})
			}
		})
	}
}

func TestCacheInvalidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	emptyEventStreamData, err := structpb.NewStruct(map[string]interface{}{
		"flags": map[string]interface{}{},
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]struct {
		flagKey string
		mockOut *schemav1.ResolveObjectResponse
		setup   func(
			t *testing.T, ctx context.Context, provider *Provider, mockSvc *mock.MockIService,
			flagKey string, mockOut *schemav1.ResolveObjectResponse, ready chan<- struct{},
		)
		expectedRes of.InterfaceResolutionDetail
	}{
		"invalidate cache when flag key is present in configuration_change event": {
			flagKey: "foo",
			mockOut: &schemav1.ResolveObjectResponse{
				Variant: "on",
				Reason:  flagdModels.StaticReason,
			},
			setup: func(
				t *testing.T, ctx context.Context, provider *Provider, mockSvc *mock.MockIService,
				flagKey string, mockOut *schemav1.ResolveObjectResponse, ready chan<- struct{},
			) {
				eventStreamData, err := structpb.NewStruct(map[string]interface{}{
					"flags": map[string]interface{}{
						flagKey: nil,
					},
				})
				if err != nil {
					t.Fatal(err)
				}
				mockSvc.EXPECT().ResolveObject(gomock.Any(), flagKey, gomock.Any()).Return(mockOut, nil).Times(2)
				mockSvc.EXPECT().IsEventStreamAlive().Return(true).AnyTimes()
				mockSvc.EXPECT().EventStream(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(_ context.Context, eventChan chan<- *schemav1.EventStreamResponse, _ int, errChan chan<- error) {

						provider.ObjectEvaluation(ctx, flagKey, "", of.FlattenedContext{}) // store flag in cache
						eventChan <- &schemav1.EventStreamResponse{
							Type: string(flagdService.ConfigurationChange),
							Data: eventStreamData,
						}
						eventChan <- &schemav1.EventStreamResponse{
							Type: string(flagdService.ConfigurationChange),
							Data: emptyEventStreamData,
						} // blocks until previous event has been processed
						ready <- struct{}{}

					})

				go func() {
					if err := provider.handleEvents(provider.ctx); err != nil {
						t.Error(fmt.Errorf("handle events: %w", err))
					}
				}()
			},
			expectedRes: of.InterfaceResolutionDetail{
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:  flagdModels.StaticReason,
					Variant: "on",
				},
			},
		},
		"don't invalidate cache when flag key isn't present in configuration_change event": {
			flagKey: "foo",
			mockOut: &schemav1.ResolveObjectResponse{
				Variant: "on",
				Reason:  flagdModels.StaticReason,
			},
			setup: func(
				t *testing.T, ctx context.Context, provider *Provider, mockSvc *mock.MockIService,
				flagKey string, mockOut *schemav1.ResolveObjectResponse, ready chan<- struct{},
			) {
				eventStreamData, err := structpb.NewStruct(map[string]interface{}{
					"flags": map[string]interface{}{
						"bar": nil,
					},
				})
				if err != nil {
					t.Fatal(err)
				}
				mockSvc.EXPECT().ResolveObject(gomock.Any(), flagKey, gomock.Any()).Return(mockOut, nil)
				mockSvc.EXPECT().IsEventStreamAlive().Return(true).AnyTimes()
				mockSvc.EXPECT().EventStream(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(_ context.Context, eventChan chan<- *schemav1.EventStreamResponse, _ int, errChan chan<- error) {

						provider.ObjectEvaluation(ctx, flagKey, "", of.FlattenedContext{}) // store flag in cache
						eventChan <- &schemav1.EventStreamResponse{
							Type: string(flagdService.ConfigurationChange),
							Data: eventStreamData,
						}
						eventChan <- &schemav1.EventStreamResponse{
							Type: string(flagdService.ConfigurationChange),
							Data: emptyEventStreamData,
						} // blocks until previous event has been processed
						ready <- struct{}{}

					})

				go func() {
					if err := provider.handleEvents(provider.ctx); err != nil {
						t.Error(fmt.Errorf("handle events: %w", err))
					}
				}()
			},
			expectedRes: of.InterfaceResolutionDetail{
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:  constant.ReasonCached,
					Variant: "on",
				},
			},
		},
		"clear cache if malformed configuration_change event": {
			flagKey: "foo",
			mockOut: &schemav1.ResolveObjectResponse{
				Variant: "on",
				Reason:  flagdModels.StaticReason,
			},
			setup: func(
				t *testing.T, ctx context.Context, provider *Provider, mockSvc *mock.MockIService,
				flagKey string, mockOut *schemav1.ResolveObjectResponse, ready chan<- struct{},
			) {
				mockSvc.EXPECT().ResolveObject(gomock.Any(), flagKey, gomock.Any()).Return(mockOut, nil).Times(2)
				mockSvc.EXPECT().IsEventStreamAlive().Return(true).AnyTimes()
				mockSvc.EXPECT().EventStream(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(_ context.Context, eventChan chan<- *schemav1.EventStreamResponse, _ int, errChan chan<- error) {

						provider.ObjectEvaluation(ctx, flagKey, "", of.FlattenedContext{}) // store flag in cache
						eventChan <- &schemav1.EventStreamResponse{
							Type: string(flagdService.ConfigurationChange),
							Data: nil,
						}
						eventChan <- &schemav1.EventStreamResponse{
							Type: string(flagdService.ConfigurationChange),
							Data: emptyEventStreamData,
						} // blocks until previous event has been processed
						ready <- struct{}{}

					})

				go func() {
					if err := provider.handleEvents(provider.ctx); err != nil {
						t.Error(fmt.Errorf("handle events: %w", err))
					}
				}()
			},
			expectedRes: of.InterfaceResolutionDetail{
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:  flagdModels.StaticReason,
					Variant: "on",
				},
			},
		},
		"disable cache if event handler sends error on channel": {
			flagKey: "foo",
			mockOut: &schemav1.ResolveObjectResponse{
				Variant: "on",
				Reason:  flagdModels.StaticReason,
			},
			setup: func(
				t *testing.T, ctx context.Context, provider *Provider, mockSvc *mock.MockIService,
				flagKey string, mockOut *schemav1.ResolveObjectResponse, ready chan<- struct{},
			) {
				mockSvc.EXPECT().ResolveObject(gomock.Any(), flagKey, gomock.Any()).Return(mockOut, nil).Times(2)
				mockSvc.EXPECT().IsEventStreamAlive().Return(true).AnyTimes()
				mockSvc.EXPECT().EventStream(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(_ context.Context, eventChan chan<- *schemav1.EventStreamResponse, _ int, errChan chan<- error) {

						provider.ObjectEvaluation(ctx, flagKey, "", of.FlattenedContext{}) // store flag in cache
						errChan <- errors.New("forced error")

					})

				go func() {
					if err := provider.handleEvents(provider.ctx); err == nil {
						t.Error(errors.New("expected error, got nil"))
					}
					ready <- struct{}{}
				}()
			},
			expectedRes: of.InterfaceResolutionDetail{
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:  flagdModels.StaticReason,
					Variant: "on",
				},
			},
		},
	}

	cacheImplementations := []struct {
		name  string
		apply func(provider *Provider)
	}{
		{
			name: "in memory",
			apply: func(provider *Provider) {
				WithBasicInMemoryCache()(provider)
			},
		},
		{
			name: "lru",
			apply: func(provider *Provider) {
				WithLRUCache(100)(provider)
			},
		},
	}

	for _, cacheImplementation := range cacheImplementations {
		t.Run(cacheImplementation.name, func(t *testing.T) {
			for name, tt := range tests {
				t.Run(name, func(t *testing.T) {
					ctx := context.Background()
					mockSvc := mock.NewMockIService(ctrl)

					provider := &Provider{
						ctx: context.Background(), service: mockSvc, isReady: make(chan struct{}),
						logger: logr.New(logger.Logger{}),
					}
					cacheImplementation.apply(provider)

					readyChan := make(chan struct{})
					tt.setup(t, ctx, provider, mockSvc, tt.flagKey, tt.mockOut, readyChan)

					<-readyChan
					got := provider.ObjectEvaluation(ctx, tt.flagKey, 0, of.FlattenedContext{})

					if diff := cmp.Diff(
						tt.expectedRes, got,
						cmpopts.IgnoreFields(of.ProviderResolutionDetail{}, "ResolutionError"),
						cmpopts.IgnoreFields(of.InterfaceResolutionDetail{}, "Value"),
					); diff != "" {
						t.Errorf("mismatch (-expected +got):\n%s", diff)
					}
				})
			}
		})
	}
}
