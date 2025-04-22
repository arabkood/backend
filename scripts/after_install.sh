#!/bin/bash
set -e

# Set ownership and permissions for the application binary
chown ec2-user:ec2-user /opt/go-app/main
chmod +x /opt/go-app/main

mv /opt/go-app/config.dev.yaml /opt/go-app/config.yaml
chown ec2-user:ec2-user /opt/go-app/config.yaml

setcap 'cap_net_bind_service=+ep' /opt/go-app/main
