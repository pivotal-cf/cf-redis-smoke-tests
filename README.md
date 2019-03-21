## Running smoke tests

1. Find or create the appropriate environment config file in `assets/`.
 This can be copied from `/var/vcap/jobs/smoke-tests/config.json` for cf-redis-broker
 or `/var/vcap/jobs/on-demand-broker-smoke-tests/config.json` for redis-on-demand-broker.

1. Modify the `.envrc` to point to the config file identified.

1. Run `direnv allow`

1. Run `bin/test`

* Note `bin/test` does not run retry tests but that is just testing test helpers for use in waiting for asyncronous processes to complete. All tests are run when called from cf-redis-release and redis-service-adapter-release.