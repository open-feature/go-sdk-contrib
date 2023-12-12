package rpc

import (
	"context"
	"strings"
	"testing"

	schemaConnectV1 "buf.build/gen/go/open-feature/flagd/connectrpc/go/schema/v1/schemav1connect"
	v1 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/schema/v1"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/internal/cache"
	of "github.com/open-feature/go-sdk/openfeature"
	"google.golang.org/protobuf/types/known/structpb"
)

// Service tests for flag evaluation

var flagKey = "key"
var metadata map[string]interface{}
var metadataStruct *structpb.Struct
var log logr.Logger

type responseType interface {
	of.BoolResolutionDetail | of.StringResolutionDetail | of.FloatResolutionDetail | of.IntResolutionDetail | of.InterfaceResolutionDetail
}

type testStruct[T responseType] struct {
	name           string
	getCache       func() *cache.Service
	getMockClient  func() schemaConnectV1.ServiceClient
	expectResponse T
	isCached       bool
	errorText      string
}

func init() {
	metadata = map[string]interface{}{
		"scope": "flagd-scope",
	}

	var err error
	metadataStruct, err = structpb.NewStruct(metadata)
	if err != nil {
		panic(err)
	}
}

func TestBooleanEvaluation(t *testing.T) {
	defaultValue := false

	tests := []testStruct[of.BoolResolutionDetail]{
		{
			name: "happy path - simple uncached evaluation",
			getCache: func() *cache.Service {
				// disable cache
				return cache.NewCacheService(cache.DisabledValue, 10, log)
			},
			getMockClient: func() schemaConnectV1.ServiceClient {
				return &MockClient{
					booleanResponse: v1.ResolveBooleanResponse{
						Value:    true,
						Reason:   string(of.StaticReason),
						Variant:  "on",
						Metadata: metadataStruct,
					},
				}
			},
			expectResponse: of.BoolResolutionDetail{
				Value: true,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:       of.StaticReason,
					Variant:      "on",
					FlagMetadata: metadata,
				},
			},
			isCached: false,
		},
		{
			name: "cached flags are served with cache reason",
			getCache: func() *cache.Service {
				cacheService := cache.NewCacheService(cache.InMemValue, 10, log)

				cacheService.GetCache().Add(flagKey, of.BoolResolutionDetail{
					Value: true,
					ProviderResolutionDetail: of.ProviderResolutionDetail{
						Reason:       of.StaticReason,
						Variant:      "on",
						FlagMetadata: metadata,
					},
				})

				return cacheService
			},
			getMockClient: func() schemaConnectV1.ServiceClient {
				// empty mock
				return &MockClient{}
			},
			expectResponse: of.BoolResolutionDetail{
				Value: true,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:       of.CachedReason,
					Variant:      "on",
					FlagMetadata: metadata,
				},
			},
			isCached: true,
		},
		{
			name: "static resolving will be cached",
			getCache: func() *cache.Service {
				return cache.NewCacheService(cache.InMemValue, 10, log)
			},
			getMockClient: func() schemaConnectV1.ServiceClient {
				return &MockClient{
					booleanResponse: v1.ResolveBooleanResponse{
						Value:    true,
						Reason:   string(of.StaticReason),
						Variant:  "on",
						Metadata: metadataStruct,
					},
				}
			},
			expectResponse: of.BoolResolutionDetail{
				Value: true,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:       of.StaticReason,
					Variant:      "on",
					FlagMetadata: metadata,
				},
			},
			isCached: true,
		},
		{
			name: "simple error check - flag not found",
			getCache: func() *cache.Service {
				return cache.NewCacheService(cache.DisabledValue, 0, log)
			},
			getMockClient: func() schemaConnectV1.ServiceClient {
				return &MockClient{
					error: of.NewFlagNotFoundResolutionError("requested flag not found"),
				}
			},
			expectResponse: of.BoolResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewFlagNotFoundResolutionError("requested flag not found"),
				},
			},
			isCached:  false,
			errorText: string(of.FlagNotFoundCode),
		},
	}

	for _, test := range tests {
		service := Service{
			cache:  test.getCache(),
			logger: log,
			client: test.getMockClient(),
		}

		resolutionDetail := service.ResolveBoolean(context.Background(), flagKey, defaultValue, map[string]interface{}{})
		validate(t, test, resolutionDetail, resolutionDetail.ResolutionError, service)
	}
}

