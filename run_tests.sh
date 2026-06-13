#!/bin/bash
# run_tests.sh — Run the full project test suite

set -e

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Running tests for code-search-golang project..."
echo ""

echo "=== Backend Go Tests ==="
cd "$PROJECT_DIR"
go test -v ./...
GO_RESULT=$?

echo ""
echo "=== Frontend Unit Tests (Vitest) ==="
cd "$PROJECT_DIR/frontend"
npm test
FRONTEND_RESULT=$?

echo ""
echo "=== Frontend Type Check ==="
cd "$PROJECT_DIR/frontend"
npx tsc --noEmit
TSC_RESULT=$?

echo ""
if [ $GO_RESULT -eq 0 ] && [ $FRONTEND_RESULT -eq 0 ] && [ $TSC_RESULT -eq 0 ]; then
    echo "✅ All tests passed!"
    echo "✅ Go backend tests"
    echo "✅ Frontend unit tests"
    echo "✅ TypeScript type check"
    exit 0
else
    echo "❌ Some tests failed"
    exit 1
fi