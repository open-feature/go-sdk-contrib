package mocks

import (
	"context"
	"reflect"
	"testing"

	of "github.com/open-feature/golang-sdk/pkg/openfeature"
	schemaV1 "go.buf.build/grpc/go/open-feature/flagd/schema/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

type MockClient struct {
	RBArgs MockResolveBooleanArgs
	RNArgs MockResolveNumberArgs
	RSArgs MockResolveStringArgs
	ROArgs MockResolveObjectArgs

	ReturnNilClient bool

	Testing *testing.T
}

func (m *MockClient) Instance() schemaV1.ServiceClient {
	if m.ReturnNilClient {
		return nil
	}
	return &MockServiceClient{
		RBArgs: m.RBArgs,
		RNArgs: m.RNArgs,
		RSArgs: m.RSArgs,
		ROArgs: m.ROArgs,

		Testing: m.Testing,
	}
}

type MockServiceClient struct {
	RBArgs MockResolveBooleanArgs
	RNArgs MockResolveNumberArgs
	RSArgs MockResolveStringArgs
	ROArgs MockResolveObjectArgs

	Testing *testing.T
}

type MockResolveBooleanArgs struct {
	InFK   string
	InCtx  of.EvaluationContext
	Out    *schemaV1.ResolveBooleanResponse
	OutErr error
}

func (m *MockServiceClient) ResolveBoolean(ctx context.Context, in *schemaV1.ResolveBooleanRequest, opts ...grpc.CallOption) (*schemaV1.ResolveBooleanResponse, error) {
	if in.FlagKey != m.RBArgs.InFK {
		m.Testing.Errorf("unexpected value for flagKey received, expected %v got %v", m.RBArgs.InFK, in.FlagKey)
		return m.RBArgs.Out, m.RBArgs.OutErr
	}
	inF, err := structpb.NewStruct(m.RBArgs.InCtx.Attributes)
	if err != nil {
		m.Testing.Error(err)
		return m.RBArgs.Out, m.RBArgs.OutErr
	}
	if !reflect.DeepEqual(inF, in.Context) {
		m.Testing.Errorf("unexpected value for context received, expected %v got %v", inF, in.Context)
	}
	return m.RBArgs.Out, m.RBArgs.OutErr
}

type MockResolveNumberArgs struct {
	InFK   string
	InCtx  of.EvaluationContext
	Out    *schemaV1.ResolveNumberResponse
	OutErr error
}

func (m *MockServiceClient) ResolveNumber(ctx context.Context, in *schemaV1.ResolveNumberRequest, opts ...grpc.CallOption) (*schemaV1.ResolveNumberResponse, error) {
	if in.FlagKey != m.RNArgs.InFK {
		m.Testing.Errorf("unexpected value for flagKey received, expected %v got %v", m.RNArgs.InFK, in.FlagKey)
		return m.RNArgs.Out, m.RNArgs.OutErr
	}
	inF, err := structpb.NewStruct(m.RNArgs.InCtx.Attributes)
	if err != nil {
		m.Testing.Error(err)
		return m.RNArgs.Out, m.RNArgs.OutErr
	}
	if !reflect.DeepEqual(inF, in.Context) {
		m.Testing.Errorf("unexpected value for context received, expected %v got %v", inF, in.Context)
	}
	return m.RNArgs.Out, m.RNArgs.OutErr
}

type MockResolveStringArgs struct {
	InFK   string
	InCtx  of.EvaluationContext
	Out    *schemaV1.ResolveStringResponse
	OutErr error
}

func (m *MockServiceClient) ResolveString(ctx context.Context, in *schemaV1.ResolveStringRequest, opts ...grpc.CallOption) (*schemaV1.ResolveStringResponse, error) {
	if in.FlagKey != m.RSArgs.InFK {
		m.Testing.Errorf("unexpected value for flagKey received, expected %v got %v", m.RSArgs.InFK, in.FlagKey)
		return m.RSArgs.Out, m.RSArgs.OutErr
	}
	inF, err := structpb.NewStruct(m.RSArgs.InCtx.Attributes)
	if err != nil {
		m.Testing.Error(err)
		return m.RSArgs.Out, m.RSArgs.OutErr
	}
	if !reflect.DeepEqual(inF, in.Context) {
		m.Testing.Errorf("unexpected value for context received, expected %v got %v", inF, in.Context)
	}
	return m.RSArgs.Out, m.RSArgs.OutErr
}

type MockResolveObjectArgs struct {
	InFK   string
	InCtx  of.EvaluationContext
	OutMap map[string]interface{}
	Out    *schemaV1.ResolveObjectResponse
	OutErr error
}

func (m *MockServiceClient) ResolveObject(ctx context.Context, in *schemaV1.ResolveObjectRequest, opts ...grpc.CallOption) (*schemaV1.ResolveObjectResponse, error) {
	if in.FlagKey != m.ROArgs.InFK {
		m.Testing.Errorf("unexpected value for flagKey received, expected %v got %v", m.ROArgs.InFK, in.FlagKey)
		return m.ROArgs.Out, m.ROArgs.OutErr
	}
	inF, err := structpb.NewStruct(m.ROArgs.InCtx.Attributes)
	if err != nil {
		m.Testing.Error(err)
		return m.ROArgs.Out, m.ROArgs.OutErr
	}
	if !reflect.DeepEqual(inF, in.Context) {
		m.Testing.Errorf("unexpected value for context received, expected %v got %v", inF, in.Context)
	}
	return m.ROArgs.Out, m.ROArgs.OutErr
}
