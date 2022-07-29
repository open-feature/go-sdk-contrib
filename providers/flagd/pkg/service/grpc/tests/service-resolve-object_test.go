package tests

import (
	"errors"
	"reflect"
	"testing"

	models "github.com/open-feature/flagd/pkg/model"
	service "github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service/grpc"
	"github.com/open-feature/golang-sdk-contrib/providers/flagd/pkg/service/grpc/tests/mocks"
	schemaV1 "go.buf.build/grpc/go/open-feature/flagd/schema/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type TestServiceResolveObjectArgs struct {
	name string
	mocks.MockResolveObjectArgs
	nilClient bool

	flagKey string
	evCtx   interface{}

	valueIn   map[string]interface{}
	valueOut  map[string]interface{}
	variant   string
	reason    string
	err       error
	customErr string
}

func TestServiceResolveObject(t *testing.T) {
	tests := []TestServiceResolveObjectArgs{
		{
			name: "happy path",
			MockResolveObjectArgs: mocks.MockResolveObjectArgs{
				InFK: "bool",
				InCtx: map[string]interface{}{
					"dog": "cat",
				},
				Out: &schemaV1.ResolveObjectResponse{
					Variant: "on",
					Reason:  models.StaticReason,
				},
			},
			flagKey: "bool",
			evCtx: map[string]interface{}{
				"dog": "cat",
			},
			variant:  "on",
			valueIn:  map[string]interface{}{"food": "bars"},
			valueOut: map[string]interface{}{"food": "bars"},
			reason:   models.StaticReason,
			err:      nil,
		},
		{
			name:    "FormatAsStructpb fails",
			flagKey: "bool",
			evCtx:   "not a map[string]interface{}!",
			reason:  models.ErrorReason,
			err:     errors.New(models.ParseErrorCode),
		},
		{
			name: "custom error response",
			MockResolveObjectArgs: mocks.MockResolveObjectArgs{
				InFK: "bool",
				InCtx: map[string]interface{}{
					"dog": "cat",
				},
				OutErr: status.Error(codes.NotFound, "custom message"),
			},
			flagKey: "bool",
			evCtx: map[string]interface{}{
				"dog": "cat",
			},
			reason:    models.ErrorReason,
			customErr: "CUSTOM ERROR",
			err:       errors.New("CUSTOM ERROR"),
		},
		{
			name: "nil client",
			MockResolveObjectArgs: mocks.MockResolveObjectArgs{
				InFK: "bool",
				InCtx: map[string]interface{}{
					"dog": "cat",
				},
				OutErr: status.Error(codes.NotFound, "custom message"),
			},
			flagKey: "bool",
			evCtx: map[string]interface{}{
				"dog": "cat",
			},
			nilClient: true,
			reason:    models.ErrorReason,
			err:       errors.New("CONNECTION_ERROR"),
		},
		{
			name: "error parse failure",
			MockResolveObjectArgs: mocks.MockResolveObjectArgs{
				InFK: "bool",
				InCtx: map[string]interface{}{
					"dog": "cat",
				},
				OutErr: status.Error(codes.NotFound, "custom message"),
			},
			flagKey: "bool",
			evCtx: map[string]interface{}{
				"dog": "cat",
			},
			reason: models.ErrorReason,
			err:    errors.New("GENERAL"),
		},
	}

	for _, test := range tests {
		if test.customErr != "" {
			st, ok := status.FromError(test.MockResolveObjectArgs.OutErr)
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
			test.MockResolveObjectArgs.OutErr = stWD.Err()
		}
		if test.valueIn != nil && test.valueOut != nil {
			inF, err := structpb.NewStruct(test.valueIn)
			if err != nil {
				t.Error(err)
			}
			test.MockResolveObjectArgs.Out.Value = inF
		}
		srv := service.GRPCService{
			Client: &mocks.MockClient{
				ReturnNilClient: test.nilClient,
				ROArgs:          test.MockResolveObjectArgs,
				Testing:         t,
			},
		}
		res, err := srv.ResolveObject(test.flagKey, test.evCtx)
		if (test.err != nil && err != nil) && test.err.Error() != err.Error() {
			t.Errorf("%s: unexpected error received, expected %v, got %v", test.name, test.err, err)
		}
		if res.Reason != test.reason {
			t.Errorf("%s: unexpected reason received, expected %v, got %v", test.name, test.reason, res.Reason)
		}
		if res.Value != nil && test.valueOut != nil && !reflect.DeepEqual(res.Value.AsMap(), test.valueOut) {
			t.Errorf("%s: unexpected value received, expected %v, got %v", test.name, test.valueOut, res.Value)
		}
		if res.Variant != test.variant {
			t.Errorf("%s: unexpected variant received, expected %v, got %v", test.name, test.variant, res.Variant)
		}
	}
}
