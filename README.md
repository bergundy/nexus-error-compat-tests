# Nexus Compatibility Test Harness

A Go test harness for testing Nexus operation compatibility between different versions of Temporal servers and SDKs.

## Architecture

The test harness supports a multi-process architecture for comprehensive compatibility testing:

- **Test Process**: Runs tests, caller worker, and client (single process)
- **Handler Worker**: Separate process hosting Nexus operation handlers
- **Caller Server**: Temporal server for the caller namespace
- **Handler Server**: Temporal server for the handler namespace

## Running Tests

`run.sh` takes four arguments for the following:
1. caller SDK
1. caller server
1. handler server
1. handler SDK

Valid values are `old` or `new`.

See example here:

```bash
bash ./run.sh new old new old
```

The test assertions can be modified using the `TEST_ASSERTIONS` env var, to run with modified assertions set this env
var to `new`.

The `new` SDK and `server` are in:

- The `temporalio/temporal` repo on the `nexus-error-model` branch
- The `temporalio/sdk-go` repo in Quinn's fork at `git@github.com:Quinn-With-Two-Ns/sdk-go.git` on the `nexus-error-changes` branch

**NOTE**: Server processes may get stuck and you may end up falsly thinking you are testing with the right server versions. Run `pkill temporal` to ensure that no lingering server processes are left between runs.

