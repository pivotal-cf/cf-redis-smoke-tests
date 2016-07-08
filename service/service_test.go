package service_test

import (
	"fmt"
	"strings"
	"time"

	"github.com/pborman/uuid/uuid"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/runner"
	"github.com/cloudfoundry-incubator/cf-test-helpers/services"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Redis Service", func() {
	var shortTimeout = time.Second * 180
	var longTimeout = time.Minute * 15
	var retryInterval = time.Second * 1
	var appPath = "../assets/cf-redis-example-app"

	var appName string
	var context services.Context

	randomName := func() string {
		return uuid.NewRandom().String()
	}

	appUri := func(appName string) string {
		return "https://" + appName + "." + testConfig.AppsDomain
	}

	assertAppIsRunning := func(appName string) {
		pingUri := appUri(appName) + "/ping"
		fmt.Println("Checking that the app is responding at url: ", pingUri)
		Eventually(runner.Curl(pingUri, "-k"), shortTimeout, retryInterval).Should(Say("key not present"),
			"Test app deployed but did not respond in time",
		)
		fmt.Println()
	}

	createTestOrgAndSpace := func() {
		Eventually(cf.Cf("auth", testConfig.AdminUser, testConfig.AdminPassword), shortTimeout).Should(Exit(0), "Failed to `cf auth` with target Cloud Foundry")
		Eventually(cf.Cf("create-org", testConfig.OrgName), shortTimeout).Should(Exit(0), "Failed to create CF test org")
		Eventually(cf.Cf("target", "-o", testConfig.OrgName)).Should(Exit(0))
		Eventually(cf.Cf("create-space", testConfig.SpaceName), shortTimeout).Should(Exit(0), "Failed to create CF test space")
	}

	createAndBindSecurityGroup := func() {
		Eventually(cf.Cf("auth", testConfig.AdminUser, testConfig.AdminPassword), shortTimeout).Should(Exit(0), "Failed to `cf auth` with target Cloud Foundry")
		Eventually(cf.Cf("create-security-group", "redis-smoke-tests-sg", securityGroupConfigPath), shortTimeout).Should(Exit(0), "Failed to create security group")
		Eventually(cf.Cf("bind-security-group", "redis-smoke-tests-sg", testConfig.OrgName, testConfig.SpaceName), shortTimeout).Should(Exit(0), "Failed to bind security group to space")
	}

	deleteSecurityGroup := func() {
		Eventually(cf.Cf("auth", testConfig.AdminUser, testConfig.AdminPassword), shortTimeout).Should(Exit(0), "Failed to `cf auth` with target Cloud Foundry")
		Eventually(cf.Cf("delete-security-group", "redis-smoke-tests-sg", "-f"), shortTimeout).Should(Exit(0), "Failed to remove security group")
	}

	BeforeSuite(func() {
		createTestOrgAndSpace()
		createAndBindSecurityGroup()

		context = services.NewContext(testConfig, "redis-test")
		context.Setup()
	})

	BeforeEach(func() {
		appName = randomName()
		Eventually(cf.Cf("push", appName, "-m", "256M", "-p", appPath, "-s", "cflinuxfs2", "-no-start"), shortTimeout).Should(Exit(0), "Failed to `cf push` test app")
	})

	AfterEach(func() {
		Eventually(cf.Cf("delete", appName, "-f"), shortTimeout).Should(Exit(0), "Failed to `cf delete` test app")
	})

	AfterSuite(func() {
		context.Teardown()
		deleteSecurityGroup()
	})

	AssertLifeCycleBehavior := func(planName string) {
		It(strings.ToUpper(planName)+": create, bind to, write to, read from, unbind, and destroy a service instance", func() {
			serviceInstanceName := randomName()

			createServiceSession := cf.Cf("create-service", redisConfig.ServiceName, planName, serviceInstanceName)
			createServiceSession.Wait(shortTimeout)

			createServiceStdout := createServiceSession.Out

			select {
			case <-createServiceStdout.Detect("FAILED"):
				Eventually(createServiceSession, shortTimeout).Should(Say("instance limit for this service has been reached"),
					"Failed to bind Redis service instance to test app",
				)
				Eventually(createServiceSession, shortTimeout).Should(Exit(1))
				fmt.Println("No Plan Instances available for testing " + planName + " plan")

			case <-createServiceStdout.Detect("OK"):
				Eventually(createServiceSession, shortTimeout).Should(Exit(0),
					"Failed to create Redis service instance",
				)
				Eventually(cf.Cf("bind-service", appName, serviceInstanceName), shortTimeout).Should(Exit(0),
					"Failed to bind Redis service instance to test app",
				)
				Eventually(cf.Cf("start", appName), longTimeout).Should(Exit(0),
					"Failed to start test app",
				)

				assertAppIsRunning(appName)

				uri := appUri(appName) + "/mykey"
				fmt.Println("Posting to url: ", uri)
				Eventually(runner.Curl("-d", "data=myvalue", "-X", "PUT", uri, "-k"), shortTimeout, retryInterval).Should(Say("success"),
					"Failed to write to test "+planName+" instance",
				)
				fmt.Println()

				fmt.Println("Getting from url: ", uri)
				Eventually(runner.Curl(uri, "-k"), shortTimeout, retryInterval).Should(Say("myvalue"),
					"Failed to read from test "+planName+" instance",
				)
				fmt.Println()

				Eventually(cf.Cf("unbind-service", appName, serviceInstanceName), shortTimeout).Should(Exit(0),
					"Failed to unbind "+planName+" instance from test app",
				)
				Eventually(cf.Cf("delete-service", "-f", serviceInstanceName), shortTimeout).Should(Exit(0),
					"Failed to delete test "+planName+" instance",
				)
			}
			createServiceStdout.CancelDetects()

		})
	}

	Context("for each plan", func() {
		for _, planName := range redisConfig.PlanNames {
			AssertLifeCycleBehavior(planName)
		}
	})
})
