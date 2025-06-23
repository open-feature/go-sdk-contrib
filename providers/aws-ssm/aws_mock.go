package awsssm

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)


type mockSSMClient struct {
	err       error
	responses map[string]*types.Parameter
}

var (
	mockValue    = "mock-value"
	mockDataType = "text/plain"
)

func NewMockSSMClient() *mockSSMClient {
	return &mockSSMClient{
		responses: make(map[string]*types.Parameter),
	}
}

func (m *mockSSMClient) WithResponse(name string, value string, pType types.ParameterType) *mockSSMClient {
	m.responses[name] = &types.Parameter{
		Value:    &value,
		Type:     pType,
		DataType: &mockDataType,
	}
	return m
}

func (m *mockSSMClient) WithError(err error) *mockSSMClient {
	m.err = err
	return m
}

func (m *mockSSMClient) GetParameter(ctx context.Context, params *ssm.GetParameterInput, opts ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if param, ok := m.responses[*params.Name]; ok {
		return &ssm.GetParameterOutput{
			Parameter: param,
		}, nil
	}

	return &ssm.GetParameterOutput{
		Parameter: &types.Parameter{
			Value:    &mockValue,
			Type:     types.ParameterTypeString,
			DataType: &mockDataType,
		},
	}, nil
}
