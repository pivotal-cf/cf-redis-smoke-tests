package service_test

import (
	"encoding/json"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/cf-redis-smoke-tests/retry"
	"github.com/pivotal-cf/cf-redis-smoke-tests/service/reporter"
)

type retryConfig struct {
	BaselineMilliseconds uint   `json:"baseline_interval_milliseconds"`
	Attempts             uint   `json:"max_attempts"`
	BackoffAlgorithm     string `json:"backoff"`
}

func (rc retryConfig) Backoff() retry.Backoff {
	baseline := time.Duration(rc.BaselineMilliseconds) * time.Millisecond

	algorithm := strings.ToLower(rc.BackoffAlgorithm)

	switch algorithm {
	case "linear":
		return retry.Linear(baseline)
	case "exponential":
		return retry.Linear(baseline)
	default:
		return retry.None(baseline)
	}
}

func (rc retryConfig) MaxRetries() int {
	return int(rc.Attempts)
}

type redisTestConfig struct {
	config.Config

	ServiceName string      `json:"service_name"`
	PlanNames   []string    `json:"plan_names"`
	Retry       retryConfig `json:"retry"`
}

func loadRedisTestConfig(path string) redisTestConfig {
	file, err := os.Open(path)
	Expect(err).NotTo(HaveOccurred())

	defer file.Close()

	testConfig := redisTestConfig{}
	err = json.NewDecoder(file).Decode(&testConfig)
	Expect(err).NotTo(HaveOccurred())

	testConfig.Config.TimeoutScale = 3

	return testConfig
}

var (
	configPath   = os.Getenv("CONFIG_PATH")

	redisConfig  redisTestConfig

	smokeTestReporter *reporter.SmokeTestReport

	wfh *workflowhelpers.ReproducibleTestSuiteSetup
)

func TestService(t *testing.T) {
	smokeTestReporter = new(reporter.SmokeTestReport)

	reporter := []Reporter{
		Reporter(smokeTestReporter),
	}

	SynchronizedBeforeSuite(func() []byte {
		redisConfig  = loadRedisTestConfig(configPath)

		wfh = workflowhelpers.NewTestSuiteSetup(&redisConfig.Config)
		wfh.Setup()

		return []byte{}
	}, func(data []byte) {})

	SynchronizedAfterSuite(func() {}, func() {
		wfh.Teardown()
	})

	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "P-Redis Smoke Tests", reporter)
}
