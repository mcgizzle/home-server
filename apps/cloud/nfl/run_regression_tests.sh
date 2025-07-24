#!/bin/bash

# NFL Ratings App - Regression Test Runner
# Run this before and after each refactor step to ensure no regressions

echo "🧪 Running NFL Ratings Regression Tests..."
echo ""

# Run the lean edge tests
go test -v -run "TestTemplateData_GoldenMaster|TestHTTP_" | grep -E "(PASS|FAIL|RUN)"

EXIT_CODE=${PIPESTATUS[0]}

if [ $EXIT_CODE -eq 0 ]; then
    echo ""
    echo "✅ All regression tests passed!"
    echo "🔒 Application behavior is stable - safe to proceed with refactoring"
else
    echo ""
    echo "❌ Regression tests failed!"
    echo "🚨 Application behavior has changed - review before continuing"
fi

exit $EXIT_CODE 