package cf_helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/craigfurman/herottp"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/gomega"
)

type binding struct {
	Credentials interface{} `json:"credentials"`
	Name        string      `json:"name"`
}

var certIgnoringHTTPClient = herottp.New(herottp.Config{
	DisableTLSCertificateVerification: true,
	Timeout: 30 * time.Second,
})

func findURL(cliOutput string) string {
	for _, line := range strings.Split(cliOutput, "\n") {
		if strings.HasPrefix(line, "urls:") {
			return strings.Fields(line)[1]
		}
	}
	return ""
}

func PushAndBindApp(appName, serviceName, testAppPath string) string {
	Eventually(cf.Cf("push", "-p", testAppPath, "-f", filepath.Join(testAppPath, "manifest.yml"), "--no-start", appName), CfTimeout).Should(gexec.Exit(0))
	Eventually(cf.Cf("bind-service", appName, serviceName), CfTimeout).Should(gexec.Exit(0))

	// The first time apps start, it is very slow as the buildpack downloads runtimes and caches them
	Eventually(cf.Cf("start", appName), LongCfTimeout).Should(gexec.Exit(0))
	appDetails := cf.Cf("app", appName)
	Eventually(appDetails, CfTimeout).Should(gexec.Exit(0))
	appDetailsOutput := string(appDetails.Buffer().Contents())
	testAppURL := findURL(appDetailsOutput)
	Expect(testAppURL).NotTo(BeEmpty())
	return testAppURL
}

func PutToTestApp(testAppURL, key, value string) {
	putReq, err := http.NewRequest(
		"PUT",
		fmt.Sprintf("https://%s/%s", testAppURL, key),
		strings.NewReader(fmt.Sprintf("data=%s", value)),
	)
	Expect(err).ToNot(HaveOccurred())
	putReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	makeAndCheckHttpRequest(putReq)
}

func GetFromTestApp(testAppURL, key string) string {
	getReq, err := http.NewRequest(
		"GET",
		fmt.Sprintf("https://%s/%s", testAppURL, key),
		nil,
	)
	Expect(err).ToNot(HaveOccurred())
	return makeAndCheckHttpRequest(getReq)
}

func PushToTestAppQueue(testAppURL, queueName, message string) {
	postReq, err := http.NewRequest(
		"POST",
		fmt.Sprintf("https://%s/queues/%s", testAppURL, queueName),
		strings.NewReader(message),
	)
	Expect(err).ToNot(HaveOccurred())
	makeAndCheckHttpRequest(postReq)
}

func PopFromTestAppQueue(testAppURL, queueName string) string {
	getReq, err := http.NewRequest(
		"GET",
		fmt.Sprintf("https://%s/queues/%s", testAppURL, queueName),
		nil,
	)
	Expect(err).ToNot(HaveOccurred())
	return makeAndCheckHttpRequest(getReq)
}

func appEnv(appName string) (io.Reader, error) {
	session := cf.Cf(
		"curl",
		fmt.Sprintf("/v2/apps/%s/env", appGUID(appName)),
	)

	Eventually(session, CfTimeout).Should(gexec.Exit(0))
	return bytes.NewReader(session.Out.Contents()), nil
}

func appBinding(appName, serviceName string) (*binding, error) {
	env, err := appEnv(appName)
	if err != nil {
		return nil, err
	}

	var envBindings struct {
		Env struct {
			Services map[string][]binding `json:"VCAP_SERVICES"`
		} `json:"system_env_json"`
	}

	if err = json.NewDecoder(env).Decode(&envBindings); err != nil {
		return nil, err
	}

	if bindings, found := envBindings.Env.Services[serviceName]; found {
		return &bindings[0], nil
	}

	return nil, fmt.Errorf("app not bound to service %q", serviceName)
}

type credentials struct {
	Password string `json:"password"`
}

func AppBindingCreds(appName, serviceName string) (interface{}, error) {
	b, err := appBinding(appName, serviceName)
	if err != nil {
		return "", err
	}

	return b.Credentials, nil
}

func guid(name, typ string) string {
	session := cf.Cf(typ, name, "--guid")
	Eventually(session, CfTimeout).Should(gexec.Exit(0))
	return strings.Trim(string(session.Out.Contents()), " \n")
}

func appGUID(appName string) string {
	return guid(appName, "app")
}

func ServiceInstanceGUID(serviceName string) string {
	return guid(serviceName, "service")
}

func makeAndCheckHttpRequest(req *http.Request) string {
	resp, err := certIgnoringHTTPClient.Do(req)
	Expect(err).ToNot(HaveOccurred())
	defer resp.Body.Close()
	bodyContent, err := ioutil.ReadAll(resp.Body)
	Expect(err).ToNot(HaveOccurred())
	ginkgo.GinkgoWriter.Write([]byte(fmt.Sprintf(
		"response from %s %s: %d\n----------------------------------------\n%s\n----------------------------------------\n",
		req.Method,
		req.URL.String(),
		resp.StatusCode,
		bodyContent,
	)))
	Expect(resp.StatusCode).To(BeNumerically("<", 300))
	return string(bodyContent)
}
