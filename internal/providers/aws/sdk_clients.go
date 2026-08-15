package aws

import (
	"context"

	awsSDK "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/pricing"
)

type EC2API interface {
	DescribeInstances(
		ctx context.Context,
		params *ec2.DescribeInstancesInput,
		optFns ...func(*ec2.Options),
	) (*ec2.DescribeInstancesOutput, error)

	DescribeInstanceTypes(
		ctx context.Context,
		params *ec2.DescribeInstanceTypesInput,
		optFns ...func(*ec2.Options),
	) (*ec2.DescribeInstanceTypesOutput, error)
}

type PricingAPI interface {
	GetProducts(
		ctx context.Context,
		params *pricing.GetProductsInput,
		optFns ...func(*pricing.Options),
	) (*pricing.GetProductsOutput, error)
}

type CostExplorerAPI interface {
	GetCostAndUsage(
		ctx context.Context,
		params *costexplorer.GetCostAndUsageInput,
		optFns ...func(*costexplorer.Options),
	) (*costexplorer.GetCostAndUsageOutput, error)
}

type SDKClients struct {
	EC2 EC2API

	Pricing PricingAPI

	CostExplorer CostExplorerAPI
}

func NewSDKClients(
	cfg awsSDK.Config,
) *SDKClients {

	return &SDKClients{
		EC2: ec2.NewFromConfig(
			cfg,
		),

		Pricing: pricing.NewFromConfig(
			cfg,
			func(
				options *pricing.Options,
			) {

				options.Region =
					"us-east-1"
			},
		),

		CostExplorer: costexplorer.NewFromConfig(
			cfg,
			func(
				options *costexplorer.Options,
			) {

				options.Region =
					"us-east-1"
			},
		),
	}
}
