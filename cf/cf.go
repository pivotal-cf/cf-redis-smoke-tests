package cf

import (
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	helpersCF "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
)

//CF is a testing wrapper around the cf cli
type CF struct {
	ShortTimeout time.Duration
	LongTimeout  time.Duration
}

//API is equivalent to `cf api {endpoint} [--skip-ssl-validation]`
func (cf *CF) API(endpoint string, skipSSLValidation bool) func() {
	return func() {
		apiCmd := []string{"api", endpoint}

		if skipSSLValidation {
			apiCmd = append(apiCmd, "--skip-ssl-validation")
		}

		Eventually(helpersCF.Cf(apiCmd...), cf.ShortTimeout).Should(
			gexec.Exit(0),
			`{"FailReason": "Failed to target Cloud Foundry"}`,
		)
	}
}

//Auth is equivalent to `cf auth {user} {password}`
func (cf *CF) Auth(user, password string) func() {
	return func() {
		Eventually(helpersCF.Cf("auth", user, password), cf.ShortTimeout).Should(
			gexec.Exit(0),
			"{\"FailReason\": \"Failed to `cf auth` with target Cloud Foundry\"}",
		)
	}
}

//CreateQuota is equivalent to `cf create-quota {name} [args...]`
func (cf *CF) CreateQuota(name string, args ...string) func() {
	return func() {
		cfArgs := []string{"create-quota", name}
		cfArgs = append(cfArgs, args...)
		Eventually(helpersCF.Cf(cfArgs...), cf.ShortTimeout).Should(
			gexec.Exit(0),
			"{\"FailReason\": \"Failed to `cf create-quota` with target Cloud Foundry\"}",
		)
	}
}

//CreateOrg is equivalent to `cf create-org {org} -q {quota}`
func (cf *CF) CreateOrg(org, quota string) func() {
	return func() {
		Eventually(helpersCF.Cf("create-org", org, "-q", quota), cf.ShortTimeout).Should(
			gexec.Exit(0),
			`{"FailReason": "Failed to create CF test org"}`,
		)
	}
}

//EnableServiceAccess is equivalent to `cf enable-service-access -o {org} {service-offering}`
func (cf *CF) EnableServiceAccess(org, service string) func() {
	return func() {
		Eventually(helpersCF.Cf("enable-service-access", "-o", org, service), cf.ShortTimeout).Should(
			gexec.Exit(0),
			`{"FailReason": "Failed to enable service access for CF test org"}`,
		)
	}
}

//TargetOrg is equivalent to `cf target -o {org}`
func (cf *CF) TargetOrg(org string) func() {
	return func() {
		Eventually(helpersCF.Cf("target", "-o", org), cf.ShortTimeout).Should(
			gexec.Exit(0),
			`{"FailReason": "Failed to target test org"}`,
		)
	}
}

//TargetOrgAndSpace is equivalent to `cf target -o {org} -s {space}`
func (cf *CF) TargetOrgAndSpace(org, space string) func() {
	return func() {
		Eventually(helpersCF.Cf("target", "-o", org, "-s", space), cf.ShortTimeout).Should(
			gexec.Exit(0),
			`{"FailReason": "Failed to target test org"}`,
		)
	}
}

//CreateSpace is equivalent to `cf create-space {space}`
func (cf *CF) CreateSpace(space string) func() {
	return func() {
		Eventually(helpersCF.Cf("create-space", space), cf.ShortTimeout).Should(
			gexec.Exit(0),
			`{"FailReason": "Failed to create CF test space"}`,
		)
	}
}

//CreateSecurityGroup is equivalent to `cf create-security-group {securityGroup} {configPath}`
func (cf *CF) CreateSecurityGroup(securityGroup, configPath string) func() {
	return func() {
		Eventually(helpersCF.Cf("create-security-group", securityGroup, configPath), cf.ShortTimeout).Should(
			gexec.Exit(0),
			`{"FailReason": "Failed to create security group"}`,
		)
	}
}

//DeleteSecurityGroup is equivalent to `cf delete-security-group {securityGroup} -f`
func (cf *CF) DeleteSecurityGroup(securityGroup string) func() {
	return func() {
		Eventually(helpersCF.Cf("delete-security-group", securityGroup, "-f"), cf.ShortTimeout).Should(
			gexec.Exit(0),
			`{"FailReason": "Failed to delete security group"}`,
		)
	}
}

//BindSecurityGroup is equivalent to `cf bind-security-group {securityGroup} {org} {space}`
func (cf *CF) BindSecurityGroup(securityGroup, org, space string) func() {
	return func() {
		Eventually(helpersCF.Cf("bind-security-group", securityGroup, org, space), cf.ShortTimeout).Should(
			gexec.Exit(0),
			`{"FailReason": "Failed to bind security group to space"}`,
		)
	}
}

