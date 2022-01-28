package redis

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/cf-redis-smoke-tests/retry"
)

// App is a helper around reading and writing to redis-example-app endpoints
type App struct {
	uri          string
	timeout      time.Duration
	retryBackoff retry.Backoff
}

// New is the correct way to create a redis.App
func NewApp(uri string, timeout, retryInterval time.Duration) *App {
	return &App{
		uri:          uri,
		timeout:      timeout,
		retryBackoff: retry.None(retryInterval),
	}
}

func (app *App) keyURI(key string) string {
	return fmt.Sprintf("%s/%s", app.uri, key)
}

func (app *App) keyTLSURI(version string, key string) string {
	tlsVersion := strings.Replace(strings.ToLower(version), "tls", "", -1)
	return fmt.Sprintf("%s/tls/%s/%s", app.uri, tlsVersion, key)
}

// IsRunning pings the App
func (app *App) IsRunning(enforced bool, tlsVersions []string) func() {
	return func() {
		var pingURI string
		if enforced {
			pingURI = fmt.Sprintf("%s/", app.keyTLSURI(tlsVersions[0], "ping"))
		} else {
			pingURI = fmt.Sprintf("%s/ping", app.uri)
		}

		curlFn := func() *gexec.Session {
			fmt.Println("Checking that the app is responding at url: ", pingURI)
			return helpers.CurlSkipSSL(true, pingURI)
		}

		retry.Session(curlFn).WithSessionTimeout(app.timeout).AndBackoff(app.retryBackoff).Until(
			retry.MatchesOutput(regexp.MustCompile("key not present")),
			`{"FailReason": "Test app deployed but did not respond in time"}`,
		)
	}
}

func (app *App) Write(shouldFail bool, key, value string) func() {
	return func() {
		curlFn := func() *gexec.Session {
			fmt.Println("Posting to url: ", app.keyURI(key))
			return helpers.CurlSkipSSL(true, "-d", fmt.Sprintf("data=%s", value), "-X", "PUT", app.keyURI(key))
		}
		if shouldFail {
			retry.Session(curlFn).WithSessionTimeout(app.timeout).AndBackoff(app.retryBackoff).Until(
				retry.MatchesOutput(regexp.MustCompile("fail")),
				fmt.Sprintf(`{"FailReason": "If enforced, it should not put %s"}`, app.keyURI(key)),
			)
		} else {
			retry.Session(curlFn).WithSessionTimeout(app.timeout).AndBackoff(app.retryBackoff).Until(
				retry.MatchesOutput(regexp.MustCompile("success")),
				fmt.Sprintf(`{"FailReason": "Failed to put to %s"}`, app.keyURI(key)),
			)
		}
	}
}

//ReadAssert checks that the value for the given key matches expected
func (app *App) ReadAssert(shouldFail bool, key, expectedValue string) func() {
	return func() {
		curlFn := func() *gexec.Session {
			fmt.Printf("\nGetting from url: %s\n", app.keyURI(key))
			return helpers.CurlSkipSSL(true, app.keyURI(key))
		}
		if shouldFail {
			retry.Session(curlFn).WithSessionTimeout(app.timeout).AndBackoff(app.retryBackoff).Until(
				retry.MatchesOutput(regexp.MustCompile("fail")),
				fmt.Sprintf(`{"FailReason": "Failed to get %s"}`, app.keyURI(key)),
			)
		} else {
			retry.Session(curlFn).WithSessionTimeout(app.timeout).AndBackoff(app.retryBackoff).Until(
				retry.MatchesOutput(regexp.MustCompile(expectedValue)),
				fmt.Sprintf(`{"FailReason": "Failed to get %s"}`, app.keyURI(key)),
			)
		}
	}
}

//ReadTLSAssert checks that the value for the given key matches expected
func (app *App) ReadTLSAssert(tlsVersion, key, expectedValue string) func() {
	return func() {
		curlFn := func() *gexec.Session {
			fmt.Printf("\nGetting from url: %s\n", app.keyTLSURI(tlsVersion, key))
			return helpers.CurlSkipSSL(false, app.keyTLSURI(tlsVersion, key))
		}

		retry.Session(curlFn).WithSessionTimeout(app.timeout).AndBackoff(app.retryBackoff).Until(
			retry.MatchesOutput(regexp.MustCompile(expectedValue)),
			fmt.Sprintf(`{"FailReason": "Failed to get expected value of '%s' from %s"}`, expectedValue, app.keyTLSURI(tlsVersion, key)),
		)
	}
}
