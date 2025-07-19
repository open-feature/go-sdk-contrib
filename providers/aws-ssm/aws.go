package awsssm

import (
	"context"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/open-feature/go-sdk/openfeature"
)

// SSMMetadataKey is the key used in flag metadata to store SSM result metadata
const SSMMetadataKey = "SSMMetadata"

type awsService struct {
	client     ssmClient
	decryption bool
}

type ssmClient interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, opts ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

func newAWSService(cfg aws.Config) (*awsService) {

	client := ssm.NewFromConfig(cfg)

	return &awsService{
		client: client,
	}
}

func (svc *awsService) ResolveBoolean(ctx context.Context, flag string, defaultValue bool, evalCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {

	res, err := svc.getValueFromSSM(ctx, flag)

	if err != nil {
		return openfeature.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				Reason:          openfeature.ErrorReason,
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()),
			},
		}
	}

	b, err := strconv.ParseBool(*res.Parameter.Value)

	if err != nil {
		return openfeature.BoolResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				Reason:          openfeature.ErrorReason,
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()),
			},
		}
	}

	return openfeature.BoolResolutionDetail{
		Value: b,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			Reason: openfeature.StaticReason,
			FlagMetadata: openfeature.FlagMetadata{
				SSMMetadataKey: res.ResultMetadata,
			},
		},
	}

}

func (svc *awsService) ResolveString(ctx context.Context, flag string, defaultValue string, evalCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {

	res, err := svc.getValueFromSSM(ctx, flag)

	if err != nil {
		return openfeature.StringResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				Reason:          openfeature.ErrorReason,
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()),
			},
		}
	}

	return openfeature.StringResolutionDetail{
		Value: *res.Parameter.Value,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			Reason: openfeature.StaticReason,
			FlagMetadata: openfeature.FlagMetadata{
				SSMMetadataKey: res.ResultMetadata,
			},
		},
	}

}

func (svc *awsService) ResolveInt(ctx context.Context, flag string, defaultValue int64, evalCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	res, err := svc.getValueFromSSM(ctx, flag)

	if err != nil {
		return openfeature.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				Reason:          openfeature.ErrorReason,
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()),
			},
		}
	}

	i, err := strconv.ParseInt(*res.Parameter.Value, 10, 64)

	if err != nil {
		return openfeature.IntResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				Reason:          openfeature.ErrorReason,
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()),
			},
		}
	}

	return openfeature.IntResolutionDetail{
		Value: i,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			Reason: openfeature.StaticReason,
			FlagMetadata: openfeature.FlagMetadata{
				"SSMMetadata": res.ResultMetadata,
			},
		},
	}
}

func (svc *awsService) ResolveFloat(ctx context.Context, flag string, defaultValue float64, evalCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	res, err := svc.getValueFromSSM(ctx, flag)

	if err != nil {
		return openfeature.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				Reason:          openfeature.ErrorReason,
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()),
			},
		}
	}

	f, err := strconv.ParseFloat(*res.Parameter.Value, 64)

	if err != nil {
		return openfeature.FloatResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				Reason:          openfeature.ErrorReason,
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()),
			},
		}
	}

	return openfeature.FloatResolutionDetail{
		Value: f,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			Reason: openfeature.StaticReason,
			FlagMetadata: openfeature.FlagMetadata{
				SSMMetadataKey: res.ResultMetadata,
			},
		},
	}
}

func (svc *awsService) ResolveObject(ctx context.Context, flag string, defaultValue any, evalCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	res, err := svc.getValueFromSSM(ctx, flag)

	if err != nil {
		return openfeature.InterfaceResolutionDetail{
			Value: defaultValue,
			ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
				Reason:          openfeature.ErrorReason,
				ResolutionError: openfeature.NewGeneralResolutionError(err.Error()),
			},
		}
	}

	return openfeature.InterfaceResolutionDetail{
		Value: defaultValue,
		ProviderResolutionDetail: openfeature.ProviderResolutionDetail{
			Reason: openfeature.StaticReason,
			FlagMetadata: openfeature.FlagMetadata{
				SSMMetadataKey: res.ResultMetadata,
			},
		},
	}
}

