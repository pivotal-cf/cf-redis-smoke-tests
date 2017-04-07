package cf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	helpersCF "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	"github.com/pivotal-cf/cf-redis-smoke-tests/retry"
)

//CF is a testing wrapper around the cf cli
type CF struct {
	ShortTimeout time.Duration
	LongTimeout  time.Duration
	MaxRetries   int
	RetryBackoff retry.Backoff
}

//API is equivalent to `cf api {endpoint} [--skip-ssl-validation]`
func (cf *CF) API(endpoint string, skipSSLValidation bool) func() {
	apiCmd := []string{"api", endpoint}

	if skipSSLValidation {
		apiCmd = append(apiCmd, "--skip-ssl-validation")
	}

	cfApiFn := func() *gexec.Session {
		return helpersCF.Cf(apiCmd...)
	}

	return func() {
		retry.Session(cfApiFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(cf.MaxRetries).AndBackoff(cf.RetryBackoff).Until(
			retry.Succeeds,
			`{"FailReason": "Failed to target Cloud Foundry"}`,
		)
	}
}

//Auth is equivalent to `cf auth {user} {password}`
func (cf *CF) Auth(user, password string) func() {
	authFn := func() *gexec.Session {
		return helpersCF.Cf("auth", user, password)
	}

	return func() {
		retry.Session(authFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(cf.MaxRetries).AndBackoff(cf.RetryBackoff).Until(
			retry.Succeeds,
			"{\"FailReason\": \"Failed to `cf auth` with target Cloud Foundry\"}",
		)
	}
}

//CreateQuota is equivalent to `cf create-quota {name} [args...]`
func (cf *CF) CreateQuota(name string, args ...string) func() {
	cfArgs := []string{"create-quota", name}
	cfArgs = append(cfArgs, args...)
	createQuotaFn := func() *gexec.Session {
		return helpersCF.Cf(cfArgs...)
	}

	return func() {
		retry.Session(createQuotaFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(cf.MaxRetries).AndBackoff(cf.RetryBackoff).Until(
			retry.Succeeds,
			"{\"FailReason\": \"Failed to `cf create-quota` with target Cloud Foundry\"}",
		)
	}
}

//CreateOrg is equivalent to `cf create-org {org} -q {quota}`
func (cf *CF) CreateOrg(org, quota string) func() {
	createOrgFn := func() *gexec.Session {
		return helpersCF.Cf("create-org", org, "-q", quota)
	}

	return func() {
		retry.Session(createOrgFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(cf.MaxRetries).AndBackoff(cf.RetryBackoff).Until(
			retry.Succeeds,
			`{"FailReason": "Failed to create CF test org"}`,
		)
	}
}

//EnableServiceAccess is equivalent to `cf enable-service-access -o {org} {service-offering}`
//In order to run enable-service-access idempotently we disable-service-access before.
func (cf *CF) EnableServiceAccess(org, service string) func() {
	disableServiceAccessFn := func() *gexec.Session {
		return helpersCF.Cf("disable-service-access", "-o", org, service)
	}
	enableServiceAccessFn := func() *gexec.Session {
		return helpersCF.Cf("enable-service-access", "-o", org, service)
	}

	return func() {
		retry.Session(disableServiceAccessFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(cf.MaxRetries).AndBackoff(cf.RetryBackoff).Until(
			retry.Succeeds,
			`{"FailReason": "Failed to disable service access for CF test org"}`,
		)
		retry.Session(enableServiceAccessFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(cf.MaxRetries).AndBackoff(cf.RetryBackoff).Until(
			retry.Succeeds,
			`{"FailReason": "Failed to enable service access for CF test org"}`,
		)
	}
}

//TargetOrg is equivalent to `cf target -o {org}`
func (cf *CF) TargetOrg(org string) func() {
	targetOrgFn := func() *gexec.Session {
		return helpersCF.Cf("target", "-o", org)
	}
	return func() {
		retry.Session(targetOrgFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(cf.MaxRetries).AndBackoff(cf.RetryBackoff).Until(
			retry.Succeeds,
			`{"FailReason": "Failed to target test org"}`,
		)
	}
}

//TargetOrgAndSpace is equivalent to `cf target -o {org} -s {space}`
func (cf *CF) TargetOrgAndSpace(org, space string) func() {
	targetFn := func() *gexec.Session {
		return helpersCF.Cf("target", "-o", org, "-s", space)
	}

	return func() {
		retry.Session(targetFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(cf.MaxRetries).AndBackoff(cf.RetryBackoff).Until(
			retry.Succeeds,
			`{"FailReason": "Failed to target test org"}`,
		)
	}
}

//CreateSpace is equivalent to `cf create-space {space}`
func (cf *CF) CreateSpace(space string) func() {
	createSpaceFn := func() *gexec.Session {
		return helpersCF.Cf("create-space", space)
	}

	return func() {
		retry.Session(createSpaceFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(cf.MaxRetries).AndBackoff(cf.RetryBackoff).Until(
			retry.Succeeds,
			`{"FailReason": "Failed to create CF test space"}`,
		)
	}
}

//CreateSecurityGroup is equivalent to `cf create-security-group {securityGroup} {configPath}`
func (cf *CF) CreateAndBindSecurityGroup(securityGroup, appName, org, space string) func() {
	return func() {
		appGuid := cf.getAppGuid(appName)

		host, port := cf.getBindingCredentials(appGuid)

		sgFile, err := ioutil.TempFile("", "smoke-test-security-group-")
		Expect(err).NotTo(HaveOccurred())
		defer sgFile.Close()
		defer os.Remove(sgFile.Name())

		sgs := []struct {
			Protocol    string `json:"protocol"`
			Destination string `json:"destination"`
			Ports       string `json:"ports"`
		}{
			{"tcp", host, fmt.Sprintf("%d", port)},
		}

		err = json.NewEncoder(sgFile).Encode(sgs)
		Expect(err).NotTo(HaveOccurred(), `{"FailReason": "Failed to encode security groups"}`)

		Eventually(helpersCF.Cf("create-security-group", securityGroup, sgFile.Name()), cf.ShortTimeout).Should(
			gexec.Exit(0),
			`{"FailReason": "Failed to create security group"}`,
		)

		Eventually(helpersCF.Cf("bind-security-group", securityGroup, org, space), cf.ShortTimeout).Should(
			gexec.Exit(0),
			`{"FailReason": "Failed to bind security group to space"}`,
		)
	}
}

//DeleteSecurityGroup is equivalent to `cf delete-security-group {securityGroup} -f`
func (cf *CF) DeleteSecurityGroup(securityGroup string) func() {
	delSecGroupFn := func() *gexec.Session {
		return helpersCF.Cf("delete-security-group", securityGroup, "-f")
	}

	return func() {
		retry.Session(delSecGroupFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(cf.MaxRetries).AndBackoff(cf.RetryBackoff).Until(
			retry.Succeeds,
			`{"FailReason": "Failed to delete security group"}`,
		)
	}
}

//CreateUser is equivalent to `cf create-user {name} {password}`
func (cf *CF) CreateUser(name, password string) func() {

	createUserFn := func() *gexec.Session {
		return helpersCF.Cf("create-user", name, password)
	}

	// if the user already exists, `cf create-user {name} {password}` is still OK
	return func() {
		retry.Session(createUserFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(cf.MaxRetries).AndBackoff(cf.RetryBackoff).Until(
			retry.Succeeds,
			`{"FailReason": "Failed to create user"}`,
		)
	}
}

//DeleteUser is equivalent to `cf delete-user -f {name}`
func (cf *CF) DeleteUser(name string) func() {
	deleteUserFn := func() *gexec.Session {
		return helpersCF.Cf("delete-user", "-f", name)
	}

	return func() {
		retry.Session(deleteUserFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(cf.MaxRetries).AndBackoff(cf.RetryBackoff).Until(
			retry.Succeeds,
			`{"FailReason": "Failed to delete user"}`,
		)
	}
}

//SetSpaceRole is equivalent to `cf set-space-role {name} {org} {space} {role}`
func (cf *CF) SetSpaceRole(name, org, space, role string) func() {
	setSpaceRoleFn := func() *gexec.Session {
		return helpersCF.Cf("set-space-role", name, org, space, role)
	}

	return func() {
		retry.Session(setSpaceRoleFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(cf.MaxRetries).AndBackoff(cf.RetryBackoff).Until(
			retry.Succeeds,
			`{"FailReason": "Failed to set space role"}`,
		)
	}
}

//Push is equivalent to `cf push {appName} [args...]`
func (cf *CF) Push(appName string, args ...string) func() {
	pushArgs := []string{"push", appName}
	pushArgs = append(pushArgs, args...)

	pushFn := func() *gexec.Session {
		return helpersCF.Cf(pushArgs...)
	}

	return func() {
		retry.Session(pushFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(cf.MaxRetries).AndBackoff(cf.RetryBackoff).Until(
			retry.Succeeds,
			"{\"FailReason\": \"Failed to `cf push` test app\"}",
		)
	}
}

//Delete is equivalent to `cf delete {appName} -f`
func (cf *CF) Delete(appName string) func() {
	deleteAppFn := func() *gexec.Session {
		return helpersCF.Cf("delete", appName, "-f", "-r")
	}

	return func() {
		retry.Session(deleteAppFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(cf.MaxRetries).AndBackoff(cf.RetryBackoff).Until(
			retry.Succeeds,
			"{\"FailReason\": \"Failed to `cf delete` test app\"}",
		)
	}
}

//CreateService is equivalent to `cf create-service {serviceName} {planName} {instanceName}`
func (cf *CF) CreateService(serviceName, planName, instanceName string, skip *bool) func() {
	createServiceFn := func() *gexec.Session {
		return helpersCF.Cf("create-service", serviceName, planName, instanceName)
	}

	var SucceedsOrQuotaReached = func() retry.Condition {
		return func(session *gexec.Session) bool {
			actualSuccess := regexp.MustCompile("OK").Match(session.Out.Contents()) && session.ExitCode() == 0
			failureBecauseQuotaReached :=
				regexp.MustCompile("FAILED").Match(session.Out.Contents()) && (
				// legacy release
				regexp.MustCompile("instance limit for this service has been reached").Match(session.Out.Contents()) ||
					// ODB
					regexp.MustCompile("The quota for this service plan has been exceeded.").Match(session.Out.Contents())) &&
					session.ExitCode() == 1
			if failureBecauseQuotaReached {
				fmt.Printf("No Plan Instances available for testing %s plan\n", planName)
				*skip = true
			}
			return actualSuccess || failureBecauseQuotaReached
		}
	}

	return func() {
		retry.Session(createServiceFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(cf.MaxRetries).AndBackoff(cf.RetryBackoff).Until(SucceedsOrQuotaReached(),
			`{"FailReason": "Failed to create Redis service instance"}`,
		)
		if !(*skip) {
			cf.awaitServiceCreation(instanceName)
		}
	}
}

func (cf *CF) awaitServiceCreation(instanceName string) func() {
	serviceFn := func() *gexec.Session {
		return helpersCF.Cf("service", instanceName)
	}

	// longer retry backoff due to asynchronous creates
	backoff := retry.Exponential(time.Second)
	maxRetries := 10

	return func() {
		retry.Session(serviceFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(maxRetries).AndBackoff(backoff).Until(
			retry.MatchesOutput(regexp.MustCompile("create succeeded")),
			fmt.Sprintf(`{"FailReason": "Failed to create Redis service instance %s"}`, instanceName),
		)
	}
}

//DeleteService is equivalent to `cf delete-service {instanceName} -f`
func (cf *CF) DeleteService(instanceName string) func() {
	deleteFn := func() *gexec.Session {
		return helpersCF.Cf("delete-service", "-f", instanceName)
	}

	return func() {
		retry.Session(deleteFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(cf.MaxRetries).AndBackoff(cf.RetryBackoff).Until(
			retry.Succeeds,
			fmt.Sprintf(`{"FailReason": "Failed to delete service %s"}`, instanceName),
		)
	}
}

func (cf *CF) EnsureServiceInstanceGone(instanceName string) func() {
	serviceFn := func() *gexec.Session {
		return helpersCF.Cf("service", instanceName)
	}

	// longer retry backoff due to asynchronous deletes
	backoff := retry.Exponential(time.Second)
	maxRetries := 10

	return func() {
		retry.Session(serviceFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(maxRetries).AndBackoff(backoff).Until(
			retry.MatchesOutput(regexp.MustCompile(fmt.Sprintf("Service instance %s not found", instanceName))),
			fmt.Sprintf(`{"FailReason": "Failed to make sure service %s does not exist"}`, instanceName),
		)
	}
}

func (cf *CF) EnsureAllServiceInstancesGone() func() {
	serviceFn := func() *gexec.Session {
		return helpersCF.Cf("services")
	}

	// longer retry backoff due to asynchronous deletes
	backoff := retry.Exponential(time.Second)
	maxRetries := 10

	return func() {
		retry.Session(serviceFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(maxRetries).AndBackoff(backoff).Until(
			retry.MatchesOutput(regexp.MustCompile("No services found")),
			`{"FailReason": "Failed to make sure no service instances exist"}`,
		)
	}
}

//BindService is equivalent to `cf bind-service {appName} {instanceName}`
func (cf *CF) BindService(appName, instanceName string) func() {
	bindFn := func() *gexec.Session {
		return helpersCF.Cf("bind-service", appName, instanceName)
	}

	return func() {
		retry.Session(bindFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(cf.MaxRetries).AndBackoff(cf.RetryBackoff).Until(
			retry.Succeeds,
			`{"FailReason": "Failed to bind Redis service instance to test app"}`,
		)
	}
}

//UnbindService is equivalent to `cf unbind-service {appName} {instanceName}`
func (cf *CF) UnbindService(appName, instanceName string) func() {
	unbindFn := func() *gexec.Session {
		return helpersCF.Cf("unbind-service", appName, instanceName)
	}

	return func() {
		retry.Session(unbindFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(cf.MaxRetries).AndBackoff(cf.RetryBackoff).Until(
			retry.MatchesStdOrErrorOutput(regexp.MustCompile(fmt.Sprintf("(OK|Service instance %s not found)", instanceName))),
			fmt.Sprintf(`{"FailReason": "Failed to unbind %s instance from %s"}`, instanceName, appName),
		)
	}
}

//Start is equivalent to `cf start {appName}`
func (cf *CF) Start(appName string) func() {
	startFn := func() *gexec.Session {
		return helpersCF.Cf("start", appName)
	}

	return func() {
		retry.Session(startFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(cf.MaxRetries).AndBackoff(cf.RetryBackoff).Until(
			retry.Succeeds,
			`{"FailReason": "Failed to start test app"}`,
		)
	}
}

//SetEnv is equivalent to `cf set-env {appName} {envVarName} {instanceName}`
func (cf *CF) SetEnv(appName, environmentVariable, instanceName string) func() {
	setEnvFn := func() *gexec.Session {
		return helpersCF.Cf("set-env", appName, environmentVariable, instanceName)
	}

	return func() {
		retry.Session(setEnvFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(cf.MaxRetries).AndBackoff(cf.RetryBackoff).Until(
			retry.Succeeds,
			`{"FailReason": "Failed to set environment variable for test app"}`,
		)
	}
}

//Logout is equivalent to `cf logout`
func (cf *CF) Logout() func() {
	logoutFn := func() *gexec.Session {
		return helpersCF.Cf("logout")
	}

	return func() {
		retry.Session(logoutFn).WithSessionTimeout(cf.ShortTimeout).AndMaxRetries(cf.MaxRetries).AndBackoff(cf.RetryBackoff).Until(
			retry.Succeeds,
			`{"FailReason": "Failed to logout"}`,
		)
	}
}

func (cf *CF) getAppGuid(appName string) string {
	session := helpersCF.Cf("app", "--guid", appName)
	Eventually(session, cf.ShortTimeout).Should(gexec.Exit(0), `{"FailReason": "Failed to retrieve GUID for app"}`)

	return strings.Trim(string(session.Out.Contents()), " \n")
}

func (cf *CF) getBindingCredentials(appGuid string) (string, int) {
	session := helpersCF.Cf("curl", fmt.Sprintf("/v2/apps/%s/service_bindings", appGuid))
	Eventually(session, cf.ShortTimeout).Should(gexec.Exit(0), `{"FailReason": "Failed to retrieve service bindings for app"}`)

	var resp = new(struct {
		Resources []struct {
			Entity struct {
				Credentials struct {
					Host string
					Port int
				}
			}
		}
	})

	err := json.NewDecoder(bytes.NewBuffer(session.Out.Contents())).Decode(resp)
	Expect(err).NotTo(HaveOccurred(), `{"FailReason": "Failed to decode service binding response"}`)
	Expect(resp.Resources).To(HaveLen(1), `{"FailReason": "Invalid binding response, expected exactly one binding"}`)

	host, port := resp.Resources[0].Entity.Credentials.Host, resp.Resources[0].Entity.Credentials.Port
	Expect(host).NotTo(BeEmpty(), `{"FailReason": "Invalid binding, missing host"}`)
	Expect(port).NotTo(BeZero(), `{"FailReason": "Invalid binding, missing port"}`)
	return host, port
}