func TestStringEvaluation(t *testing.T) {
	defaultValue := "other"

	tests := []testStruct[of.StringResolutionDetail]{
		{
			name: "happy path - simple uncached evaluation",
			getCache: func() *cache.Service {
				// disable cache
				return cache.NewCacheService(cache.DisabledValue, 10, log)
			},
			getMockClient: func() schemaConnectV1.ServiceClient {
				return &MockClient{
					stringResponse: v1.ResolveStringResponse{
						Value:    "valid",
						Reason:   string(of.StaticReason),
						Variant:  "on",
						Metadata: metadataStruct,
					},
				}
			},
			expectResponse: of.StringResolutionDetail{
				Value: "valid",
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:       of.StaticReason,
					Variant:      "on",
					FlagMetadata: metadata,
				},
			},
			isCached: false,
		},
		{
			name: "cached flags are served with cache reason",
			getCache: func() *cache.Service {
				cacheService := cache.NewCacheService(cache.InMemValue, 10, log)

				cacheService.GetCache().Add(flagKey, of.StringResolutionDetail{
					Value: "valid",
					ProviderResolutionDetail: of.ProviderResolutionDetail{
						Reason:       of.StaticReason,
						Variant:      "on",
						FlagMetadata: metadata,
					},
				})

				return cacheService
			},
			getMockClient: func() schemaConnectV1.ServiceClient {
				// empty mock
				return &MockClient{}
			},
			expectResponse: of.StringResolutionDetail{
				Value: "valid",
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:       of.CachedReason,
					Variant:      "on",
					FlagMetadata: metadata,
				},
			},
			isCached: true,
		},
		{
			name: "static resolving will be cached",
			getCache: func() *cache.Service {
				return cache.NewCacheService(cache.InMemValue, 10, log)
			},
			getMockClient: func() schemaConnectV1.ServiceClient {
				return &MockClient{
					stringResponse: v1.ResolveStringResponse{
						Value:    "valid",
						Reason:   string(of.StaticReason),
						Variant:  "on",
						Metadata: metadataStruct,
					},
				}
			},
			expectResponse: of.StringResolutionDetail{
				Value: "valid",
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:       of.StaticReason,
					Variant:      "on",
					FlagMetadata: metadata,
				},
			},
			isCached: true,
		},
		{
			name: "simple error check - flag not found",
			getCache: func() *cache.Service {
				return cache.NewCacheService(cache.DisabledValue, 0, log)
			},
			getMockClient: func() schemaConnectV1.ServiceClient {
				return &MockClient{
					error: of.NewFlagNotFoundResolutionError("requested flag not found"),
				}
			},
			expectResponse: of.StringResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewFlagNotFoundResolutionError("requested flag not found"),
				},
			},
			isCached:  false,
			errorText: string(of.FlagNotFoundCode),
		},
	}

	for _, test := range tests {
		service := Service{
			cache:  test.getCache(),
			logger: log,
			client: test.getMockClient(),
		}

		resolutionDetail := service.ResolveString(context.Background(), flagKey, defaultValue, map[string]interface{}{})
		validate(t, test, resolutionDetail, resolutionDetail.ResolutionError, service)
	}
}

func TestFloatEvaluation(t *testing.T) {
	defaultValue := 0.05

	tests := []testStruct[of.FloatResolutionDetail]{
		{
			name: "happy path - simple uncached evaluation",
			getCache: func() *cache.Service {
				// disable cache
				return cache.NewCacheService(cache.DisabledValue, 10, log)
			},
			getMockClient: func() schemaConnectV1.ServiceClient {
				return &MockClient{
					floatResponse: v1.ResolveFloatResponse{
						Value:    1.005,
						Reason:   string(of.StaticReason),
						Variant:  "on",
						Metadata: metadataStruct,
					},
				}
			},
			expectResponse: of.FloatResolutionDetail{
				Value: 1.005,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:       of.StaticReason,
					Variant:      "on",
					FlagMetadata: metadata,
				},
			},
			isCached: false,
		},
		{
			name: "cached flags are served with cache reason",
			getCache: func() *cache.Service {
				cacheService := cache.NewCacheService(cache.InMemValue, 10, log)

				cacheService.GetCache().Add(flagKey, of.FloatResolutionDetail{
					Value: 1.005,
					ProviderResolutionDetail: of.ProviderResolutionDetail{
						Reason:       of.StaticReason,
						Variant:      "on",
						FlagMetadata: metadata,
					},
				})

				return cacheService
			},
			getMockClient: func() schemaConnectV1.ServiceClient {
				// empty mock
				return &MockClient{}
			},
			expectResponse: of.FloatResolutionDetail{
				Value: 1.005,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:       of.CachedReason,
					Variant:      "on",
					FlagMetadata: metadata,
				},
			},
			isCached: true,
		},
		{
			name: "static resolving will be cached",
			getCache: func() *cache.Service {
				return cache.NewCacheService(cache.InMemValue, 10, log)
			},
			getMockClient: func() schemaConnectV1.ServiceClient {
				return &MockClient{
					floatResponse: v1.ResolveFloatResponse{
						Value:    1.005,
						Reason:   string(of.StaticReason),
						Variant:  "on",
						Metadata: metadataStruct,
					},
				}
			},
			expectResponse: of.FloatResolutionDetail{
				Value: 1.005,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:       of.StaticReason,
					Variant:      "on",
					FlagMetadata: metadata,
				},
			},
			isCached: true,
		},
		{
			name: "simple error check - flag not found",
			getCache: func() *cache.Service {
				return cache.NewCacheService(cache.DisabledValue, 0, log)
			},
			getMockClient: func() schemaConnectV1.ServiceClient {
				return &MockClient{
					error: of.NewFlagNotFoundResolutionError("requested flag not found"),
				}
			},
			expectResponse: of.FloatResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewFlagNotFoundResolutionError("requested flag not found"),
				},
			},
			isCached:  false,
			errorText: string(of.FlagNotFoundCode),
		},
	}

	for _, test := range tests {
		service := Service{
			cache:  test.getCache(),
			logger: log,
			client: test.getMockClient(),
		}

		resolutionDetail := service.ResolveFloat(context.Background(), flagKey, defaultValue, map[string]interface{}{})
		validate(t, test, resolutionDetail, resolutionDetail.ResolutionError, service)
	}
}

