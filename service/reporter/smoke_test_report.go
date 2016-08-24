package reporter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/types"
)

type failure struct {
	title   string
	message string
}

type SmokeTestReport struct {
	testCount int
	failures  []failure
}

func (report *SmokeTestReport) SpecSuiteWillBegin(
	config config.GinkgoConfigType,
	summary *types.SuiteSummary,
) {
	report.printMessageTitle("Beginning test suite setup")
}

func (report *SmokeTestReport) BeforeSuiteDidRun(summary *types.SetupSummary) {
	if summary.State == types.SpecStateFailed ||
		summary.State == types.SpecStatePanicked ||
		summary.State == types.SpecStateTimedOut {

		report.failures = append(report.failures, failure{
			title:   "Suite setup",
			message: summary.Failure.Message,
		})
	}
	report.printMessageTitle("Finished test suite setup")
}

func (report *SmokeTestReport) SpecWillRun(summary *types.SpecSummary) {
	report.testCount++

	title := report.getTitleFromComponents(summary)
	message := fmt.Sprintf("START %d. %s", report.testCount, title)
	report.printMessageTitle(message)
}

func (report *SmokeTestReport) SpecDidComplete(summary *types.SpecSummary) {
	if summary.Failed() {
		report.failures = append(report.failures, failure{
			title:   summary.ComponentTexts[len(summary.ComponentTexts)-1],
			message: summary.Failure.Message,
		})
	}
	title := report.getTitleFromComponents(summary)
	message := fmt.Sprintf("END %d. %s", report.testCount, title)
	report.printMessageTitle(message)
}

func (report *SmokeTestReport) AfterSuiteDidRun(summary *types.SetupSummary) {
	report.printMessageTitle("Finished suite teardown")
}

func (report *SmokeTestReport) SpecSuiteDidEnd(summary *types.SuiteSummary) {
	matchJSON, err := regexp.Compile(`{"FailReason":\s"(.*)"}`)
	if err != nil {
		fmt.Printf("\nSkipping \"Summarising failure reasons\": %s\n", err.Error())
		return
	}

	if summary.NumberOfFailedSpecs > 0 {
		report.printMessageTitle("Summarising Failures")

		for _, failure := range report.failures {
			fmt.Printf("\n%s\n", failure.title)

			failMessage := matchJSON.FindStringSubmatch(failure.message)
			if failMessage != nil {
				fmt.Printf("> %s\n", failMessage[1])
			}
		}
		fmt.Printf("\nFor help with troubleshooting, visit: https://docs.pivotal.io/redis/smoke-tests.html\n\n")
	}
}

func (report *SmokeTestReport) getTitleFromComponents(summary *types.SpecSummary) (title string) {
	if len(summary.ComponentTexts) > 0 {
		title = summary.ComponentTexts[len(summary.ComponentTexts)-1]
	}
	return
}

func (report *SmokeTestReport) printMessageTitle(message string) {
	border := strings.Repeat("-", len(message)+2)
	fmt.Printf("\n\n|%s|\n", border)
	fmt.Printf("| %s |\n", message)
	fmt.Printf("|%s|\n\n", border)
}
