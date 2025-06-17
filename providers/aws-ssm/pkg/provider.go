package awsssm

import (
	"github.com/open-feature/go-sdk-contrib/providers/aws-ssm/pkg/service"
	"github.com/open-feature/go-sdk/openfeature"
)


const providerName = "AWS SSM"

type Provider struct {
	service *service.AWS
}


func NewProvider(opts ProviderOptions) *Provider{
	return &Provider{}
}


func (p *Provider) Metadata() openfeature.Metadata {
	return openfeature.Metadata{
		Name: "aws-ssm",
	}
}

func (p *Provider) Hooks() []openfeature.Hook {
	return []openfeature.Hook{}
}
