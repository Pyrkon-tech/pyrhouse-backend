#!/bin/bash

# Uruchom migracje
echo "Running database migrations..."
./main -migrate -dir=./migrations

# Sprawdź, czy migracje zakończyły się sukcesem
if [ $? -eq 0 ]; then
  echo "Migrations completed successfully."
  
  # Uruchom aplikację
  echo "Starting application..."
  ./main
else
  echo "Migrations failed. Exiting."
  exit 1
fi 