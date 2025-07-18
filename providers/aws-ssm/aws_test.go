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

func TestWithDecryption(t *testing.T) {
	mockClient := NewMockSSMClient()

	aws := &awsService{
		client:     mockClient,
		decryption: false,
	}

	aws = aws.WithDecryption(true)
	if !aws.decryption {
		t.Error("Decryption flag should be true after WithDecryption(true)")
	}

	aws = aws.WithDecryption(false)
	if aws.decryption {
		t.Error("Decryption flag should be false after WithDecryption(false)")
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
