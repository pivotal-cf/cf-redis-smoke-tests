package service_test

import (
	"fmt"
	"time"

	"github.com/pivotal-cf/cf-redis-smoke-tests/redis"
	"github.com/pivotal-cf/cf-redis-smoke-tests/service/reporter"

	smokeTestCF "github.com/pivotal-cf/cf-redis-smoke-tests/cf"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/cf-redis-smoke-tests/service/utils"
)

var _ = Describe("Redis HA", Label("ha"), func() {
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
	)

	Context("service instance", func() {
		var skip bool

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
				fmt.Sprintf("Create a '%s' plan instance of Redis\n    Please refer to http://docs.pivotal.io/redis/smoke-tests.html for more help on diagnosing this issue", planName),
				testCF.CreateService(redisConfig.ServiceName, planName, serviceInstanceName, &skip),
			)

			smokeTestReporter.RegisterSpecSteps([]*reporter.Step{enableServiceAccessStep, serviceCreateStep})
			enableServiceAccessStep.Perform()
			serviceCreateStep.Perform()

			serviceCreateStep.Description = fmt.Sprintf("Create a '%s' plan instance of Redis", planName)

			specSteps = []*reporter.Step{
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

		Context("on-demand-cache", func() {
			for _, currentPlan := range redisConfig.Plans {
				if !currentPlan.HAEnabled {
					continue
				}

				planName = currentPlan.Name

				It("should successfully perform failover", func() {
					uri := fmt.Sprintf("https://%s.%s", appName, redisConfig.Config.AppsDomain)

					if redisConfig.UseHttpApp {
						uri = fmt.Sprintf("http://%s.%s", appName, redisConfig.Config.AppsDomain)
					}

					app := redis.NewApp(uri, testCF.ShortTimeout, retryInterval)

					if !skip {
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

						master := basicMasterValidations(app)
						replicas := basicReplicasValidation(app)

						performFailOver(app)

						newMaster := basicMasterValidations(app)
						newReplicas := basicReplicasValidation(app)

						performCrossValidation(master, newMaster, replicas, newReplicas)

						smokeTestReporter.RegisterSpecSteps(standardPortSpecs)
						performSteps(standardPortSpecs)
					}
				})
			}
		})
	})
})

func basicMasterValidations(app *redis.App) utils.RedisMasterInfo {
	master, err := utils.NewRedisMasterInfo(app.Read("master"))
	Expect(err).To(BeNil())

	Expect(master.Name).To(Equal("redis-master"))
	Expect(master.RoleReported).To(Equal("master"))
	Expect(master.NumOtherSentinels).To(Equal("2"))
	Expect(master.NumSlaves).To(Equal("2"))
	Expect(master.Quorum).To(Equal("2"))

	return master
}

func basicReplicasValidation(app *redis.App) []utils.RedisReplicaInfo {
	replicas, err := utils.NewRedisReplicaInfo(app.Read("replicas"))
	Expect(err).To(BeNil())

	Expect(replicas).To(HaveLen(2))
	for _, replica := range replicas {
		Expect(replica.MasterLinkStatus).To(Equal("ok"), fmt.Sprintf("replica %s's link to master is not 'ok'", replica.Name))
	}

	return replicas
}

func performFailOver(app *redis.App) {
	failoverSpecs := []*reporter.Step{
		reporter.NewStep(
			"Perform manual failover",
			app.ReadAssert("failover", "OK"),
		),
		reporter.NewStep(
			"Wait 20 seconds before failover completes",
			func() {
				time.Sleep(20 * time.Second)
			},
		),
	}

	smokeTestReporter.RegisterSpecSteps(failoverSpecs)
	performSteps(failoverSpecs)
}

func performCrossValidation(master, newMaster utils.RedisMasterInfo, replicas, newReplicas []utils.RedisReplicaInfo) {
	//Check if new master and old master is not the same
	Expect(newMaster.Ip).NotTo(Equal(master.Ip))
	Expect(newMaster.Runid).NotTo(Equal(master.Runid))

	//Check if new master is one of the old replica
	index := utils.GetIndexOfInstanceMatchingIpAndRunId(replicas, newMaster.Ip, newMaster.Runid)
	Expect(index).NotTo(Equal(-1))

	oldReplica := replicas[index]
	Expect(newMaster.Ip).To(Equal(oldReplica.Ip))
	Expect(newMaster.Port).To(Equal(oldReplica.Port))
	Expect(newMaster.Runid).To(Equal(oldReplica.Runid))

	//Check if odl master has joined as replica
	index = utils.GetIndexOfInstanceMatchingIpAndRunId(newReplicas, master.Ip, master.Runid)
	Expect(index).NotTo(Equal(-1))

	oldMaster := newReplicas[index]
	Expect(oldMaster.Ip).To(Equal(master.Ip))
	Expect(oldMaster.Port).To(Equal(master.Port))
	Expect(oldMaster.Runid).To(Equal(master.Runid))
}
