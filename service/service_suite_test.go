package service_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/cloudfoundry-incubator/cf-test-helpers/services"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type redisTestConfig struct {
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

var testConfig services.Config
var redisConfig = loadConfig()

func TestService(t *testing.T) {
	services.LoadConfig(os.Getenv("CONFIG_PATH"), &testConfig)
	// context_setup.TimeoutScale = 3
	// context_setup.SetupEnvironment(context_setup.NewContext(config.IntegrationConfig, "p-redis-smoke-tests"))
	RegisterFailHandler(Fail)
	RunSpecs(t, "P-Redis Smoke Tests")
}
