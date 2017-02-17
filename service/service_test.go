package service_test

import (
	"fmt"
	"strings"
	"time"

	"github.com/pborman/uuid"
	"github.com/pivotal-cf/cf-redis-smoke-tests/redis"
	"github.com/pivotal-cf/cf-redis-smoke-tests/service/reporter"

	"github.com/pivotal-cf-experimental/cf-test-helpers/services"
	smokeTestCF "github.com/pivotal-cf/cf-redis-smoke-tests/cf"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("Redis Service", func() {
	var (
		shortTimeout        = time.Minute * 3
		longTimeout         = time.Minute * 15
		retryInterval       = time.Second * 1
		appPath             = "../assets/cf-redis-example-app"
		serviceInstanceName string
		appName             string
		planName            string
		context             services.Context
	)

	BeforeSuite(func() {
		context = services.NewContext(testConfig, "redis-test")

		testCF := smokeTestCF.CF{ShortTimeout: shortTimeout}

		createQuotaArgs := []string{
			"-m", "10G",
			"-r", "1000",
			"-s", "100",
			"--allow-paid-service-plans",
		}

		regularContext := context.RegularUserContext()

		beforeSuiteSteps := []*reporter.Step{
			reporter.NewStep(
				"Connect to CloudFoundry",
				testCF.API(testConfig.ApiEndpoint, testConfig.SkipSSLValidation),
			),
			reporter.NewStep(
				"Log in as admin",
				testCF.Auth(testConfig.AdminUser, testConfig.AdminPassword),
			),
			reporter.NewStep(
				"Create 'redis-smoke-tests' quota",
				testCF.CreateQuota("redis-smoke-test-quota", createQuotaArgs...),
			),
			reporter.NewStep(
				fmt.Sprintf("Create '%s' org", testConfig.OrgName),
				testCF.CreateOrg(testConfig.OrgName, "redis-smoke-test-quota"),
			),
			reporter.NewStep(
				fmt.Sprintf("Enable service access for '%s' org", testConfig.OrgName),
				testCF.EnableServiceAccess(testConfig.OrgName, redisConfig.ServiceName),
			),
			reporter.NewStep(
				fmt.Sprintf("Target '%s' org", testConfig.OrgName),
				testCF.TargetOrg(testConfig.OrgName),
			),
			reporter.NewStep(
				fmt.Sprintf("Create '%s' space", testConfig.SpaceName),
				testCF.CreateSpace(testConfig.SpaceName),
			),
			reporter.NewStep(
				fmt.Sprintf("Create user '%s'", regularContext.Username),
				testCF.CreateUser(regularContext.Username, testConfig.ConfigurableTestPassword),
			),
			reporter.NewStep(
				fmt.Sprintf(
					"Assign user '%s' to 'SpaceManager' role for '%s'",
					regularContext.Username,
					testConfig.SpaceName,
				),
				testCF.SetSpaceRole(regularContext.Username, regularContext.Org, testConfig.SpaceName, "SpaceManager"),
			),
			reporter.NewStep(
				fmt.Sprintf(
					"Assign user '%s' to 'SpaceDeveloper' role for '%s'",
					regularContext.Username,
					testConfig.SpaceName,
				),
				testCF.SetSpaceRole(regularContext.Username, regularContext.Org, testConfig.SpaceName, "SpaceDeveloper"),
			),
			reporter.NewStep(
				fmt.Sprintf(
					"Assign user '%s' to 'SpaceAuditor' role for '%s'",
					regularContext.Username,
					testConfig.SpaceName,
				),
				testCF.SetSpaceRole(regularContext.Username, regularContext.Org, testConfig.SpaceName, "SpaceAuditor"),
			),
			reporter.NewStep(
				"Create security group for running smoke tests",
				testCF.CreateSecurityGroup("redis-smoke-tests-sg", securityGroupConfigPath),
			),
			reporter.NewStep(
				fmt.Sprintf("Bind security group for running smoke tests to '%s'", testConfig.SpaceName),
				testCF.BindSecurityGroup("redis-smoke-tests-sg", testConfig.OrgName, testConfig.SpaceName),
			),
			reporter.NewStep(
				"Log out",
				testCF.Logout(),
			),
		}

		smokeTestReporter.RegisterBeforeSuiteSteps(beforeSuiteSteps)

		for _, task := range beforeSuiteSteps {
			task.Perform()
		}
	})

	BeforeEach(func() {
		testCF := smokeTestCF.CF{ShortTimeout: shortTimeout}
		regularContext := context.RegularUserContext()
		appName = randomName()
		serviceInstanceName = randomName()

		pushArgs := []string{
			"-m", "256M",
			"-p", appPath,
			"-s", "cflinuxfs2",
			"-no-start",
		}

		specSteps := []*reporter.Step{
			reporter.NewStep(
				fmt.Sprintf("Log in as %s", regularContext.Username),
				testCF.Auth(regularContext.Username, regularContext.Password),
			),
			reporter.NewStep(
				fmt.Sprintf("Target '%s' org and '%s' space", testConfig.OrgName, testConfig.SpaceName),
				testCF.TargetOrgAndSpace(testConfig.OrgName, testConfig.SpaceName),
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
		testCF := smokeTestCF.CF{ShortTimeout: shortTimeout}

		specSteps := []*reporter.Step{
			reporter.NewStep(
				fmt.Sprintf("Unbind the %q plan instance", planName),
				testCF.UnbindService(appName, serviceInstanceName),
			),
			reporter.NewStep(
				fmt.Sprintf("Delete the %q plan instance", planName),
				testCF.DeleteService(serviceInstanceName),
			),
			reporter.NewStep(
				"Delete the app",
				testCF.Delete(appName),
			),
			reporter.NewStep(
				"Log out",
				testCF.Logout(),
			),
		}

		smokeTestReporter.RegisterSpecSteps(specSteps)

		for _, task := range specSteps {
			task.Perform()
		}
	})

	AfterSuite(func() {
		regularContext := context.RegularUserContext()

		testCF := smokeTestCF.CF{ShortTimeout: shortTimeout}

		afterSuiteSteps := []*reporter.Step{
			reporter.NewStep(
				"Connect to CloudFoundry",
				testCF.API(testConfig.ApiEndpoint, testConfig.SkipSSLValidation),
			),
			reporter.NewStep(
				"Log in as admin",
				testCF.Auth(testConfig.AdminUser, testConfig.AdminPassword),
			),
			reporter.NewStep(
				fmt.Sprintf("Delete user '%s'", regularContext.Username),
				testCF.DeleteUser(regularContext.Username),
			),
			reporter.NewStep(
				"Delete security group 'redis-smoke-tests-sg'",
				testCF.DeleteSecurityGroup("redis-smoke-tests-sg"),
			),
			reporter.NewStep(
				"Log out",
				testCF.Logout(),
			),
		}

		smokeTestReporter.RegisterAfterSuiteSteps(afterSuiteSteps)

		for _, task := range afterSuiteSteps {
			task.Perform()
		}
	})

	AssertLifeCycleBehavior := func(planName string) {
		It(strings.ToUpper(planName)+": create, bind to, write to, read from, unbind, and destroy a service instance", func() {
			testCF := smokeTestCF.CF{
				ShortTimeout: shortTimeout,
				LongTimeout:  longTimeout,
			}

			var skip bool

			uri := fmt.Sprintf("https://%s.%s", appName, testConfig.AppsDomain)
			app := redis.NewApp(uri, shortTimeout, retryInterval)

			serviceCreateStep := reporter.NewStep(
				fmt.Sprintf("Create a '%s' plan instance of Redis\n    Please refer to http://docs.pivotal.io/redis/smoke-tests.html for more help on diagnosing this issue", planName),
				testCF.CreateService(redisConfig.ServiceName, planName, serviceInstanceName, &skip),
			)

			smokeTestReporter.RegisterSpecSteps([]*reporter.Step{serviceCreateStep})

			specSteps := []*reporter.Step{
				reporter.NewStep(
					"Bind the redis sample app to the shared vm plan instance of Redis",
					testCF.BindService(appName, serviceInstanceName),
				),
				reporter.NewStep(
					"Set the service name of the bound instance as an environment variable for the app",
					testCF.SetEnv(appName, "service_name", serviceInstanceName),
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
