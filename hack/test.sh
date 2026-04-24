#!/bin/bash
set -eux -o pipefail

# check for local installation of go-junit-report otherwise install locally
which go-junit-report || go install github.com/jstemmer/go-junit-report@latest

# check for local installation of gotestsum otherwise install locally
which gotestsum || go install gotest.tools/gotestsum@latest

TEST_RESULTS=${TEST_RESULTS:-test-results}
TEST_FLAGS=${TEST_FLAGS:-}
DIST_DIR=${DIST_DIR:-dist}
GOTESTSUM_FORMAT=testname

if [[ "${TEST_E2E_DEBUG:-}" == "true" ]]; then
	GOTESTSUM_FORMAT=standard-verbose
fi

# Add DIST_DIR to PATH so binaries installed for argo are found first
export PATH="${DIST_DIR}:${PATH}"

if test "${ATHENA_TEST_PARALLELISM:-}" != ""; then
	TEST_FLAGS="$TEST_FLAGS -p $ATHENA_TEST_PARALLELISM"
fi
if test "${ATHENA_TEST_VERBOSE:-}" != ""; then
	TEST_FLAGS="$TEST_FLAGS -v"
fi

mkdir -p "$TEST_RESULTS"

# `TEST_FLAGS` cannot be quoted as empty needs to evaluate to 0 arguments.
# `TEST_FLAGS` cannot be turned into array without backward incompatible change of script input
# shellcheck disable=SC2086
GODEBUG="tarinsecurepath=0,zipinsecurepath=0" \
    gotestsum --rerun-fails-report=rerunreport.txt --junitfile="$TEST_RESULTS/junit.xml" --format="$GOTESTSUM_FORMAT" \
    --rerun-fails="$RERUN_FAILS" --packages="${TEST_MODULE:-$PACKAGES}" -- -cover $TEST_FLAGS "$@"
