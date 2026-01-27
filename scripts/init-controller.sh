#!/bin/sh

echo "Starting controller service..."
echo "Database initialization is handled by the controller binary. Database path: ${DATABASE_PATH}"

# Ensure data directory exists
mkdir -p "$(dirname "${DATABASE_PATH}")"

exec /app/controller
