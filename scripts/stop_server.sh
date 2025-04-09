#!/bin/bash
set -e

# Stop the service if it exists and is running
if systemctl is-active --quiet go-app; then
  systemctl stop go-app
fi

# Optional: Disable the service from starting on boot if you want it stopped completely
# systemctl disable go-app
