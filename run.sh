#!/bin/bash

set -exuo pipefail

CALLER_WORKER=$1
CALLER_SERVER=$2
HANDLER_SERVER=$3
HANDLER_WORKER=$4

if test "$HANDLER_SERVER" = "new"; then
  export NEW_HANDLER_SERVER=true
fi

if test "$CALLER_SERVER" = "new"; then
	export NEW_CALLER_SERVER=true
fi

if test "$HANDLER_WORKER" = "new"; then
  export NEW_HANDLER_WORKER=true
fi

# Sanity checks to verify no old servers are left around
if nc -z localhost 7233; then
  echo "server already listening on port 7233, killing in the name of"
  lsof -i :7233 -P | grep LISTEN | awk '{print $2}' | xargs kill
fi
if nc -z localhost 8233; then
  echo "server already listening on port 8233, killing in the name of"
  lsof -i :8233 -P | grep LISTEN | awk '{print $2}' | xargs kill
fi

go test -C "./tests-${CALLER_WORKER}/" -timeout 15s -v -run '^(TestAsyncOperationFailure)$'
sleep 2
go test -C "./tests-${CALLER_WORKER}/" -timeout 15s -v -run '^(TestSyncOperationFailure)$'
