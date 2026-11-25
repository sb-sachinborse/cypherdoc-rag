#!/bin/sh
# Creates .env with a random database password. Safe to re-run.
set -eu
cd "$(dirname "$0")/.."
if [ -f .env ]; then echo ".env already exists, leaving it alone"; exit 0; fi
pw=$(head -c 24 /dev/urandom | od -An -tx1 | tr -d ' \n')
sed "s/^POSTGRES_PASSWORD=.*/POSTGRES_PASSWORD=$pw/" .env.example > .env
chmod 600 .env
echo "Created .env with a random database password"
