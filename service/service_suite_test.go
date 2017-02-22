package service_test

import (
	"encoding/json"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/cf-test-helpers/services"
	"github.com/pivotal-cf/cf-redis-smoke-tests/service/reporter"
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

	err = json.NewDecoder(configFile).Decode(&testConfig)
	if err != nil {
		panic(err)
	}

	return testConfig
}

var (
	testConfig        services.Config
	smokeTestReporter *reporter.SmokeTestReport

	redisConfig = loadConfig()
)

func TestService(t *testing.T) {
	services.LoadConfig(os.Getenv("CONFIG_PATH"), &testConfig)
	err := services.ValidateConfig(&testConfig)
	if err != nil {
		panic(err)
	}

	testConfig.TimeoutScale = 3

	smokeTestReporter = new(reporter.SmokeTestReport)

	reporter := []Reporter{
		Reporter(smokeTestReporter),
	}

	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "P-Redis Smoke Tests", reporter)
}
