## Overview

|variant|result|
|-------|------|
|oooo|pass|
|ooon|proto: (line 1:19): unknown field \"nexusSdkFailureErrorInfo\"" error-type=errors.prefixError attempt=15 unexpected-error-attempts=14 lifecycle=ProcessingFailed|
|oono|pass|
|oonn|[see below](#oonn)|
|onoo|Passes with modified assertions [see below](#onoo)|
|onon|Passes with modified assertions|
|onno|Passes with modified assertions|
|onnn|Passes with modified assertions|
|nooo|pass|
|noon|proto: (line 1:19): unknown field \"nexusSdkFailureErrorInfo\"" error-type=errors.prefixError attempt=15 unexpected-error-attempts=14 lifecycle=ProcessingFailed|
|nono|pass|
|nonn|[see below](#nonn)|
|nnoo|Passes with modified assertions [see below](#nnoo)|
|nnon|Passes with modified assertions|
|nnno|Passes with modified assertions|
|nnnn|Passes with old assertions [see below](#nnnn)|

## oonn

```
--- FAIL: TestSyncOperationFailure (3.42s)
    --- PASS: TestSyncOperationFailure/OperationFailure (0.02s)
    --- FAIL: TestSyncOperationFailure/WrappedApplicationError (0.01s)
    --- FAIL: TestSyncOperationFailure/ApplicationError (0.01s)
    --- PASS: TestSyncOperationFailure/HandlerError (0.01s)
    --- PASS: TestSyncOperationFailure/Canceled (0.01s)
    --- FAIL: TestSyncOperationFailure/OperationCancelationWithDetails (0.01s)

    sync_test.go:136:
                Error Trace:    /Users/bergundy/temporal/nexus-error-compat-tests/tests-old/sync_test.go:136
                                                        /Users/bergundy/temporal/nexus-error-compat-tests/tests-old/sync_test.go:166
                Error:          Received unexpected error:
                                payload item 0: unable to decode: json: cannot unmarshal object into Go value of type string
                Test:           TestSyncOperationFailure/OperationCancelationWithDetails
    sync_test.go:70:
                Error Trace:    /Users/bergundy/temporal/nexus-error-compat-tests/tests-old/sync_test.go:70
                                                        /Users/bergundy/temporal/nexus-error-compat-tests/tests-old/sync_test.go:166
                Error:          Not equal:
                                expected: "application error for test"
                                actual  : ""

                                Diff:
                                --- Expected
                                +++ Actual
                                @@ -1 +1 @@
                                -application error for test
                                +
                Test:           TestSyncOperationFailure/WrappedApplicationError
    sync_test.go:94:
                Error Trace:    /Users/bergundy/temporal/nexus-error-compat-tests/tests-old/sync_test.go:94
                                                        /Users/bergundy/temporal/nexus-error-compat-tests/tests-old/sync_test.go:166
                Error:          Not equal:
                                expected: "application error for test"
                                actual  : "handler error (INTERNAL): application error for test (type: TestErrorType, retryable: false)"

                                Diff:
                                --- Expected
                                +++ Actual
                                @@ -1 +1 @@
                                -application error for test
                                +handler error (INTERNAL): application error for test (type: TestErrorType, retryable: false)
                Test:           TestSyncOperationFailure/ApplicationError

```

## onoo

```
--- FAIL: TestSyncOperationFailure (3.39s)
    --- FAIL: TestSyncOperationFailure/OperationFailure (0.01s)
    --- FAIL: TestSyncOperationFailure/WrappedApplicationError (0.01s)
    --- PASS: TestSyncOperationFailure/ApplicationError (0.01s)
    --- PASS: TestSyncOperationFailure/HandlerError (0.01s)
    --- PASS: TestSyncOperationFailure/Canceled (0.01s)
    --- FAIL: TestSyncOperationFailure/OperationCancelationWithDetails (0.01s)

    sync_test.go:136:
                Error Trace:    /Users/bergundy/temporal/nexus-error-compat-tests/tests-old/sync_test.go:136
                                                        /Users/bergundy/temporal/nexus-error-compat-tests/tests-old/sync_test.go:166
                Error:          Received unexpected error:
                                no data available
                Test:           TestSyncOperationFailure/OperationCancelationWithDetails
    sync_test.go:54:
                Error Trace:    /Users/bergundy/temporal/nexus-error-compat-tests/tests-old/sync_test.go:54
                                                        /Users/bergundy/temporal/nexus-error-compat-tests/tests-old/sync_test.go:166
                Error:          Should be in error chain:
                                expected: %!q(**internal.ApplicationError=0x14000134580)
                                in chain:
                Test:           TestSyncOperationFailure/OperationFailure
    sync_test.go:69:
                Error Trace:    /Users/bergundy/temporal/nexus-error-compat-tests/tests-old/sync_test.go:69
                                                        /Users/bergundy/temporal/nexus-error-compat-tests/tests-old/sync_test.go:166
                Error:          Should be in error chain:
                                expected: %!q(**internal.ApplicationError=0x140001905d8)
                                in chain:
                Test:           TestSyncOperationFailure/WrappedApplicationError

```

## nonn

```
--- FAIL: TestSyncOperationFailure (3.40s)
    --- PASS: TestSyncOperationFailure/OperationFailure (0.02s)
    --- FAIL: TestSyncOperationFailure/WrappedApplicationError (0.01s)
    --- FAIL: TestSyncOperationFailure/ApplicationError (0.01s)
    --- PASS: TestSyncOperationFailure/HandlerError (0.01s)
    --- PASS: TestSyncOperationFailure/Canceled (0.01s)
    --- FAIL: TestSyncOperationFailure/OperationCancelationWithDetails (0.01s)

    sync_test.go:70:
                Error Trace:    /Users/bergundy/temporal/nexus-error-compat-tests/tests-new/sync_test.go:70
                                                        /Users/bergundy/temporal/nexus-error-compat-tests/tests-new/sync_test.go:166
                Error:          Not equal:
                                expected: "application error for test"
                                actual  : ""

                                Diff:
                                --- Expected
                                +++ Actual
                                @@ -1 +1 @@
                                -application error for test
                                +
                Test:           TestSyncOperationFailure/WrappedApplicationError
    sync_test.go:94:
                Error Trace:    /Users/bergundy/temporal/nexus-error-compat-tests/tests-new/sync_test.go:94
                                                        /Users/bergundy/temporal/nexus-error-compat-tests/tests-new/sync_test.go:166
                Error:          Not equal:
                                expected: "application error for test"
                                actual  : "handler error (INTERNAL): application error for test (type: TestErrorType, retryable: false)"

                                Diff:
                                --- Expected
                                +++ Actual
                                @@ -1 +1 @@
                                -application error for test
                                +handler error (INTERNAL): application error for test (type: TestErrorType, retryable: false)
                Test:           TestSyncOperationFailure/ApplicationError
    sync_test.go:136:
                Error Trace:    /Users/bergundy/temporal/nexus-error-compat-tests/tests-new/sync_test.go:136
                                                        /Users/bergundy/temporal/nexus-error-compat-tests/tests-new/sync_test.go:166
                Error:          Received unexpected error:
                                payload item 0: unable to decode: json: cannot unmarshal object into Go value of type string
                Test:           TestSyncOperationFailure/OperationCancelationWithDetails
```

## nnoo

```
--- FAIL: TestSyncOperationFailure (3.38s)
    --- FAIL: TestSyncOperationFailure/OperationFailure (0.01s)
    --- FAIL: TestSyncOperationFailure/WrappedApplicationError (0.01s)
    --- PASS: TestSyncOperationFailure/ApplicationError (0.01s)
    --- PASS: TestSyncOperationFailure/HandlerError (0.01s)
    --- PASS: TestSyncOperationFailure/Canceled (0.01s)
    --- FAIL: TestSyncOperationFailure/OperationCancelationWithDetails (0.01s)

    sync_test.go:54:
                Error Trace:    /Users/bergundy/temporal/nexus-error-compat-tests/tests-new/sync_test.go:54
                                                        /Users/bergundy/temporal/nexus-error-compat-tests/tests-new/sync_test.go:166
                Error:          Should be in error chain:
                                expected: %!q(**internal.ApplicationError=0x14000190460)
                                in chain:
                Test:           TestSyncOperationFailure/OperationFailure
    sync_test.go:69:
                Error Trace:    /Users/bergundy/temporal/nexus-error-compat-tests/tests-new/sync_test.go:69
                                                        /Users/bergundy/temporal/nexus-error-compat-tests/tests-new/sync_test.go:166
                Error:          Should be in error chain:
                                expected: %!q(**internal.ApplicationError=0x140001905f0)
                                in chain:
                Test:           TestSyncOperationFailure/WrappedApplicationError
    sync_test.go:136:
                Error Trace:    /Users/bergundy/temporal/nexus-error-compat-tests/tests-new/sync_test.go:136
                                                        /Users/bergundy/temporal/nexus-error-compat-tests/tests-new/sync_test.go:166
                Error:          Received unexpected error:
                                no data available
                Test:           TestSyncOperationFailure/OperationCancelationWithDetails
```

## nnnn

```
--- FAIL: TestSyncOperationFailure (2.66s)
    --- PASS: TestSyncOperationFailure/OperationFailure (0.02s)
    --- FAIL: TestSyncOperationFailure/WrappedApplicationError (0.01s)
    --- PASS: TestSyncOperationFailure/ApplicationError (0.01s)
    --- PASS: TestSyncOperationFailure/HandlerError (0.01s)
    --- PASS: TestSyncOperationFailure/Canceled (0.01s)
    --- FAIL: TestSyncOperationFailure/OperationCancelationWithDetails (0.01s)

    sync_test.go:76:
                Error Trace:    /Users/bergundy/temporal/nexus-error-compat-tests/tests-new/sync_test.go:76
                                                        /Users/bergundy/temporal/nexus-error-compat-tests/tests-new/sync_test.go:166
                Error:          Not equal:
                                expected: "application error for test"
                                actual  : "nexus operation completed unsuccessfully"

                                Diff:
                                --- Expected
                                +++ Actual
                                @@ -1 +1 @@
                                -application error for test
                                +nexus operation completed unsuccessfully
                Test:           TestSyncOperationFailure/WrappedApplicationError
    sync_test.go:140:
                Error Trace:    /Users/bergundy/temporal/nexus-error-compat-tests/tests-new/sync_test.go:140
                                                        /Users/bergundy/temporal/nexus-error-compat-tests/tests-new/sync_test.go:166
                Error:          Should be false
                Test:           TestSyncOperationFailure/OperationCancelationWithDetails
```
