package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"cloud-efficiency-engine/internal/domain"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/pricing"
	pricingtypes "github.com/aws/aws-sdk-go-v2/service/pricing/types"
)

const (
	defaultCPUCostWeight    = 0.50
	defaultMemoryCostWeight = 0.50
)

type AWSPricingClient struct {
	inventory *NodeInventory

	capacityResolver *NodeCapacityResolver

	pricingClient PricingAPI

	cpuCostWeight float64
}

func NewAWSPricingClient(
	ec2Client EC2API,
	pricingClient PricingAPI,
) *AWSPricingClient {

	return &AWSPricingClient{
		inventory: NewNodeInventory(
			ec2Client,
		),

		capacityResolver: NewNodeCapacityResolver(
			ec2Client,
		),

		pricingClient: pricingClient,

		cpuCostWeight: defaultCPUCostWeight,
	}
}

func (c *AWSPricingClient) GetPricing(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
) (ResourcePrice, error) {

	if c.inventory == nil {
		return ResourcePrice{},
			fmt.Errorf(
				"AWS node inventory is not configured",
			)
	}

	if c.capacityResolver == nil {
		return ResourcePrice{},
			fmt.Errorf(
				"AWS capacity resolver is not configured",
			)
	}

	if c.pricingClient == nil {
		return ResourcePrice{},
			fmt.Errorf(
				"AWS pricing client is not configured",
			)
	}

	nodes, err :=
		c.inventory.ListRunningNodes(
			ctx,
			analysisContext,
		)

	if err != nil {
		return ResourcePrice{}, err
	}

	if len(nodes) == 0 {
		return ResourcePrice{},
			fmt.Errorf(
				"no running AWS EKS nodes found for cluster %q",
				analysisContext.ClusterName,
			)
	}

	instanceTypes :=
		make(
			[]string,
			0,
			len(nodes),
		)

	for _, node := range nodes {

		instanceTypes =
			append(
				instanceTypes,
				node.InstanceType,
			)
	}

	capacities, err :=
		c.capacityResolver.ResolveMany(
			ctx,
			instanceTypes,
		)

	if err != nil {
		return ResourcePrice{}, err
	}

	priceByInstanceType :=
		make(
			map[string]float64,
			len(capacities),
		)

	for instanceType := range capacities {

		price, err :=
			c.getOnDemandHourlyPrice(
				ctx,
				analysisContext.Region,
				instanceType,
			)

		if err != nil {
			return ResourcePrice{},
				err
		}

		priceByInstanceType[instanceType] =
			price
	}

	totalHourlyCost := 0.0

	totalVCPUs := 0.0

	totalMemoryGB := 0.0

	for _, node := range nodes {

		capacity, ok :=
			capacities[node.InstanceType]

		if !ok {
			return ResourcePrice{},
				fmt.Errorf(
					"capacity not found for instance type %q",
					node.InstanceType,
				)
		}

		hourlyPrice, ok :=
			priceByInstanceType[node.InstanceType]

		if !ok {
			return ResourcePrice{},
				fmt.Errorf(
					"price not found for instance type %q",
					node.InstanceType,
				)
		}

		totalHourlyCost +=
			hourlyPrice

		totalVCPUs +=
			float64(
				capacity.VCPUs,
			)

		totalMemoryGB +=
			float64(
				capacity.MemoryBytes,
			) /
				float64(
					1024*1024*1024,
				)
	}

	if totalVCPUs <= 0 {
		return ResourcePrice{},
			fmt.Errorf(
				"AWS cluster has no allocatable vCPU capacity",
			)
	}

	if totalMemoryGB <= 0 {
		return ResourcePrice{},
			fmt.Errorf(
				"AWS cluster has no allocatable memory capacity",
			)
	}

	memoryCostWeight :=
		1 -
			c.cpuCostWeight

	return ResourcePrice{
		CPUPerCoreHour: totalHourlyCost *
			c.cpuCostWeight /
			totalVCPUs,

		MemoryPerGBHour: totalHourlyCost *
			memoryCostWeight /
			totalMemoryGB,
	}, nil
}

