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

		context services.Context
	)

	BeforeSuite(func() {
		context = services.NewContext(cfTestConfig, "redis-test")

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
				testCF.API(cfTestConfig.ApiEndpoint, cfTestConfig.SkipSSLValidation),
			),
			reporter.NewStep(
				"Log in as admin",
				testCF.Auth(cfTestConfig.AdminUser, cfTestConfig.AdminPassword),
			),
			reporter.NewStep(
				"Create 'redis-smoke-tests' quota",
				testCF.CreateQuota("redis-smoke-test-quota", createQuotaArgs...),
			),
			reporter.NewStep(
				fmt.Sprintf("Create '%s' org", cfTestConfig.OrgName),
				testCF.CreateOrg(cfTestConfig.OrgName, "redis-smoke-test-quota"),
			),
			reporter.NewStep(
				fmt.Sprintf("Enable service access for '%s' org", cfTestConfig.OrgName),
				testCF.EnableServiceAccess(cfTestConfig.OrgName, redisConfig.ServiceName),
			),
			reporter.NewStep(
				fmt.Sprintf("Target '%s' org", cfTestConfig.OrgName),
				testCF.TargetOrg(cfTestConfig.OrgName),
			),
			reporter.NewStep(
				fmt.Sprintf("Create '%s' space", cfTestConfig.SpaceName),
				testCF.CreateSpace(cfTestConfig.SpaceName),
			),
			reporter.NewStep(
				fmt.Sprintf("Create user '%s'", regularContext.Username),
				testCF.CreateUser(regularContext.Username, regularContext.Password),
			),
			reporter.NewStep(
				fmt.Sprintf(
					"Assign user '%s' to 'SpaceManager' role for '%s'",
					regularContext.Username,
					cfTestConfig.SpaceName,
				),
				testCF.SetSpaceRole(regularContext.Username, regularContext.Org, cfTestConfig.SpaceName, "SpaceManager"),
			),
			reporter.NewStep(
				fmt.Sprintf(
					"Assign user '%s' to 'SpaceDeveloper' role for '%s'",
					regularContext.Username,
					cfTestConfig.SpaceName,
				),
				testCF.SetSpaceRole(regularContext.Username, regularContext.Org, cfTestConfig.SpaceName, "SpaceDeveloper"),
			),
			reporter.NewStep(
				fmt.Sprintf(
					"Assign user '%s' to 'SpaceAuditor' role for '%s'",
					regularContext.Username,
					cfTestConfig.SpaceName,
				),
				testCF.SetSpaceRole(regularContext.Username, regularContext.Org, cfTestConfig.SpaceName, "SpaceAuditor"),
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
				fmt.Sprintf("Target '%s' org and '%s' space", cfTestConfig.OrgName, cfTestConfig.SpaceName),
				testCF.TargetOrgAndSpace(cfTestConfig.OrgName, cfTestConfig.SpaceName),
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
				fmt.Sprintf("Delete the %q plan instance", planName),
				testCF.DeleteService(serviceInstanceName),
			),
			reporter.NewStep(
				fmt.Sprintf("Ensure service instance for plan %q has been deleted", planName),
				testCF.EnsureServiceInstanceGone(serviceInstanceName),
			),
			reporter.NewStep(
				"Delete the app",
				testCF.Delete(appName),
			),
			reporter.NewStep(
				"Log out",
				testCF.Logout(),
			),
			reporter.NewStep(
				"Log in as admin",
				testCF.Auth(cfTestConfig.AdminUser, cfTestConfig.AdminPassword),
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

		smokeTestReporter.RegisterSpecSteps(specSteps)

		for _, task := range specSteps {
			task.Perform()
		}
	})

	AfterSuite(func() {
		regularContext := context.RegularUserContext()

		afterSuiteSteps := []*reporter.Step{
			reporter.NewStep(
				"Connect to CloudFoundry",
				testCF.API(cfTestConfig.ApiEndpoint, cfTestConfig.SkipSSLValidation),
			),
			reporter.NewStep(
				"Log in as admin",
				testCF.Auth(cfTestConfig.AdminUser, cfTestConfig.AdminPassword),
			),
			reporter.NewStep(
				fmt.Sprintf("Target '%s' org and '%s' space", cfTestConfig.OrgName, cfTestConfig.SpaceName),
				testCF.TargetOrgAndSpace(cfTestConfig.OrgName, cfTestConfig.SpaceName),
			),
			reporter.NewStep(
				"Ensure no service-instances left",
				testCF.EnsureAllServiceInstancesGone(),
			),
			reporter.NewStep(
				fmt.Sprintf("Delete user '%s'", regularContext.Username),
				testCF.DeleteUser(regularContext.Username),
			),
			reporter.NewStep(
				fmt.Sprintf("Delete org '%s'", cfTestConfig.OrgName),
				testCF.DeleteOrg(cfTestConfig.OrgName),
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
			regularContext := context.RegularUserContext()

			var skip bool

			uri := fmt.Sprintf("https://%s.%s", appName, cfTestConfig.AppsDomain)
			app := redis.NewApp(uri, testCF.ShortTimeout, retryInterval)

			serviceCreateStep := reporter.NewStep(
				fmt.Sprintf("Create a '%s' plan instance of Redis\n    Please refer to http://docs.pivotal.io/redis/smoke-tests.html for more help on diagnosing this issue", planName),
				testCF.CreateService(redisConfig.ServiceName, planName, serviceInstanceName, &skip),
			)

			smokeTestReporter.RegisterSpecSteps([]*reporter.Step{serviceCreateStep})

			specSteps := []*reporter.Step{
				reporter.NewStep(
					fmt.Sprintf("Bind the redis sample app to the '%s' plan instance of Redis", planName),
					testCF.BindService(appName, serviceInstanceName),
				),
				reporter.NewStep(
					"Log in as admin",
					testCF.Auth(cfTestConfig.AdminUser, cfTestConfig.AdminPassword),
				),
				reporter.NewStep(
					fmt.Sprintf("Target '%s' org and '%s' space", cfTestConfig.OrgName, cfTestConfig.SpaceName),
					testCF.TargetOrgAndSpace(cfTestConfig.OrgName, cfTestConfig.SpaceName),
				),
				reporter.NewStep(
					"Create and bind security group for running smoke tests",
					testCF.CreateAndBindSecurityGroup("redis-smoke-tests-sg", appName, cfTestConfig.OrgName, cfTestConfig.SpaceName),
				),
				reporter.NewStep(
					fmt.Sprintf("Log in as %s", regularContext.Username),
					testCF.Auth(regularContext.Username, regularContext.Password),
				),
				reporter.NewStep(
					fmt.Sprintf("Target '%s' org and '%s' space", cfTestConfig.OrgName, cfTestConfig.SpaceName),
					testCF.TargetOrgAndSpace(cfTestConfig.OrgName, cfTestConfig.SpaceName),
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
