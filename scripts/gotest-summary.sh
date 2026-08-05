#!/usr/bin/env bash
# Called by gotestsum --post-run-command; env vars are set by gotestsum.
# Always prints: DONE N tests, X passed, Y failed, Z skipped in <elapsed>
set -euo pipefail
total=${TESTS_TOTAL:-0}
failed=${TESTS_FAILED:-0}
skipped=${TESTS_SKIPPED:-0}
passed=$((total - failed - skipped))
if (( passed < 0 )); then
  passed=0
fi
elapsed=${GOTESTSUM_ELAPSED:-?}
echo "DONE ${total} tests, ${passed} passed, ${failed} failed, ${skipped} skipped in ${elapsed}"
