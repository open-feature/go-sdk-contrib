package grpc_service

import (
	"errors"
	"testing"

	models "github.com/open-feature/flagd/pkg/model"
	"github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service/grpc/mocks"
	of "github.com/open-feature/golang-sdk/pkg/openfeature"
	"github.com/stretchr/testify/assert"
	schemaV1 "go.buf.build/grpc/go/open-feature/flagd/schema/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TestServiceResolveNumberArgs struct {
	name string
	mocks.MockResolveNumberArgs
	nilClient bool

	flagKey string
	evCtx   of.EvaluationContext

	value     float32
	variant   string
	reason    string
	err       error
	customErr string
}

func TestServiceResolveNumber(t *testing.T) {
	tests := []TestServiceResolveNumberArgs{
		{
			name: "happy path",
			MockResolveNumberArgs: mocks.MockResolveNumberArgs{
				InFK: "bool",
				InCtx: of.EvaluationContext{
					Attributes: map[string]interface{}{
						"con": "text",
					},
				},
				Out: &schemaV1.ResolveNumberResponse{
					Value:   12,
					Variant: "on",
					Reason:  models.StaticReason,
				},
			},
			flagKey: "bool",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"con": "text",
				},
			},
			variant: "on",
			value:   12,
			reason:  models.StaticReason,
			err:     nil,
		},
		{
			name: "custom error response",
			MockResolveNumberArgs: mocks.MockResolveNumberArgs{
				InFK: "bool",
				InCtx: of.EvaluationContext{
					Attributes: map[string]interface{}{
						"con": "text",
					},
				},
				OutErr: status.Error(codes.NotFound, "custom message"),
			},
			flagKey: "bool",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"con": "text",
				},
			},
			reason:    models.ErrorReason,
			customErr: "CUSTOM ERROR",
			err:       errors.New("CUSTOM ERROR"),
		},
		{
			name: "nil client",
			MockResolveNumberArgs: mocks.MockResolveNumberArgs{
				InFK: "bool",
				InCtx: of.EvaluationContext{
					Attributes: map[string]interface{}{
						"con": "text",
					},
				},
				OutErr: status.Error(codes.NotFound, "custom message"),
			},
			flagKey: "bool",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"con": "text",
				},
			},
			nilClient: true,
			reason:    models.ErrorReason,
			err:       errors.New("CONNECTION_ERROR"),
		},
		{
			name: "error parse failure",
			MockResolveNumberArgs: mocks.MockResolveNumberArgs{
				InFK: "bool",
				InCtx: of.EvaluationContext{
					Attributes: map[string]interface{}{
						"con": "text",
					},
				},
				OutErr: status.Error(codes.NotFound, "custom message"),
			},
			flagKey: "bool",
			evCtx: of.EvaluationContext{
				Attributes: map[string]interface{}{
					"con": "text",
				},
			},
			reason: models.ErrorReason,
			err:    errors.New("GENERAL"),
		},
	}

	for _, test := range tests {
		if test.customErr != "" {
			st, ok := status.FromError(test.MockResolveNumberArgs.OutErr)
			if !ok {
				t.Errorf("%s: malformed error status recieved, cannot attach custom properties", test.name)
			}
			stWD, err := st.WithDetails(&schemaV1.ErrorResponse{
				ErrorCode: test.customErr,
				Reason:    models.ErrorReason,
			})
			if err != nil {
				t.Error(err)
			}
			test.MockResolveNumberArgs.OutErr = stWD.Err()
		}
		srv := GRPCService{
			client: &mocks.MockClient{
				ReturnNilClient: test.nilClient,
				RNArgs:          test.MockResolveNumberArgs,
				Testing:         t,
			},
		}
		res, err := srv.ResolveNumber(test.flagKey, test.evCtx)
		if test.err != nil && !assert.EqualError(t, err, test.err.Error()) {
			t.Errorf("%s: unexpected error received, expected %v, got %v", test.name, test.err, err)
		}
		if res.Reason != test.reason {
			t.Errorf("%s: unexpected reason received, expected %v, got %v", test.name, test.reason, res.Reason)
		}
		if res.Value != test.value {
			t.Errorf("%s: unexpected value received, expected %v, got %v", test.name, test.value, res.Value)
		}
		if res.Variant != test.variant {
			t.Errorf("%s: unexpected variant received, expected %v, got %v", test.name, test.variant, res.Variant)
		}
	}
}
