package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
)

func LoadSDKClients(
	ctx context.Context,
	region string,
) (*SDKClients, error) {

	if region == "" {
		return nil,
			fmt.Errorf(
				"AWS region must not be empty",
			)
	}

	cfg, err :=
		config.LoadDefaultConfig(
			ctx,
			config.WithRegion(region),
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"load AWS configuration: %w",
				err,
			)
	}

	return NewSDKClients(cfg), nil
}
