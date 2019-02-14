package redis

import (
	"fmt"
	"regexp"
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

// IsRunning pings the App
func (app *App) IsRunning() func() {
	return func() {
		pingURI := fmt.Sprintf("%s/ping", app.uri)

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

func (app *App) Write(key, value string) func() {
	return func() {
		curlFn := func() *gexec.Session {
			fmt.Println("Posting to url: ", app.keyURI(key))
			return helpers.CurlSkipSSL(true, "-d", fmt.Sprintf("data=%s", value), "-X", "PUT", app.keyURI(key))
		}

		retry.Session(curlFn).WithSessionTimeout(app.timeout).AndBackoff(app.retryBackoff).Until(
			retry.MatchesOutput(regexp.MustCompile("success")),
			fmt.Sprintf(`{"FailReason": "Failed to put to %s"}`, app.keyURI(key)),
		)
	}
}

//ReadAssert checks that the value for the given key matches expected
func (app *App) ReadAssert(key, expectedValue string) func() {
	return func() {
		curlFn := func() *gexec.Session {
			fmt.Printf("\nGetting from url: %s\n", app.keyURI(key))
			return helpers.CurlSkipSSL(true, app.keyURI(key))
		}

		retry.Session(curlFn).WithSessionTimeout(app.timeout).AndBackoff(app.retryBackoff).Until(
			retry.MatchesOutput(regexp.MustCompile(expectedValue)),
			fmt.Sprintf(`{"FailReason": "Failed to get %s"}`, app.keyURI(key)),
		)
	}
}
