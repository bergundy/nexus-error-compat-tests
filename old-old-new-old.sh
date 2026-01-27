#!/bin/bash

set -euo pipefail

# Handler server configuration
export HANDLER_SERVER_CMD="go run -C $HOME/temporal/temporal ./cmd/server --config-file config/development-sqlite-alt-port.yaml --allow-no-auth start"

# Test timeout
export TEST_TIMEOUT="60s"

# Run tests
go test -v ./tests/
