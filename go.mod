module service

go 1.13

require (
	github.com/cloudfoundry-incubator/cf-test-helpers v1.0.1-0.20190819181953-621a86920bf4
	github.com/google/uuid v1.1.1 // indirect
	github.com/onsi/ginkgo v1.10.3
	github.com/onsi/gomega v1.7.1
	github.com/pborman/uuid v0.0.0-20180906182336-adf5a7427709
	github.com/pivotal-cf/cf-redis-smoke-tests v0.0.0-20191018125631-1be92aca4769
	golang.org/x/net v0.0.0-20191014212845-da9a3fd4c582 // indirect
	golang.org/x/sys v0.0.0-20191018095205-727590c5006e // indirect
	golang.org/x/text v0.3.2 // indirect
)

replace gopkg.in/fsnotify.v1 v1.4.7 => github.com/fsnotify/fsnotify v1.4.7
