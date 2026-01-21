package transport

import (
	"net/http"
	"testing"

	of "github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	offlipt "github.com/open-feature/go-sdk-contrib/providers/flipt/pkg/service"
	flipt "go.flipt.io/flipt/rpc/flipt"
	"go.flipt.io/flipt/rpc/flipt/evaluation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	reqID    = "987654321"
	entityID = "123456789"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		opts     []Option
		expected Service
	}{
		{
			name: "default",
			expected: Service{
				address: "http://localhost:8080",
			},
		},
		{
			name: "with host",
			opts: []Option{WithAddress("foo:9000")},
			expected: Service{
				address: "foo:9000",
			},
		},
		{
			name: "with certificate path",
			opts: []Option{WithCertificatePath("foo")},
			expected: Service{
				address:         "http://localhost:8080",
				certificatePath: "foo",
			},
		},
		{
			name: "with gRPC dial options",
			opts: []Option{WithGRPCDialOptions(grpc.WithUserAgent("Flipt/1.0"))},
			expected: Service{
				address: "http://localhost:8080",
				grpcDialOptions: []grpc.DialOption{
					grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
					grpc.WithUserAgent("Flipt/1.0"),
				},
			},
		},
	}

	//nolint (copylocks)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(tt.opts...)

			assert.NotNil(t, s)
			assert.Equal(t, tt.expected.address, s.address)
		})
	}
}

func TestGetFlag(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectedErr error
		expected    *flipt.Flag
	}{
		{
			name: "success",
			expected: &flipt.Flag{
				Key:          "foo",
				NamespaceKey: "foo-namespace",
			},
		},
		{
			name:        "flag not found",
			err:         status.Error(codes.NotFound, `flag "foo" not found`),
			expectedErr: of.NewFlagNotFoundResolutionError(`flag "foo" not found`),
		},
		{
			name:        "other error",
			err:         status.Error(codes.Internal, "internal error"),
			expectedErr: of.NewGeneralResolutionError("internal error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := offlipt.NewMockClient(t)

			mockClient.On("GetFlag", mock.Anything, &flipt.GetFlagRequest{
				Key:          "foo",
				NamespaceKey: "foo-namespace",
			}).Return(tt.expected, tt.err)

			s := &Service{
				client: mockClient,
			}

			actual, err := s.GetFlag(t.Context(), "foo-namespace", "foo")
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, actual)
			}
		})
	}
}

func TestEvaluate_NonBoolean(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectedErr error
		expected    *evaluation.VariantEvaluationResponse
	}{
		{
			name: "success",
			expected: &evaluation.VariantEvaluationResponse{
				Match:       true,
				SegmentKeys: []string{"foo-segment"},
			},
		},
		{
			name:        "flag not found",
			err:         status.Error(codes.NotFound, `flag "foo" not found`),
			expectedErr: of.NewFlagNotFoundResolutionError(`flag "foo" not found`),
		},
		{
			name:        "other error",
			err:         status.Error(codes.Internal, "internal error"),
			expectedErr: of.NewGeneralResolutionError("internal error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := offlipt.NewMockClient(t)

			mockClient.EXPECT().Variant(mock.Anything, &evaluation.EvaluationRequest{
				FlagKey:      "foo",
				NamespaceKey: "foo-namespace",
				RequestId:    reqID,
				EntityId:     entityID,
				Context: map[string]string{
					"requestID":    reqID,
					"targetingKey": entityID,
				},
			}).Return(tt.expected, tt.err)

			s := &Service{
				client: mockClient,
			}

			evalCtx := map[string]any{
				"requestID":     reqID,
				of.TargetingKey: entityID,
			}

			actual, err := s.Evaluate(t.Context(), "foo-namespace", "foo", evalCtx)
			if tt.expectedErr != nil {
				assert.ErrorContains(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.Match, actual.Match)
				assert.Equal(t, tt.expected.SegmentKeys, actual.SegmentKeys)
			}
		})
	}
}

func TestEvaluate_Boolean(t *testing.T) {
	ber := &evaluation.BooleanEvaluationResponse{
		Enabled: false,
	}

	mockClient := offlipt.NewMockClient(t)

	mockClient.EXPECT().Boolean(mock.Anything, &evaluation.EvaluationRequest{
		FlagKey:      "foo",
		NamespaceKey: "foo-namespace",
		RequestId:    reqID,
		EntityId:     entityID,
		Context: map[string]string{
			"requestID":    reqID,
			"targetingKey": entityID,
		},
	}).Return(ber, nil)

	s := &Service{
		client: mockClient,
	}

	evalCtx := map[string]any{
		"requestID":     reqID,
		of.TargetingKey: entityID,
	}

	actual, err := s.Boolean(t.Context(), "foo-namespace", "foo", evalCtx)
	assert.NoError(t, err)
	assert.False(t, actual.Enabled, "match value should be false")
}

func TestEvaluateInvalidContext(t *testing.T) {
	s := &Service{}

	_, err := s.Evaluate(t.Context(), "foo-namespace", "foo", nil)
	assert.EqualError(t, err, of.NewInvalidContextResolutionError("evalCtx is nil").Error())

	_, err = s.Evaluate(t.Context(), "foo-namespace", "foo", map[string]any{})
	assert.EqualError(t, err, of.NewTargetingKeyMissingResolutionError("targetingKey is missing").Error())
}

func TestLoadTLSCredentials(t *testing.T) {
	tests := []struct {
		name           string
		certificate    string
		expectedErrMsg string
	}{
		{
			name:           "no certificate",
			certificate:    "foo",
			expectedErrMsg: "failed to load certificate: open foo: no such file or directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadTLSCredentials(tt.certificate)

			if tt.expectedErrMsg != "" {
				assert.EqualError(t, err, tt.expectedErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGRPCToOpenFeatureError(t *testing.T) {
	tests := []struct {
		name        string
		grpcStatus  *status.Status
		expectedErr of.ResolutionError
	}{
		{
			name:        "invalid argument",
			grpcStatus:  status.New(codes.InvalidArgument, "invalid argument"),
			expectedErr: of.NewInvalidContextResolutionError("invalid argument"),
		},
		{
			name:        "not found",
			grpcStatus:  status.New(codes.NotFound, "not found"),
			expectedErr: of.NewFlagNotFoundResolutionError("not found"),
		},
		{
			name:        "unavailable",
			grpcStatus:  status.New(codes.Unavailable, "unavailable"),
			expectedErr: of.NewProviderNotReadyResolutionError("unavailable"),
		},
		{
			name:        "unknown",
			grpcStatus:  status.New(codes.Unknown, "unknown"),
			expectedErr: of.NewGeneralResolutionError("unknown"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gRPCToOpenFeatureError(tt.grpcStatus.Err())

			assert.EqualError(t, err, tt.expectedErr.Error())
		})
	}
}

func TestWithHTTPClientOption(t *testing.T) {
	client := &http.Client{}
	p := New(WithHTTPClient(client))
	assert.Equal(t, client, p.httpClient)
}
