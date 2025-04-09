#!/bin/bash
set -e # Exit immediately if a command exits with a non-zero status.

# Stop the service if it exists and is running
if systemctl is-active --quiet go-app; then
  systemctl stop go-app
fi

# Clean the deployment directory
rm -rf /opt/go-app/*

# Optional: Ensure the directory exists
mkdir -p /opt/go-app
