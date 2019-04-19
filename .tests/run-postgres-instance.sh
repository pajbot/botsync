#!/bin/bash
docker kill pajlada-botsync-postgres-integration-test &>/dev/null
echo -n " * Starting PostgreSQL instance..."
docker run --name pajlada-botsync-postgres-integration-test -d --rm -v "$(pwd)/.tests/postgresql.conf":/etc/postgresql/postgresql.conf -p 5433:5433 pajlada-botsync-postgres -c 'config_file=/etc/postgresql/postgresql.conf' &> /dev/null
until pg_isready -h127.0.0.1 -p5433 &>/dev/null; do
    sleep 0.5
done
echo " DONE"
