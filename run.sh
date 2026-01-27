#!/bin/bash

set -exuo pipefail

CALLER_WORKER=$1
CALLER_SERVER=$2
HANDLER_SERVER=$3
HANDLER_WORKER=$4

if test "$HANDLER_SERVER" = "new"; then
  export HANDLER_SERVER_CMD="go run -C ../../temporal ./cmd/server --config-file ${PWD}/development-sqlite-alt-port.yaml --allow-no-auth start"
fi

if test "$CALLER_SERVER" = "new"; then
	export CALLER_SERVER_CMD="go run -C ../../temporal ./cmd/server --config-file config/development-sqlite.yaml --allow-no-auth start"
fi

export HANDLER_WORKER_CMD="go run -C ../worker-${HANDLER_WORKER} ."

go test -C "./tests-${CALLER_WORKER}/" -timeout 15s -v -run '^(TestAsyncOperationFailure)$'
sleep 2
go test -C "./tests-${CALLER_WORKER}/" -timeout 15s -v -run '^(TestSyncOperationFailure)$'
