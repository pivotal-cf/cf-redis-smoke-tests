package service_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/cloudfoundry-incubator/cf-test-helpers/services/context_setup"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type redisTestConfig struct {
	context_setup.IntegrationConfig

	ServiceName string   `json:"service_name"`
	PlanNames   []string `json:"plan_names"`
}

func loadConfig() (testConfig redisTestConfig) {
	path := os.Getenv("CONFIG_PATH")
	configFile, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(&testConfig)
	if err != nil {
		panic(err)
	}

	return testConfig
}

var config = loadConfig()

func TestService(t *testing.T) {
	context_setup.TimeoutScale = 3
	context_setup.SetupEnvironment(context_setup.NewContext(config.IntegrationConfig, "p-redis-smoke-tests"))
	RegisterFailHandler(Fail)
	RunSpecs(t, "P-Redis Smoke Tests")
}
