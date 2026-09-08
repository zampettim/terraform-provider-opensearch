package provider

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/opensearch-project/opensearch-go/v4/signer"
	"github.com/opensearch-project/opensearch-go/v4/signer/awsv2"
)

func newAWSSigner(conf *ProviderConf) (signer.Signer, error) {
	ctx := context.Background()
	service := conf.awsSig4Service
	if service == "" {
		service = "es"
	}

	var options []func(*config.LoadOptions) error
	if conf.awsRegion != "" {
		options = append(options, config.WithRegion(conf.awsRegion))
	}
	if conf.awsAccessKeyId != "" {
		options = append(options, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(conf.awsAccessKeyId, conf.awsSecretAccessKey, conf.awsSessionToken)))
	} else if conf.awsWebIdentityRoleArn != "" && conf.awsWebIdentityTokenFile != "" {
		baseConfig, err := config.LoadDefaultConfig(ctx, options...)
		if err != nil {
			return nil, fmt.Errorf("error loading AWS config for web identity: %w", err)
		}
		webIdentitySTS := sts.NewFromConfig(baseConfig)
		webIdentity := stscreds.NewWebIdentityRoleProvider(
			webIdentitySTS,
			conf.awsWebIdentityRoleArn,
			stscreds.IdentityTokenFile(conf.awsWebIdentityTokenFile),
			func(options *stscreds.WebIdentityRoleOptions) {
				options.RoleSessionName = conf.awsAssumeRoleSessionName
			},
		)

		if conf.awsAssumeRoleArn == "" {
			options = append(options, config.WithCredentialsProvider(webIdentity))
		} else {
			baseConfig.Credentials = webIdentity
			assumeRoleSTS := sts.NewFromConfig(baseConfig)
			var assumeOptions []func(*stscreds.AssumeRoleOptions)
			if conf.awsAssumeRoleExternalID != "" {
				assumeOptions = append(assumeOptions, func(options *stscreds.AssumeRoleOptions) {
					options.ExternalID = &conf.awsAssumeRoleExternalID
				})
			}
			if conf.awsAssumeRoleSessionName != "" {
				assumeOptions = append(assumeOptions, func(options *stscreds.AssumeRoleOptions) {
					options.RoleSessionName = conf.awsAssumeRoleSessionName
				})
			}
			options = append(options, config.WithCredentialsProvider(stscreds.NewAssumeRoleProvider(assumeRoleSTS, conf.awsAssumeRoleArn, assumeOptions...)))
		}
	} else if conf.awsAssumeRoleArn != "" {
		baseConfig, err := config.LoadDefaultConfig(ctx, options...)
		if err != nil {
			return nil, fmt.Errorf("error loading AWS config for assume role: %w", err)
		}
		stsClient := sts.NewFromConfig(baseConfig)
		var assumeOptions []func(*stscreds.AssumeRoleOptions)
		if conf.awsAssumeRoleExternalID != "" {
			assumeOptions = append(assumeOptions, func(options *stscreds.AssumeRoleOptions) {
				options.ExternalID = &conf.awsAssumeRoleExternalID
			})
		}
		if conf.awsAssumeRoleSessionName != "" {
			assumeOptions = append(assumeOptions, func(options *stscreds.AssumeRoleOptions) {
				options.RoleSessionName = conf.awsAssumeRoleSessionName
			})
		}
		options = append(options, config.WithCredentialsProvider(stscreds.NewAssumeRoleProvider(stsClient, conf.awsAssumeRoleArn, assumeOptions...)))
	} else if conf.awsProfile != "" {
		options = append(options, config.WithSharedConfigProfile(conf.awsProfile))
	}

	awsConfig, err := config.LoadDefaultConfig(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("error loading AWS config: %w", err)
	}
	if service == "aoss" {
		return awsv2.NewSignerWithService(awsConfig, service)
	}
	return awsv2.NewSigner(awsConfig)
}
