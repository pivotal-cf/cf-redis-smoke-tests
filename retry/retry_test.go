package retry_test

import (
	"math"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"github.com/pivotal-cf/cf-redis-smoke-tests/retry"
)

func SucceedingSession() *gexec.Session {
	cmd := exec.Command("echo", "hello")
	s, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	return s
}

func FailingSession() *gexec.Session {
	cmd := exec.Command("ls", "not-a-file-that-exists")
	s, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	return s
}

var _ = Describe("retry", func() {
	Describe("Until", func() {
		Context("when the session succeeds immediatly", func() {
			var (
				attempts = 0

				fn = func() *gexec.Session {
					attempts += 1
					return SucceedingSession()
				}
			)

			It("succeeds without retrying", func() {
				retry.Session(fn).WithMaxRetries(3).AndBackoff(retry.None(10 * time.Millisecond)).Until(retry.Succeeds)

				Expect(attempts).To(Equal(1))
			})
		})

		Context("when the session always fails", func() {
			var (
				attempts int
				failed   bool

				maxRetries = 3

				fn = func() *gexec.Session {
					attempts += 1
					return FailingSession()
				}

				failHandler = func(msg string, i ...int) {
					failed = true
				}
			)

			BeforeEach(func() {
				attempts = 0
				failed = false
			})

			It("calls the fail handler", func() {
				retry.Session(fn).WithMaxRetries(maxRetries).AndBackoff(retry.None(10 * time.Millisecond)).AndFailHandler(failHandler).Until(retry.Succeeds)
				Expect(failed).To(BeTrue())
			})

			It("tries up to maxRetries", func() {
				retry.Session(fn).WithMaxRetries(maxRetries).AndBackoff(retry.None(10 * time.Millisecond)).AndFailHandler(failHandler).Until(retry.Succeeds)
				Expect(attempts).To(Equal(maxRetries + 1))
			})

			It("calls the backoff function", func() {
				backoffCalls := 0

				backoff := func(count uint) time.Duration {
					backoffCalls += 1
					return time.Millisecond
				}

				retry.Session(fn).WithMaxRetries(maxRetries).AndFailHandler(failHandler).AndBackoff(backoff).Until(retry.Succeeds)

				Expect(backoffCalls).To(Equal(attempts))
			})
		})

		Context("when the session eventually succeeds", func() {
			var (
				attempts  = 0
				failCount = 3

				fn = func() *gexec.Session {
					attempts += 1

					if attempts <= failCount {
						return FailingSession()
					}

					return SucceedingSession()
				}
			)

			It("retries until it succeeds", func() {
				retry.Session(fn).WithMaxRetries(failCount * 2).AndBackoff(retry.None(10 * time.Millisecond)).Until(retry.Succeeds)

				Expect(attempts).To(Equal(failCount + 1))
			})
		})
	})

	Context("Backoff", func() {
		var baseline = time.Second

		Describe("None", func() {
			var backoff = retry.None(baseline)

			It("implements no backoff", func() {
				Expect(backoff(0)).To(Equal(time.Duration(0)))

				for i := 1; i < 10; i++ {
					Expect(backoff(uint(i))).To(Equal(baseline))
				}
			})
		})

		Describe("Linear", func() {
			var backoff = retry.Linear(baseline)

			It("implements a linear backoff", func() {
				for i := 0; i < 10; i++ {
					Expect(backoff(uint(i))).To(Equal(time.Duration(i) * baseline))
				}
			})
		})

		Describe("Exponential", func() {
			var backoff = retry.Exponential(baseline)

			It("implements a exponential backoff", func() {
				Expect(backoff(0)).To(Equal(time.Duration(0)))

				for i := 1; i < 10; i++ {
					Expect(backoff(uint(i))).To(Equal(time.Duration(math.Pow(2, float64(i))) * baseline))
				}
			})
		})
	})
})