//CreateUser is equivalent to `cf create-user {name} {password}`
func (cf *CF) CreateUser(name, password string) func() {
	return func() {
		createUserCmd := helpersCF.Cf("create-user", name, password)
		Eventually(createUserCmd, cf.ShortTimeout).Should(gexec.Exit())
		if createUserCmd.ExitCode() != 0 {
			Expect(createUserCmd.Out).To(
				gbytes.Say("scim_resource_already_exists"),
				`{"FailReason": "Failed to create user"}`,
			)
		}
	}
}

//DeleteUser is equivalent to `cf delete-user -f {name}`
func (cf *CF) DeleteUser(name string) func() {
	return func() {
		Eventually(helpersCF.Cf("delete-user", "-f", name), cf.ShortTimeout).Should(
			gexec.Exit(0),
			`{"FailReason": "Failed to delete user"}`,
		)
	}
}

//SetSpaceRole is equivalent to `cf set-space-role {name} {org} {space} {role}`
func (cf *CF) SetSpaceRole(name, org, space, role string) func() {
	return func() {
		Eventually(helpersCF.Cf("set-space-role", name, org, space, role), cf.ShortTimeout).Should(
			gexec.Exit(0),
			`{"FailReason": "Failed to set space role"}`,
		)
	}
}

//Push is equivalent to `cf push {appName} [args...]`
func (cf *CF) Push(appName string, args ...string) func() {
	pushArgs := []string{"push", appName}
	pushArgs = append(pushArgs, args...)
	return func() {
		Eventually(helpersCF.Cf(pushArgs...), cf.ShortTimeout).Should(
			gexec.Exit(0),
			"{\"FailReason\": \"Failed to `cf push` test app\"}",
		)
	}
}

//Delete is equivalent to `cf delete {appName} -f`
func (cf *CF) Delete(appName string) func() {
	return func() {
		Eventually(helpersCF.Cf("delete", appName, "-f", "-r"), cf.ShortTimeout).Should(
			gexec.Exit(0),
			"{\"FailReason\": \"Failed to `cf delete` test app\"}",
		)
	}
}

//CreateService is equivalent to `cf create-service {serviceName} {planName} {instanceName}`
func (cf *CF) CreateService(serviceName, planName, instanceName string, skip *bool) func() {
	return func() {
		session := helpersCF.Cf("create-service", serviceName, planName, instanceName)
		session.Wait(cf.ShortTimeout)
		createServiceStdout := session.Out

		defer createServiceStdout.CancelDetects()
		select {
		case <-createServiceStdout.Detect("FAILED"):
			Eventually(session, cf.ShortTimeout).Should(
				gbytes.Say("instance limit for this service has been reached"),
				`{"FailReason": "Failed to bind Redis service instance to test app"}`,
			)
			Eventually(session, cf.ShortTimeout).Should(gexec.Exit(1))
			fmt.Printf("No Plan Instances available for testing %s plan\n", planName)
			*skip = true
		case <-createServiceStdout.Detect("OK"):
			Eventually(session, cf.ShortTimeout).Should(
				gexec.Exit(0),
				`{"FailReason": "Failed to create Redis service instance"}`,
			)
		}
	}
}

//DeleteService is equivalent to `cf delete-service {instanceName} -f`
func (cf *CF) DeleteService(instanceName string) func() {
	return func() {
		Eventually(helpersCF.Cf("delete-service", "-f", instanceName), cf.ShortTimeout).Should(
			gexec.Exit(0),
			fmt.Sprintf(`{"FailReason": "Failed to delete service %s"}`, instanceName),
		)
	}
}

//BindService is equivalent to `cf bind-service {appName} {instanceName}`
func (cf *CF) BindService(appName, instanceName string) func() {
	return func() {
		Eventually(helpersCF.Cf("bind-service", appName, instanceName), cf.ShortTimeout).Should(
			gexec.Exit(0),
			`{"FailReason": "Failed to bind Redis service instance to test app"}`,
		)
	}
}

//UnbindService is equivalent to `cf unbind-service {appName} {instanceName}`
func (cf *CF) UnbindService(appName, instanceName string) func() {
	return func() {
		Eventually(helpersCF.Cf("unbind-service", appName, instanceName), cf.ShortTimeout).Should(
			gexec.Exit(0),
			fmt.Sprintf(`{"FailReason": "Failed to unbind %s instance from %s"}`, instanceName, appName),
		)
	}
}

//Start is equivalent to `cf start {appName}`
func (cf *CF) Start(appName string) func() {
	return func() {
		Eventually(helpersCF.Cf("start", appName), cf.LongTimeout).Should(
			gexec.Exit(0),
			`{"FailReason": "Failed to start test app"}`,
		)
	}
}

//Logout is equivalent to `cf logout`
func (cf *CF) Logout() func() {
	return func() {
		Eventually(helpersCF.Cf("logout")).Should(
			gexec.Exit(0),
			`{"FailReason": "Failed to logout"}`,
		)
	}
}
