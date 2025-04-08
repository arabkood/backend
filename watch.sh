#!/bin/bash

PID_FILE="/tmp/arabkood-api.pid"

umask 0

build_and_run() {
  go run ./cmd/server &
  echo $! >$PID_FILE
}

build_and_run

find . -name '*.go' | entr -n -r sh -c '
  if [ -f '$PID_FILE' ]; then
    kill $(cat '$PID_FILE')
  fi

  echo ""
  echo ""
  echo ""
  echo ""
  echo ""
  echo ""
  echo ""
  echo ""
  echo ""
  echo ""
  echo "=== Restarting Server ==="
  echo ""
  echo ""

  go run ./cmd/server &
  echo $! > '$PID_FILE'
'
