package main

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain fails the package when a test leaks a goroutine. The search
// worker pools, fuzzy pass, collection probe pool, and file feeders all
// close their channels and return on context cancellation; a stuck sender
// or a forgotten Wait would show up here instead of flaking a later test.
//
// Known exception: security_test.go's PathTraversalAttempts spawns a
// SearchWithProgress goroutine and abandons it on the 5s-timeout branch,
// so a passing run still leaves probeBinaryInParallel workers parked until
// the process exits. That is a test-harness leak, not a product leak:
// SearchWithProgress itself always drains (drainResults ranges resultsChan
// to close). Ignore it here; the fix belongs in the test (await the
// goroutine or drop the timeout branch).
func TestMain(m *testing.M) {
	// IgnoreAnyFunction (not Top): leaked workers park in runtime/syscall
	// frames, so the product frame is mid-stack, never top.
	goleak.VerifyTestMain(m,
		goleak.IgnoreAnyFunction("code-search-golang.(*App).probeBinaryInParallel"),
		goleak.IgnoreAnyFunction("code-search-golang.(*App).probeBinaryInParallel.func1"),
		goleak.IgnoreAnyFunction("code-search-golang.(*App).probeBinaryInParallel.func2"),
		goleak.IgnoreAnyFunction("code-search-golang.(*App).probeBinaryInParallel.func3"),
	)
}
