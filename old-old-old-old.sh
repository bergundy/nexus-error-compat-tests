#!/bin/bash

set -euo pipefail

# Test timeout
export TEST_TIMEOUT="60s"

# Run tests
go test -v ./tests/
