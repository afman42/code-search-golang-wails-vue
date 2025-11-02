#!/bin/bash

echo "Running tests for enhanced code-search-golang project..."

echo ""
echo "=== Backend Go Tests ==="
cd /home/afman42/repo/code-search-golang
go test -v
GO_RESULT=$?

echo ""
echo "=== Frontend Build Test ==="
cd /home/afman42/repo/code-search-golang/frontend
npm run build
FRONTEND_RESULT=$?

echo ""
echo "=== Build Application Test ==="
cd /home/afman42/repo/code-search-golang
wails build
BUILD_RESULT=$?

echo ""
if [ $GO_RESULT -eq 0 ] && [ $FRONTEND_RESULT -eq 0 ] && [ $BUILD_RESULT -eq 0 ]; then
    echo "✅ All tests passed!"
    echo "✅ Go backend tests are working"
    echo "✅ Frontend builds successfully"
    echo "✅ Full application builds successfully"
    echo ""
    echo "🚀 Enhanced Features:"
    echo "   - Directory validation function"
    echo "   - File size limits to improve performance"
    echo "   - Result truncation for large searches"
    echo "   - Regex search option"
    echo "   - Recent searches history"
    echo "   - Enhanced UI with search result highlighting"
    echo "   - Copy to clipboard functionality"
    echo "   - Better search result display"
    exit 0
else
    echo "❌ Some tests failed"
    exit 1
fi