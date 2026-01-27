#!/bin/bash

set -euo pipefail

export TEST_ASSERTIONS=new

# Caller server configuration (example)
export CALLER_SERVER_CMD="go run -C $HOME/temporal/temporal ./cmd/server --config-file config/development-sqlite.yaml --allow-no-auth start"

# Test timeout
export TEST_TIMEOUT="60s"

# Run tests
go test -v ./tests/