func TestIntEvaluation(t *testing.T) {
	defaultValue := 1

	tests := []testStruct[of.IntResolutionDetail]{
		{
			name: "happy path - simple uncached evaluation",
			getCache: func() *cache.Service {
				// disable cache
				return cache.NewCacheService(cache.DisabledValue, 10, log)
			},
			getMockClient: func() schemaConnectV1.ServiceClient {
				return &MockClient{
					intResponse: v1.ResolveIntResponse{
						Value:    2,
						Reason:   string(of.StaticReason),
						Variant:  "on",
						Metadata: metadataStruct,
					},
				}
			},
			expectResponse: of.IntResolutionDetail{
				Value: 2,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:       of.StaticReason,
					Variant:      "on",
					FlagMetadata: metadata,
				},
			},
			isCached: false,
		},
		{
			name: "cached flags are served with cache reason",
			getCache: func() *cache.Service {
				cacheService := cache.NewCacheService(cache.InMemValue, 10, log)

				cacheService.GetCache().Add(flagKey, of.IntResolutionDetail{
					Value: 2,
					ProviderResolutionDetail: of.ProviderResolutionDetail{
						Reason:       of.StaticReason,
						Variant:      "on",
						FlagMetadata: metadata,
					},
				})

				return cacheService
			},
			getMockClient: func() schemaConnectV1.ServiceClient {
				// empty mock
				return &MockClient{}
			},
			expectResponse: of.IntResolutionDetail{
				Value: 2,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:       of.CachedReason,
					Variant:      "on",
					FlagMetadata: metadata,
				},
			},
			isCached: true,
		},
		{
			name: "static resolving will be cached",
			getCache: func() *cache.Service {
				return cache.NewCacheService(cache.InMemValue, 10, log)
			},
			getMockClient: func() schemaConnectV1.ServiceClient {
				return &MockClient{
					intResponse: v1.ResolveIntResponse{
						Value:    2,
						Reason:   string(of.StaticReason),
						Variant:  "on",
						Metadata: metadataStruct,
					},
				}
			},
			expectResponse: of.IntResolutionDetail{
				Value: 2,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:       of.StaticReason,
					Variant:      "on",
					FlagMetadata: metadata,
				},
			},
			isCached: true,
		},
		{
			name: "simple error check - flag not found",
			getCache: func() *cache.Service {
				return cache.NewCacheService(cache.DisabledValue, 0, log)
			},
			getMockClient: func() schemaConnectV1.ServiceClient {
				return &MockClient{
					error: of.NewFlagNotFoundResolutionError("requested flag not found"),
				}
			},
			expectResponse: of.IntResolutionDetail{
				Value: int64(defaultValue),
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewFlagNotFoundResolutionError("requested flag not found"),
				},
			},
			isCached:  false,
			errorText: string(of.FlagNotFoundCode),
		},
	}

	for _, test := range tests {
		service := Service{
			cache:  test.getCache(),
			logger: log,
			client: test.getMockClient(),
		}

		resolutionDetail := service.ResolveInt(context.Background(), flagKey, int64(defaultValue), map[string]interface{}{})
		validate(t, test, resolutionDetail, resolutionDetail.ResolutionError, service)
	}
}