func (c *AWSPricingClient) getOnDemandHourlyPrice(
	ctx context.Context,
	region string,
	instanceType string,
) (float64, error) {

	if region == "" {
		return 0,
			fmt.Errorf(
				"AWS region must not be empty",
			)
	}

	if instanceType == "" {
		return 0,
			fmt.Errorf(
				"AWS instance type must not be empty",
			)
	}

	output, err :=
		c.pricingClient.GetProducts(
			ctx,
			&pricing.GetProductsInput{
				ServiceCode: aws.String(
					"AmazonEC2",
				),

				FormatVersion: aws.String(
					"aws_v1",
				),

				MaxResults: aws.Int32(1),

				Filters: []pricingtypes.Filter{
					{
						Type: pricingtypes.FilterTypeTermMatch,

						Field: aws.String(
							"instanceType",
						),

						Value: aws.String(
							instanceType,
						),
					},

					{
						Type: pricingtypes.FilterTypeTermMatch,

						Field: aws.String(
							"regionCode",
						),

						Value: aws.String(
							region,
						),
					},

					{
						Type: pricingtypes.FilterTypeTermMatch,

						Field: aws.String(
							"operatingSystem",
						),

						Value: aws.String(
							"Linux",
						),
					},

					{
						Type: pricingtypes.FilterTypeTermMatch,

						Field: aws.String(
							"tenancy",
						),

						Value: aws.String(
							"Shared",
						),
					},

					{
						Type: pricingtypes.FilterTypeTermMatch,

						Field: aws.String(
							"preInstalledSw",
						),

						Value: aws.String(
							"NA",
						),
					},

					{
						Type: pricingtypes.FilterTypeTermMatch,

						Field: aws.String(
							"marketoption",
						),

						Value: aws.String(
							"OnDemand",
						),
					},
				},
			},
		)

	if err != nil {
		return 0,
			fmt.Errorf(
				"get AWS EC2 price for %s in %s: %w",
				instanceType,
				region,
				err,
			)
	}

	for _, rawProduct := range output.PriceList {

		price, err :=
			parseOnDemandHourlyPrice(
				rawProduct,
			)

		if err != nil {
			continue
		}

		return price, nil
	}

	return 0,
		fmt.Errorf(
			"AWS On-Demand price not found for %s in %s",
			instanceType,
			region,
		)
}

type awsPriceListProduct struct {
	Terms struct {
		OnDemand map[string]struct {
			PriceDimensions map[string]struct {
				Unit         string            `json:"unit"`
				BeginRange   string            `json:"beginRange"`
				PricePerUnit map[string]string `json:"pricePerUnit"`
			} `json:"priceDimensions"`
		} `json:"OnDemand"`
	} `json:"terms"`
}

func parseOnDemandHourlyPrice(
	rawProduct string,
) (float64, error) {

	var product awsPriceListProduct

	if err :=
		json.Unmarshal(
			[]byte(rawProduct),
			&product,
		); err != nil {

		return 0,
			fmt.Errorf(
				"decode AWS price product: %w",
				err,
			)
	}

	for _, term := range product.Terms.OnDemand {

		for _, dimension := range term.PriceDimensions {

			if dimension.Unit != "Hrs" {
				continue
			}

			rawPrice, ok :=
				dimension.PricePerUnit["USD"]

			if !ok {
				continue
			}

			price, err :=
				strconv.ParseFloat(
					rawPrice,
					64,
				)

			if err != nil {
				return 0,
					fmt.Errorf(
						"parse AWS hourly price %q: %w",
						rawPrice,
						err,
					)
			}

			return price, nil
		}
	}

	return 0,
		fmt.Errorf(
			"AWS product does not contain an hourly USD On-Demand price",
		)
}
