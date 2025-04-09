#!/bin/bash
set -e

# Wait a few seconds for the service to start
sleep 5

# Check if the service is active
if ! systemctl is-active --quiet go-app; then
  echo "Service go-app is not active."
  exit 1
fi

# Check the health endpoint (adjust port/path if needed)
curl -f http://localhost:8080/health || exit 1

echo "Service validation successful!"
exit 0
