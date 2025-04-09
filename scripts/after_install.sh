#!/bin/bash
set -e

# Set ownership and permissions for the application binary
chown ec2-user:ec2-user /opt/go-app/main
chmod +x /opt/go-app/main
