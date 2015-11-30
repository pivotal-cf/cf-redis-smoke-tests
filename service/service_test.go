package service_test

import (
	"fmt"
	"time"

	"code.google.com/p/go-uuid/uuid"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/runner"
	"github.com/cloudfoundry-incubator/cf-test-helpers/services/context_setup"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Redis Service", func() {
	var timeout = time.Second * 60
	var retryInterval = time.Second * 1
	var appPath = "../assets/cf-redis-example-app"

	var appName string

	randomName := func() string {
		return uuid.NewRandom().String()
	}

	appUri := func(appName string) string {
		return "https://" + appName + "." + config.AppsDomain
	}

	assertAppIsRunning := func(appName string) {
		pingUri := appUri(appName) + "/ping"
		fmt.Println("Checking that the app is responding at url: ", pingUri)
		Eventually(runner.Curl(pingUri, "-k"), context_setup.ScaledTimeout(timeout), retryInterval).Should(Say("key not present"))
		fmt.Println("\n")
	}

	BeforeEach(func() {
		appName = randomName()
		Eventually(cf.Cf("push", appName, "-m", "256M", "-p", appPath, "-s", "cflinuxfs2", "-no-start"), context_setup.ScaledTimeout(timeout)).Should(Exit(0))
	})

	AfterEach(func() {
		Eventually(cf.Cf("delete", appName, "-f"), context_setup.ScaledTimeout(timeout)).Should(Exit(0))
	})

	AssertLifeCycleBehavior := func(planName string) {
		It("can create, bind to, write to, read from, unbind, and destroy a service instance using the "+planName+" plan", func() {
			serviceInstanceName := randomName()

			createServiceSession := cf.Cf("create-service", config.ServiceName, planName, serviceInstanceName)
			createServiceSession.Wait(context_setup.ScaledTimeout(timeout))

			createServiceStdout := createServiceSession.Out

			select {
			case <-createServiceStdout.Detect("FAILED"):
				Eventually(createServiceSession, context_setup.ScaledTimeout(timeout)).Should(Say("instance limit for this service has been reached"))
				Eventually(createServiceSession, context_setup.ScaledTimeout(timeout)).Should(Exit(1))
				fmt.Println("No Plan Instances available for testing plan:", planName)
			case <-createServiceStdout.Detect("OK"):
				Eventually(createServiceSession, context_setup.ScaledTimeout(timeout)).Should(Exit(0))
				Eventually(cf.Cf("bind-service", appName, serviceInstanceName), context_setup.ScaledTimeout(timeout)).Should(Exit(0))
				Eventually(cf.Cf("start", appName), context_setup.ScaledTimeout(5*time.Minute)).Should(Exit(0))
				assertAppIsRunning(appName)

				uri := appUri(appName) + "/mykey"
				fmt.Println("Posting to url: ", uri)
				Eventually(runner.Curl("-d", "data=myvalue", "-X", "PUT", uri, "-k"), context_setup.ScaledTimeout(timeout), retryInterval).Should(Say("success"))
				fmt.Println("\n")

				fmt.Println("Getting from url: ", uri)
				Eventually(runner.Curl(uri, "-k"), context_setup.ScaledTimeout(timeout), retryInterval).Should(Say("myvalue"))
				fmt.Println("\n")

				Eventually(cf.Cf("unbind-service", appName, serviceInstanceName), context_setup.ScaledTimeout(timeout)).Should(Exit(0))
				Eventually(cf.Cf("delete-service", "-f", serviceInstanceName), context_setup.ScaledTimeout(timeout)).Should(Exit(0))
			}
			createServiceStdout.CancelDetects()

		})
	}

	Context("for each plan", func() {
		for _, planName := range config.PlanNames {
			AssertLifeCycleBehavior(planName)
		}
	})
})
