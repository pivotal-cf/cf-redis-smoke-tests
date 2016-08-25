package redis

import (
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf-experimental/cf-test-helpers/runner"
)

//App is a helper around reading and writing to redis-example-app endpoints
type App struct {
	uri           string
	timeout       time.Duration
	retryInterval time.Duration
}

//New is the correct way to create a redis.Redis
func NewApp(uri string, timeout, retryInterval time.Duration) *App {
	return &App{
		uri:           uri,
		timeout:       timeout,
		retryInterval: retryInterval,
	}
}

func (app *App) keyURI(key string) string {
	return fmt.Sprintf("%s/%s", app.uri, key)
}

//IsRunning pings the App
func (app *App) IsRunning() func() {
	return func() {
		pingURI := fmt.Sprintf("%s/ping", app.uri)
		fmt.Println("Checking that the app is responding at url: ", pingURI)
		Eventually(runner.Curl(pingURI, "-k"), app.timeout, app.retryInterval).Should(
			Say("key not present"),
			`{"FailReason": "Test app deployed but did not respond in time"}`,
		)
		fmt.Println()
	}
}

func (app *App) Write(key, value string) func() {
	return func() {
		fmt.Println("Posting to url: ", app.keyURI(key))
		Eventually(
			runner.Curl("-d", fmt.Sprintf("data=%s", value), "-X", "PUT", app.keyURI(key), "-k"),
			app.timeout,
			app.retryInterval,
		).Should(
			Say("success"),
			fmt.Sprintf(`{"FailReason": "Failed to write to %s"}`, app.keyURI(key)),
		)
		fmt.Printf("\nGetting from url: %s\n", app.keyURI(key))
	}
}

//ReadAssert checks that the value for the given key matches expected
func (app *App) ReadAssert(key, expectedValue string) func() {
	return func() {
		Eventually(
			runner.Curl(app.keyURI(key), "-k"),
			app.timeout,
			app.retryInterval,
		).Should(
			Say(expectedValue),
			fmt.Sprintf(`{"FailReason": "Failed to read %s"}`, app.keyURI(key)),
		)
		fmt.Println()
	}
}
