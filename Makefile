.SILENT:

build:
	@.tests/build-docker-postgres.sh

run:
	@.tests/run-postgres-instance.sh

integration-test:
	@echo " * Running tests"
	-go test -v . -tags integration

kill:
	docker kill pajlada-botsync-postgres-integration-test

test: build run integration-test kill
