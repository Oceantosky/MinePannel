#!/bin/sh
set -e

# Create persistent data directories if they don't exist
mkdir -p /app/data/instances

# Create symbolic links so the app reads/writes to the /app/data volume

if [ ! -e /app/config.json ]; then
    ln -s /app/data/config.json /app/config.json
fi

if [ ! -e /app/data.db ]; then
    ln -s /app/data/data.db /app/data.db
fi

# Remove any existing instances folder (e.g., created accidentally) and link it
rm -rf /app/instances
ln -s /app/data/instances /app/instances

# Execute the main server binary
exec ./minepannel-server
