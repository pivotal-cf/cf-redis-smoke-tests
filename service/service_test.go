package service_test

import (
	"fmt"
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
		Eventually(runner.Curl(pingUri, "-k"), shortTimeout, retryInterval).Should(Say("key not present"))
		fmt.Println()
	}

	createTestOrgAndSpace := func() {
		fmt.Println(testConfig)
		Eventually(cf.Cf("auth", testConfig.AdminUser, testConfig.AdminPassword), shortTimeout).Should(Exit(0), "Failed to `cf auth` with target Cloud Foundry")
		Eventually(cf.Cf("create-org", testConfig.OrgName), shortTimeout).Should(Exit(0), "Failed to create CF test org")
		Eventually(cf.Cf("target", "-o", testConfig.OrgName)).Should(Exit(0))
		Eventually(cf.Cf("create-space", testConfig.SpaceName), shortTimeout).Should(Exit(0), "Failed to create CF test space")
	}

	BeforeSuite(func() {
		createTestOrgAndSpace()

		context = services.NewContext(testConfig, "redis-test")
		context.Setup()
	})

	BeforeEach(func() {
		appName = randomName()
		Eventually(cf.Cf("push", appName, "-m", "256M", "-p", appPath, "-s", "cflinuxfs2", "-no-start"), shortTimeout).Should(Exit(0), "Failed to `cf push` test app")
	})

	AfterEach(func() {
		Eventually(cf.Cf("delete", appName, "-f"), shortTimeout).Should(Exit(0))
	})

	AfterSuite(func() {
		context.Teardown()
	})

	AssertLifeCycleBehavior := func(planName string) {
		It("can create, bind to, write to, read from, unbind, and destroy a service instance using the "+planName+" plan", func() {
			serviceInstanceName := randomName()

			createServiceSession := cf.Cf("create-service", redisConfig.ServiceName, planName, serviceInstanceName)
			createServiceSession.Wait(shortTimeout)

			createServiceStdout := createServiceSession.Out

			select {
			case <-createServiceStdout.Detect("FAILED"):
				Eventually(createServiceSession, shortTimeout).Should(Say("instance limit for this service has been reached"))
				Eventually(createServiceSession, shortTimeout).Should(Exit(1))
				fmt.Println("No Plan Instances available for testing plan:", planName)
			case <-createServiceStdout.Detect("OK"):
				Eventually(createServiceSession, shortTimeout).Should(Exit(0))
				Eventually(cf.Cf("bind-service", appName, serviceInstanceName), shortTimeout).Should(Exit(0))
				Eventually(cf.Cf("start", appName), longTimeout).Should(Exit(0))
				assertAppIsRunning(appName)

				uri := appUri(appName) + "/mykey"
				fmt.Println("Posting to url: ", uri)
				Eventually(runner.Curl("-d", "data=myvalue", "-X", "PUT", uri, "-k"), shortTimeout, retryInterval).Should(Say("success"))
				fmt.Println()

				fmt.Println("Getting from url: ", uri)
				Eventually(runner.Curl(uri, "-k"), shortTimeout, retryInterval).Should(Say("myvalue"))
				fmt.Println()

				Eventually(cf.Cf("unbind-service", appName, serviceInstanceName), shortTimeout).Should(Exit(0))
				Eventually(cf.Cf("delete-service", "-f", serviceInstanceName), shortTimeout).Should(Exit(0))
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
