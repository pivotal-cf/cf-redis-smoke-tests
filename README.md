## Running smoke tests

1. Find or create the appropriate environment config file in `assets/`

1. Modify the `.envrc` to point to the config file identified. Can use a toolsmiths environment cf api credentials

1. Run `direnv allow`

1. Run `bin/test`

* Note `bin/test` does not run retry tests but that is just testing test helpers for use in waiting for asyncronous processes to complete. All tests are run when called from cf-redis-release and redis-service-adapter-release.