func (svc *awsService) getValueFromSSM(ctx context.Context, flag string) (*ssm.GetParameterOutput, error) {

	param := &ssm.GetParameterInput{
		Name:           &flag,
		WithDecryption: &svc.decryption,
	}

	res, err := svc.client.GetParameter(ctx, param)

	if err != nil {
		return nil, err
	}

	return res, nil
}
59 changes: 59 additions & 0 deletions 59
providers/aws-ssm/aws_mock.go
Viewed
Original file line number 	Diff line number 	Diff line change
@@ -0,0 +1,59 @@
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
124 changes: 124 additions & 0 deletions 124
providers/aws-ssm/aws_test.go
Viewed
Original file line number 	Diff line number 	Diff line change
@@ -0,0 +1,124 @@
package awsssm

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/open-feature/go-sdk/openfeature"
)

func TestNewAWSService(t *testing.T) {

	cfg, err := config.LoadDefaultConfig(context.Background())

	if err != nil {
		t.Fatalf("Failed to load AWS config: %v", err)
	}

	aws := newAWSService(cfg)
	if err != nil {
		t.Fatalf("Failed to create AWS service: %v", err)
	}
	if aws == nil {
		t.Fatal("AWS service should not be nil")
	}
}

func TestResolveBoolean(t *testing.T) {
	mockClient := NewMockSSMClient()
	mockClient.WithResponse("test", "true", types.ParameterTypeString)

	aws := &awsService{
		client: mockClient,
	}

	result := aws.ResolveBoolean(context.Background(), "test", false, openfeature.FlattenedContext{})
	if !result.Value {
		t.Errorf("Expected true, got %v", result.Value)
	}
	if result.ProviderResolutionDetail.Reason != openfeature.StaticReason {
		t.Errorf("Expected StaticReason, got %v", result.ProviderResolutionDetail.Reason)
	}

	mockClient = NewMockSSMClient()
	mockClient.WithResponse("test", "not-a-boolean", types.ParameterTypeString)

	aws = &awsService{
		client: mockClient,
	}
	result = aws.ResolveBoolean(context.Background(), "test", false, openfeature.FlattenedContext{})
	if result.Value != false {
		t.Errorf("Expected default value in error case, got %v", result.Value)
	}
	if result.ProviderResolutionDetail.Reason != openfeature.ErrorReason {
		t.Errorf("Expected ErrorReason, got %v", result.ProviderResolutionDetail.Reason)
	}
}

func TestResolveString(t *testing.T) {
	mockClient := NewMockSSMClient()
	mockClient.WithResponse("test", "mock-value", types.ParameterTypeString)

	aws := &awsService{
		client: mockClient,
	}

	result := aws.ResolveString(context.Background(), "test", "default", openfeature.FlattenedContext{})
	if result.Value != "mock-value" {
		t.Errorf("Expected mock-value, got %v", result.Value)
	}
	if result.ProviderResolutionDetail.Reason != openfeature.StaticReason {
		t.Errorf("Expected StaticReason, got %v", result.ProviderResolutionDetail.Reason)
	}

	mockClient = NewMockSSMClient()
	mockClient.WithError(fmt.Errorf("mock error"))

	aws = &awsService{
		client: mockClient,
	}
	result = aws.ResolveString(context.Background(), "test", "default", openfeature.FlattenedContext{})
	if result.Value != "default" {
		t.Errorf("Expected default value in error case, got %v", result.Value)
	}
	if result.ProviderResolutionDetail.Reason != openfeature.ErrorReason {
		t.Errorf("Expected ErrorReason, got %v", result.ProviderResolutionDetail.Reason)
	}
}

func TestResolveBooleanError(t *testing.T) {
	mockClient := NewMockSSMClient()
	mockClient.WithError(fmt.Errorf("mock error"))

	aws := &awsService{
		client: mockClient,
	}

	result := aws.ResolveBoolean(context.Background(), "test", false, openfeature.FlattenedContext{})
	if result.Value != false {
		t.Errorf("Expected default value in error case, got %v", result.Value)
	}
	if result.ProviderResolutionDetail.Reason != openfeature.ErrorReason {
		t.Errorf("Expected ErrorReason, got %v", result.ProviderResolutionDetail.Reason)
	}
}

func TestResolveStringError(t *testing.T) {
	mockClient := NewMockSSMClient()
	mockClient.WithError(fmt.Errorf("mock error"))

	aws := &awsService{
		client: mockClient,
	}

	result := aws.ResolveString(context.Background(), "test", "default", openfeature.FlattenedContext{})
	if result.Value != "default" {
		t.Errorf("Expected default value in error case, got %v", result.Value)
	}
	if result.ProviderResolutionDetail.Reason != openfeature.ErrorReason {
		t.Errorf("Expected ErrorReason, got %v", result.ProviderResolutionDetail.Reason)
	}
}