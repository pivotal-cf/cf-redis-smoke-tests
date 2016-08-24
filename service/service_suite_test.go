package service_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/cf-test-helpers/services"
	"github.com/pivotal-cf/cf-redis-smoke-tests/service/reporter"
)

type redisTestConfig struct {
	ServiceName    string              `json:"service_name"`
	PlanNames      []string            `json:"plan_names"`
	SecurityGroups []map[string]string `json:"security_groups"`
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
var securityGroupConfigPath string
var redisConfig = loadConfig()
var smokeTestReporter *reporter.SmokeTestReport

func TestService(t *testing.T) {
	services.LoadConfig(os.Getenv("CONFIG_PATH"), &testConfig)
	err := services.ValidateConfig(&testConfig)
	if err != nil {
		panic(err)
	}

	securityGroupConfigPath, _ = writeJSONToTempFile(redisConfig.SecurityGroups)

	testConfig.TimeoutScale = 3

	smokeTestReporter = new(reporter.SmokeTestReport)

	reporter := []Reporter{
		Reporter(new(reporter.SmokeTestReport)),
	}

	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "P-Redis Smoke Tests", reporter)
}

func writeJSONToTempFile(object interface{}) (filePath string, err error) {
	file, err := ioutil.TempFile("", "redis-smoke-tests")
	if err != nil {
		return "", err
	}
	defer file.Close()

	filePath = file.Name()
	defer func() {
		if err != nil {
			os.RemoveAll(filePath)
		}
	}()

	bytes, err := json.Marshal(object)
	if err != nil {
		return "", err
	}

	_, err = file.Write(bytes)
	if err != nil {
		return "", err
	}

	return filePath, nil
}
