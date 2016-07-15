package service_test

import (
	"fmt"
	"strings"
	"time"

	"github.com/pborman/uuid"

	"github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	"github.com/pivotal-cf-experimental/cf-test-helpers/runner"
	"github.com/pivotal-cf-experimental/cf-test-helpers/services"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Redis Service", func() {
	var shortTimeout = time.Minute * 3
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
		Eventually(runner.Curl(pingUri, "-k"), shortTimeout, retryInterval).Should(
			Say("key not present"),
			`{"FailReason": "Test app deployed but did not respond in time"}`,
		)
		fmt.Println()
	}

	createTestOrgAndSpace := func() {
		apiCmd := []string{"api", testConfig.ApiEndpoint}

		if testConfig.SkipSSLValidation {
			apiCmd = append(apiCmd, "--skip-ssl-validation")
		}

		Eventually(cf.Cf(apiCmd...), shortTimeout).Should(
			Exit(0),
			`{"FailReason": "Failed to target Cloud Foundry"}`,
		)

		Eventually(cf.Cf("auth", testConfig.AdminUser, testConfig.AdminPassword), shortTimeout).Should(
			Exit(0),
			"{\"FailReason\": \"Failed to `cf auth` with target Cloud Foundry\"}",
		)

		Eventually(cf.Cf("create-org", testConfig.OrgName), shortTimeout).Should(
			Exit(0),
			`{"FailReason": "Failed to create CF test org"}`,
		)

		Eventually(cf.Cf("target", "-o", testConfig.OrgName), shortTimeout).Should(
			Exit(0),
			`{"FailReason": "Failed to target test org"}`,
		)

		Eventually(cf.Cf("create-space", testConfig.SpaceName), shortTimeout).Should(
			Exit(0),
			`{"FailReason": "Failed to create CF test space"}`,
		)
	}

	createAndBindSecurityGroup := func() {
		Eventually(cf.Cf("auth", testConfig.AdminUser, testConfig.AdminPassword), shortTimeout).Should(
			Exit(0),
			"{\"FailReason\": \"Failed to `cf auth` with target Cloud Foundry\"}",
		)

		Eventually(
			cf.Cf("create-security-group", "redis-smoke-tests-sg", securityGroupConfigPath),
			shortTimeout,
		).Should(
			Exit(0),
			`{"FailReason": "Failed to create security group"}`,
		)

		Eventually(
			cf.Cf("bind-security-group", "redis-smoke-tests-sg", testConfig.OrgName, testConfig.SpaceName),
			shortTimeout,
		).Should(
			Exit(0),
			`{"FailReason": "Failed to bind security group to space"}`,
		)
	}

	deleteSecurityGroup := func() {
		Eventually(cf.Cf("auth", testConfig.AdminUser, testConfig.AdminPassword), shortTimeout).Should(
			Exit(0),
			"{\"FailReason\": \"Failed to `cf auth` with target Cloud Foundry\"}",
		)

		Eventually(cf.Cf("delete-security-group", "redis-smoke-tests-sg", "-f"), shortTimeout).Should(
			Exit(0),
			`{"FailReason": "Failed to remove security group"}`,
		)
	}

	BeforeSuite(func() {
		createTestOrgAndSpace()
		createAndBindSecurityGroup()

		context = services.NewContext(testConfig, "redis-test")
		context.Setup()
	})

	BeforeEach(func() {
		appName = randomName()
		Eventually(
			cf.Cf("push", appName, "-m", "256M", "-p", appPath, "-s", "cflinuxfs2", "-no-start"),
			shortTimeout,
		).Should(
			Exit(0),
			"{\"FailReason\": \"Failed to `cf push` test app\"}",
		)
	})

	AfterEach(func() {
		Eventually(cf.Cf("delete", appName, "-f"), shortTimeout).Should(
			Exit(0),
			"{\"FailReason\": \"Failed to `cf delete` test app\"}",
		)
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
				Eventually(createServiceSession, shortTimeout).Should(
					Say("instance limit for this service has been reached"),
					`{"FailReason": "Failed to bind Redis service instance to test app"}`,
				)
				Eventually(createServiceSession, shortTimeout).Should(Exit(1))
				fmt.Printf("No Plan Instances available for testing %s plan\n", planName)

			case <-createServiceStdout.Detect("OK"):
				Eventually(createServiceSession, shortTimeout).Should(
					Exit(0),
					`{"FailReason": "Failed to create Redis service instance"}`,
				)

				Eventually(cf.Cf("bind-service", appName, serviceInstanceName), shortTimeout).Should(
					Exit(0),
					`{"FailReason": "Failed to bind Redis service instance to test app"}`,
				)

				Eventually(cf.Cf("start", appName), longTimeout).Should(
					Exit(0),
					`{"FailReason": "Failed to start test app"}`,
				)

				assertAppIsRunning(appName)

				uri := appUri(appName) + "/mykey"
				fmt.Println("Posting to url: ", uri)
				Eventually(
					runner.Curl("-d", "data=myvalue", "-X", "PUT", uri, "-k"),
					shortTimeout,
					retryInterval,
				).Should(
					Say("success"),
					fmt.Sprintf(`{"FailReason": "Failed to write to test %s instance"}`, planName),
				)
				fmt.Println()
				fmt.Println("Getting from url: ", uri)

				Eventually(
					runner.Curl(uri, "-k"),
					shortTimeout,
					retryInterval,
				).Should(
					Say("myvalue"),
					fmt.Sprintf(`{"FailReason": "Failed to read from test %s instance"}`, planName),
				)
				fmt.Println()

				Eventually(cf.Cf("unbind-service", appName, serviceInstanceName), shortTimeout).Should(
					Exit(0),
					fmt.Sprintf(`{"FailReason": "Failed to unbind %s instance from test app"}`, planName),
				)

				Eventually(cf.Cf("delete-service", "-f", serviceInstanceName), shortTimeout).Should(
					Exit(0),
					fmt.Sprintf(`{"FailReason": "Failed to delete test %s instance"}`, planName),
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
