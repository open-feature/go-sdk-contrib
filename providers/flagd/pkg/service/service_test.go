package service_test

import (
	"context"
	"testing"

	"github.com/bufbuild/connect-go"
	gomock "github.com/golang/mock/gomock"
	"github.com/open-feature/go-sdk-contrib/providers/flagd/pkg/service"
	"github.com/open-feature/go-sdk/pkg/openfeature"
	"github.com/stretchr/testify/assert"
	schemav1 "go.buf.build/open-feature/flagd-connect/open-feature/flagd/schema/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type TestServiceResolveBooleanArgs struct {
	name string

	mockIn    *schemav1.ResolveBooleanRequest
	nilClient bool

	mockOut   *schemav1.ResolveBooleanResponse
	mockErr   error
	customErr string

	flagKey string
	evCtx   map[string]interface{}

	outErr     error
	outReason  openfeature.Reason
	outVariant string
	outValue   bool
}

func TestServiceResolveBoolean(t *testing.T) {
	tests := []TestServiceResolveBooleanArgs{
		{
			name: "happy path",
			mockIn: &schemav1.ResolveBooleanRequest{
				FlagKey: "flag",
				Context: nil,
			},
			mockOut: &schemav1.ResolveBooleanResponse{
				Value:   true,
				Variant: "on",
				Reason:  string(openfeature.DefaultReason),
			},
			mockErr: nil,
			flagKey: "flag",
			evCtx: map[string]interface{}{
				"this": "that",
			},
			outErr:     nil,
			outReason:  openfeature.DefaultReason,
			outValue:   true,
			outVariant: "on",
		},
		{
			name: "custom error response",
			mockIn: &schemav1.ResolveBooleanRequest{
				FlagKey: "flag",
				Context: nil,
			},
			mockOut: &schemav1.ResolveBooleanResponse{
				Value:   true,
				Variant: "on",
				Reason:  string(openfeature.DefaultReason),
			},
			mockErr:   status.Error(codes.NotFound, "custom message"),
			customErr: "CUSTOM_ERROR",
			flagKey:   "flag",
			evCtx: map[string]interface{}{
				"this": "that",
			},
			outErr:    openfeature.NewGeneralResolutionError(""),
			outReason: openfeature.ErrorReason,
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, test := range tests {
		mockClient := NewMockiClient(ctrl)
		mock := NewMockClientMockTarget(ctrl)

		f, err := structpb.NewStruct(test.evCtx)
		if err != nil {
			t.Fatal(err)
		}
		test.mockIn.Context = f

		mock.EXPECT().ResolveBoolean(gomock.Any(), connect.NewRequest(test.mockIn)).AnyTimes().Return(connect.NewResponse(test.mockOut), test.mockErr)
		mockClient.EXPECT().Instance().Return(mock)
		srv := service.Service{
			Client: mockClient,
		}
		res, err := srv.ResolveBoolean(context.Background(), test.flagKey, test.evCtx)
		if test.outErr != nil && !assert.EqualError(t, err, test.outErr.Error()) {
			t.Errorf("%s: unexpected error received, expected %v, got %v", test.name, test.outErr, err)
		}
		// if res.Reason != test.outReason {
		// 	t.Errorf("%s: unexpected reason received, expected %v, got %v", test.name, test.outReason, res.Reason)
		// }
		if res.Value != test.outValue {
			t.Errorf("%s: unexpected value received, expected %v, got %v", test.name, test.outValue, res.Value)
		}
		if res.Variant != test.outVariant {
			t.Errorf("%s: unexpected variant received, expected %v, got %v", test.name, test.outVariant, res.Variant)
		}
	}
}