func TestObjectEvaluation(t *testing.T) {
	defaultValue := map[string]interface{}{
		"f1": "zero",
		"f2": "0",
	}

	expectedValue := map[string]interface{}{
		"f1": "one",
		"f2": "1",
	}

	expectedValueAsStruct, err := structpb.NewStruct(expectedValue)
	if err != nil {
		t.Fatal(err)
	}

	tests := []testStruct[of.InterfaceResolutionDetail]{
		{
			name: "happy path - simple uncached evaluation",
			getCache: func() *cache.Service {
				// disable cache
				return cache.NewCacheService(cache.DisabledValue, 10, log)
			},
			getMockClient: func() schemaConnectV1.ServiceClient {
				return &MockClient{
					objResponse: v1.ResolveObjectResponse{
						Value:    expectedValueAsStruct,
						Reason:   string(of.StaticReason),
						Variant:  "on",
						Metadata: metadataStruct,
					},
				}
			},
			expectResponse: of.InterfaceResolutionDetail{
				Value: expectedValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:       of.StaticReason,
					Variant:      "on",
					FlagMetadata: metadata,
				},
			},
			isCached: false,
		},
		{
			name: "cached flags are served with cache reason",
			getCache: func() *cache.Service {
				cacheService := cache.NewCacheService(cache.InMemValue, 10, log)

				cacheService.GetCache().Add(flagKey, of.InterfaceResolutionDetail{
					Value: expectedValue,
					ProviderResolutionDetail: of.ProviderResolutionDetail{
						Reason:       of.StaticReason,
						Variant:      "on",
						FlagMetadata: metadata,
					},
				})

				return cacheService
			},
			getMockClient: func() schemaConnectV1.ServiceClient {
				// empty mock
				return &MockClient{}
			},
			expectResponse: of.InterfaceResolutionDetail{
				Value: expectedValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:       of.CachedReason,
					Variant:      "on",
					FlagMetadata: metadata,
				},
			},
			isCached: true,
		},
		{
			name: "static resolving will be cached",
			getCache: func() *cache.Service {
				return cache.NewCacheService(cache.InMemValue, 10, log)
			},
			getMockClient: func() schemaConnectV1.ServiceClient {
				return &MockClient{
					objResponse: v1.ResolveObjectResponse{
						Value:    expectedValueAsStruct,
						Reason:   string(of.StaticReason),
						Variant:  "on",
						Metadata: metadataStruct,
					},
				}
			},
			expectResponse: of.InterfaceResolutionDetail{
				Value: expectedValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					Reason:       of.StaticReason,
					Variant:      "on",
					FlagMetadata: metadata,
				},
			},
			isCached: true,
		},
		{
			name: "simple error check - flag not found",
			getCache: func() *cache.Service {
				return cache.NewCacheService(cache.DisabledValue, 0, log)
			},
			getMockClient: func() schemaConnectV1.ServiceClient {
				return &MockClient{
					error: of.NewFlagNotFoundResolutionError("requested flag not found"),
				}
			},
			expectResponse: of.InterfaceResolutionDetail{
				Value: defaultValue,
				ProviderResolutionDetail: of.ProviderResolutionDetail{
					ResolutionError: of.NewFlagNotFoundResolutionError("requested flag not found"),
				},
			},
			isCached:  false,
			errorText: string(of.FlagNotFoundCode),
		},
	}

	for _, test := range tests {
		service := Service{
			cache:  test.getCache(),
			logger: log,
			client: test.getMockClient(),
		}

		resolutionDetail := service.ResolveObject(context.Background(), flagKey, defaultValue, map[string]interface{}{})
		validate(t, test, resolutionDetail, resolutionDetail.ResolutionError, service)
	}
}

// validate is a generic validator
func validate[T responseType](t *testing.T, test testStruct[T], resolutionDetail T, error of.ResolutionError, service Service) {
	if diff := cmp.Diff(
		test.expectResponse, resolutionDetail,
		cmpopts.IgnoreFields(of.ProviderResolutionDetail{}, "ResolutionError"),
	); diff != "" {
		t.Errorf("test %s: mismatch (-expected +got):\n%s", test.name, diff)
	}

	// check cache for stored key
	if test.isCached {
		_, ok := service.cache.GetCache().Get(flagKey)
		if !ok {
			t.Errorf("test %s: expected flag to be cached", test.name)
		}
	}

	if test.errorText != "" && strings.Contains(test.errorText, error.Error()) {
		t.Errorf("test %s: expected error to contain %s, but error was %s",
			test.name, test.errorText, error.Error())
	}
}
