package service_test

import (
	"testing"

	"github.com/cloudfoundry-incubator/cf-test-helpers/services/context_setup"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type redisTestConfig struct {
	context_setup.IntegrationConfig

	serviceName string
	planNames   []string
}

func loadConfig() redisTestConfig {
	return redisTestConfig{
		IntegrationConfig: context_setup.IntegrationConfig{
			ApiEndpoint:                   "api.10.244.0.34.xip.io",
			AppsDomain:                    "10.244.0.34.xip.io",
			AdminUser:                     "admin",
			AdminPassword:                 "admin",
			CreatePermissiveSecurityGroup: true,
			SkipSSLValidation:             true,
		},
		serviceName: "p-redis",
		planNames:   []string{"shared-vm", "dedicated-vm"},
	}
}

var config = loadConfig()

func TestService(t *testing.T) {
	context_setup.TimeoutScale = 1
	context_setup.SetupEnvironment(context_setup.NewContext(config.IntegrationConfig, "redis"))
	RegisterFailHandler(Fail)
	RunSpecs(t, "P-Redis Smoke Tests")
}
