#!/bin/bash

# Bez DATABASE_URL nie ma czego migrować - aplikacja wstanie w trybie degraded
# (tylko /health), żeby platforma nie ubiła instancji. Po dodaniu DATABASE_URL
# i redeployu migracje wykonają się normalnie.
if [ -z "$DATABASE_URL" ]; then
  echo "DATABASE_URL is not set - skipping migrations, starting application in degraded mode..."
  exec ./main
fi

# Uruchom migracje
echo "Running database migrations..."
./main -migrate -dir=./migrations

# Sprawdź, czy migracje zakończyły się sukcesem
if [ $? -eq 0 ]; then
  echo "Migrations completed successfully."
  
  # Uruchom aplikację
  echo "Starting application..."
  exec ./main
else
  echo "Migrations failed. Exiting."
  exit 1
fi
