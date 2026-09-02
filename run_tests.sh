#!/bin/bash
# run_tests.sh — Run the full project test suite

set -e

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Running tests for code-search-golang project..."
echo ""

echo "=== Backend Go Tests ==="
cd "$PROJECT_DIR"
set +e
# Mirrors CI: -race catches the concurrency bugs the cache tests exist for, and
# -covermode=atomic is required alongside it. Timeout matches CI's 600s.
go test -race -covermode=atomic -coverprofile=coverage.out -timeout 600s ./...
GO_RESULT=$?
set -e

echo ""
echo "=== Frontend Unit Tests (Vitest) ==="
cd "$PROJECT_DIR/frontend"
set +e
npm test
FRONTEND_RESULT=$?
set -e

echo ""
echo "=== Frontend Type Check ==="
cd "$PROJECT_DIR/frontend"
set +e
# vue-tsc, not tsc: plain tsc skips .vue SFCs, so a local pass could hide the
# template type errors CI's `vue-tsc --noEmit` step catches.
npx vue-tsc --noEmit
TSC_RESULT=$?
set -e

# End-to-end UX flows (search -> results -> preview, symbol search) run against
# the Vue frontend with a mocked Wails backend. Opt-in via RUN_E2E=1 because it
# needs a browser (system Chrome) and starts a vite server; keep the default
# suite hermetic and fast. Uses `set +e` so a failure is captured, not aborted.
E2E_RESULT=0
if [ "${RUN_E2E:-0}" = "1" ]; then
    echo ""
    echo "=== Frontend E2E (Playwright) ==="
    cd "$PROJECT_DIR/frontend"
    set +e
    npm run test:e2e
    E2E_RESULT=$?
    set -e
fi

echo ""
if [ $GO_RESULT -eq 0 ] && [ $FRONTEND_RESULT -eq 0 ] && [ $TSC_RESULT -eq 0 ] && [ $E2E_RESULT -eq 0 ]; then
    echo "✅ All tests passed!"
    echo "✅ Go backend tests"
    echo "✅ Frontend unit tests"
    echo "✅ TypeScript type check"
    [ "${RUN_E2E:-0}" = "1" ] && echo "✅ Frontend E2E (Playwright)"
    exit 0
else
    echo "❌ Some tests failed"
    exit 1
fi