#!/bin/bash
set -u

if [ -z "${DATABASE_URL:-}" ]; then
  echo "DATABASE_URL is not set. Exiting."
  exit 1
fi

# Uruchom migracje
echo "Running database migrations..."
if ! ./main -migrate -dir=./migrations; then
  echo "Migrations failed. Exiting."
  exit 1
fi

echo "Migrations completed successfully."

# Uruchom aplikację - exec, żeby SIGTERM z platformy trafił do procesu Go
# i zadziałało graceful shutdown zamiast twardego ubicia basha.
echo "Starting application..."
exec ./main
