package service_test

import (
	"fmt"
	"strings"
	"time"

	"github.com/pborman/uuid"
	"github.com/pivotal-cf/cf-redis-smoke-tests/redis"
	"github.com/pivotal-cf/cf-redis-smoke-tests/service/reporter"

	smokeTestCF "github.com/pivotal-cf/cf-redis-smoke-tests/cf"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("Redis Service", func() {
	var (
		testCF = smokeTestCF.CF{
			ShortTimeout: time.Minute * 3,
			LongTimeout:  time.Minute * 15,
			RetryBackoff: redisConfig.Retry.Backoff(),
			MaxRetries:   redisConfig.Retry.MaxRetries(),
		}

		retryInterval = time.Second

		appPath             = "../assets/cf-redis-example-app"
		serviceInstanceName string
		appName             string
		planName            string
		securityGroupName   string
		serviceKeyName      string
	)

	BeforeEach(func() {
		appName = randomName()
		serviceInstanceName = randomName()
		securityGroupName = randomName()
		serviceKeyName = randomName()

		cfTestConfig := redisConfig.Config

		pushArgs := []string{
			"-m", "256M",
			"-p", appPath,
			"-d", cfTestConfig.AppsDomain,
			"-b", "ruby_buildpack",
			"--no-start",
		}

		var loginStep *reporter.Step
		if cfTestConfig.AdminClient != "" && cfTestConfig.AdminClientSecret != "" {
			loginStep = reporter.NewStep(
				"Log in as admin client",
				testCF.AuthClient(cfTestConfig.AdminClient, cfTestConfig.AdminClientSecret),
			)
		} else {
			loginStep = reporter.NewStep(
				"Log in as admin user",
				testCF.Auth(cfTestConfig.AdminUser, cfTestConfig.AdminPassword),
			)
		}

		specSteps := []*reporter.Step{
			reporter.NewStep(
				"Connect to CloudFoundry",
				testCF.API(cfTestConfig.ApiEndpoint, cfTestConfig.SkipSSLValidation),
			),
			loginStep,
			reporter.NewStep(
				fmt.Sprintf("Target '%s' org and '%s' space", wfh.GetOrganizationName(), wfh.TestSpace.SpaceName()),
				testCF.TargetOrgAndSpace(wfh.GetOrganizationName(), wfh.TestSpace.SpaceName()),
			),
			reporter.NewStep(
				"Push the redis sample app to Cloud Foundry",
				testCF.Push(appName, pushArgs...),
			),
		}

		smokeTestReporter.ClearSpecSteps()
		smokeTestReporter.RegisterSpecSteps(specSteps)

		for _, task := range specSteps {
			task.Perform()
		}
	})

	AfterEach(func() {
		specSteps := []*reporter.Step{
			reporter.NewStep(
				fmt.Sprintf("Unbind the %q plan instance", planName),
				testCF.UnbindService(appName, serviceInstanceName),
			),
			reporter.NewStep(
				fmt.Sprintf("Delete the service key %s for the %q plan instance", serviceKeyName, planName),
				testCF.DeleteServiceKey(serviceInstanceName, serviceKeyName),
			),
			reporter.NewStep(
				fmt.Sprintf("Delete the %q plan instance", planName),
				testCF.DeleteService(serviceInstanceName),
			),
			reporter.NewStep(
				fmt.Sprintf("Ensure service instance for plan %q has been deleted", planName),
				testCF.EnsureServiceInstanceGone(serviceInstanceName),
			),
			reporter.NewStep(
				fmt.Sprintf("Remove service access to plan %s", planName),
				testCF.DisableServiceAccess(wfh.GetOrganizationName(), redisConfig.ServiceName),
			),
			reporter.NewStep(
				"Delete the app",
				testCF.Delete(appName),
			),
		}

		smokeTestReporter.RegisterSpecSteps(specSteps)

		for _, task := range specSteps {
			task.Perform()
		}
	})

	AssertLifeCycleBehavior := func(planName string) {
		It(strings.ToUpper(planName)+": create, bind to, write to, read from, unbind, and destroy a service instance", func() {
			var skip bool

			uri := fmt.Sprintf("https://%s.%s", appName, redisConfig.Config.AppsDomain)
			app := redis.NewApp(uri, testCF.ShortTimeout, retryInterval)

			enableServiceAccessStep := reporter.NewStep(
				fmt.Sprintf("Enable service access for '%s' org", wfh.GetOrganizationName()),
				testCF.EnableServiceAccess(wfh.GetOrganizationName(), redisConfig.ServiceName),
			)

			serviceCreateStep := reporter.NewStep(
				fmt.Sprintf("Create a '%s' plan instance of Redis\n    Please refer to http://docs.pivotal.io/redis/smoke-tests.html for more help on diagnosing this issue", planName),
				testCF.CreateService(redisConfig.ServiceName, planName, serviceInstanceName, &skip),
			)

			smokeTestReporter.RegisterSpecSteps([]*reporter.Step{enableServiceAccessStep})
			smokeTestReporter.RegisterSpecSteps([]*reporter.Step{serviceCreateStep})

			specSteps := []*reporter.Step{
				reporter.NewStep(
					fmt.Sprintf("Bind the redis sample app '%s' to the '%s' plan instance '%s' of Redis", appName, planName, serviceInstanceName),
					testCF.BindService(appName, serviceInstanceName),
				),
				reporter.NewStep(
					fmt.Sprintf("Create service key for the '%s' plan instance '%s' of Redis", planName, serviceInstanceName),
					testCF.CreateServiceKey(serviceInstanceName, serviceKeyName),
				),
				reporter.NewStep(
					fmt.Sprintf("Create and bind security group '%s' for running smoke tests", securityGroupName),
					testCF.CreateAndBindSecurityGroup(securityGroupName, serviceInstanceName, wfh.GetOrganizationName(), wfh.TestSpace.SpaceName()),
				),
				reporter.NewStep(
					"Start the app",
					testCF.Start(appName),
				),
				reporter.NewStep(
					"Verify that the app is responding",
					app.IsRunning(),
				),
				reporter.NewStep(
					"Write a key/value pair to Redis",
					app.Write("mykey", "myvalue"),
				),
				reporter.NewStep(
					"Read the key/value pair back",
					app.ReadAssert("mykey", "myvalue"),
				),
			}

			smokeTestReporter.RegisterSpecSteps(specSteps)

			enableServiceAccessStep.Perform()
			serviceCreateStep.Perform()
			serviceCreateStep.Description = fmt.Sprintf("Create a '%s' plan instance of Redis", planName)

			if skip {
				serviceCreateStep.Result = "SKIPPED"
			} else {
				for _, task := range specSteps {
					task.Perform()
				}
			}
		})
	}

	Context("for each plan", func() {
		for _, planName = range redisConfig.PlanNames {
			AssertLifeCycleBehavior(planName)
		}
	})
})

func randomName() string {
	return uuid.NewRandom().String()
}
