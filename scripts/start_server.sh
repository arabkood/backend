#!/bin/bash
set -e

# Enable the service to start on boot (optional, but recommended)
systemctl enable go-app

# Start the service
systemctl start go-app
