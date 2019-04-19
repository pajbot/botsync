#!/bin/bash
echo -n " * Building PostgreSQL instance..."
docker build -t pajlada-botsync-postgres .tests 2>&1 1> /dev/null
echo " DONE"
