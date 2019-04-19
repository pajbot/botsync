build:
	cd .tests; docker build -t pajlada-botsync-postgres .

run:
	-docker kill pajlada-botsync-postgres-integration-test
	docker run --name pajlada-botsync-postgres-integration-test -d --rm -v "${CURDIR}/.tests/postgresql.conf":/etc/postgresql/postgresql.conf -p 5433:5433 pajlada-botsync-postgres -c 'config_file=/etc/postgresql/postgresql.conf'

integration-test:
	-go test -v . -tags integration

kill:
	docker kill pajlada-botsync-postgres-integration-test

test: build run integration-test kill
