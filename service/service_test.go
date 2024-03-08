package service_test

import (
	"fmt"
	"strings"
	"time"

	"github.com/pborman/uuid"
	"github.com/pivotal-cf/cf-redis-smoke-tests/redis"
	"github.com/pivotal-cf/cf-redis-smoke-tests/service/reporter"

	smokeTestCF "github.com/pivotal-cf/cf-redis-smoke-tests/cf"

	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Redis On-Demand", func() {
	var (
		testCF = smokeTestCF.CF{
			ShortTimeout: time.Minute * 6,
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
		serviceKey          smokeTestCF.Credentials

		CreateTlsSpecStep = func(app *redis.App, version string, key string, value string) *reporter.Step {
			tlsMessage := strings.ToUpper(version) + " clients are disabled"
			valueCheck := "protocol not supported"
			if hasTLSVersion(serviceKey, version) {
				tlsMessage = strings.ToUpper(version) + " clients are enabled"
				valueCheck = value
			}
			return reporter.NewStep(tlsMessage, app.ReadTLSAssert(version, key, valueCheck))
		}

		AssertLifeCycleBehavior = func(planName string) {
			It("creates, binds to, writes to, reads from, unbinds, and destroys", func() {
				var skip bool

				uri := fmt.Sprintf("https://%s.%s", appName, redisConfig.Config.AppsDomain)

				if redisConfig.UseHttpApp {
					uri = fmt.Sprintf("http://%s.%s", appName, redisConfig.Config.AppsDomain)
				}

				app := redis.NewApp(uri, testCF.ShortTimeout, retryInterval)

				enableServiceAccessStep := reporter.NewStep(
					fmt.Sprintf("Enable service plan access for '%s' org", wfh.GetOrganizationName()),
					testCF.EnableServiceAccessForPlan(wfh.GetOrganizationName(), redisConfig.ServiceName, planName),
				)
				serviceCreateStep := reporter.NewStep(
					fmt.Sprintf("Create a '%s' plan instance of Redis\n    Please refer to https://docs.vmware.com/en/Redis-for-VMware-Tanzu-Application-Service/3.4/redis-tanzu-application-service/GUID-smoke-tests.html for more help on diagnosing this issue", planName),
					testCF.CreateService(redisConfig.ServiceName, planName, serviceInstanceName, &skip),
				)

				smokeTestReporter.RegisterSpecSteps([]*reporter.Step{enableServiceAccessStep, serviceCreateStep})
				enableServiceAccessStep.Perform()
				serviceCreateStep.Perform()

				serviceCreateStep.Description = fmt.Sprintf("Create a '%s' plan instance of Redis", planName)

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
						"Read the Service Key",
						testCF.GetServiceKey(serviceInstanceName, &serviceKey),
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
				}

				smokeTestReporter.RegisterSpecSteps(specSteps)

				if skip {
					serviceCreateStep.Result = "SKIPPED"
				} else {
					performSteps(specSteps)
				}

				if !skip && !tlsEnforced(serviceKey) {
					standardPortSpecs := []*reporter.Step{
						reporter.NewStep(
							"Write a key/value pair to Redis",
							app.Write("mykey", "myvalue"),
						),
						reporter.NewStep(
							"Read the key/value pair back",
							app.ReadAssert("mykey", "myvalue"),
						),
					}
					smokeTestReporter.RegisterSpecSteps(standardPortSpecs)
					performSteps(standardPortSpecs)
				}
				if !skip && tlsEnabled(serviceKey) {
					tlsSpecSteps := []*reporter.Step{
						reporter.NewStep("Enable tls", testCF.SetEnv(appName, "tls_enabled", "true")),
						reporter.NewStep("Restage app", testCF.Restage(appName)),
						reporter.NewStep(
							"TLS: Write a key/value pair to Redis",
							app.Write("mykey", "myvalue2"),
						),
						reporter.NewStep(
							"TLS: Read the key/value pair back",
							app.ReadAssert("mykey", "myvalue2"),
						),
						CreateTlsSpecStep(app, "tlsv1", "mykey", "myvalue2"),
						CreateTlsSpecStep(app, "tlsv1.1", "mykey", "myvalue2"),
						CreateTlsSpecStep(app, "tlsv1.2", "mykey", "myvalue2"),
						CreateTlsSpecStep(app, "tlsv1.3", "mykey", "myvalue2"),
					}
					smokeTestReporter.RegisterSpecSteps(tlsSpecSteps)
					performSteps(tlsSpecSteps)
				}
			})
		}
	)

	Context("service instance", func() {
		Context("life-cycle", func() {
			for _, planName = range redisConfig.PlanNames {
				Context("for "+strings.ToUpper(planName)+" plans:", func() {
					AssertLifeCycleBehavior(planName)
				})
			}
		})
		BeforeEach(func() {
			appName = randomName()
			serviceInstanceName = randomName()
			securityGroupName = randomName()
			serviceKeyName = randomName()

			cfTestConfig := redisConfig.Config

			pushArgs := []string{
				"-m", "256M",
				"-p", appPath,
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
			performSteps(specSteps)
		})

		AfterEach(func() {
			specSteps := []*reporter.Step{
				reporter.NewStep(
					fmt.Sprintf("Unbind the %q plan instance", planName),
					testCF.UnbindService(appName, serviceInstanceName),
				),
				reporter.NewStep(
					fmt.Sprintf("Delete security group '%s'", securityGroupName),
					testCF.DeleteSecurityGroup(securityGroupName),
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
					"Delete the app",
					testCF.Delete(appName),
				),
			}

			smokeTestReporter.RegisterSpecSteps(specSteps)
			performSteps(specSteps)
		})
	})
})

func randomName() string {
	return uuid.NewRandom().String()
}

func hasTLSVersion(serviceKey smokeTestCF.Credentials, version string) bool {
	for _, v := range serviceKey.TLS_Versions {
		if version == v {
			return true
		}
	}
	return false
}

func tlsEnabled(serviceKey smokeTestCF.Credentials) bool {
	return (serviceKey.TLS_Port > 0)
}

func tlsEnforced(serviceKey smokeTestCF.Credentials) bool {
	return serviceKey.TLS_Port > 0 && serviceKey.Port == 0
}

func performSteps(specSteps []*reporter.Step) {
	for _, task := range specSteps {
		task.Perform()
	}
}
