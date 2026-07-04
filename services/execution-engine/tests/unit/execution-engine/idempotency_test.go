package executor_test

import (
	"testing"

	"github.com/pqap/services/execution-engine/internal/executor"
)

func TestIdempotencyChecker_UniqueIDs(t *testing.T) {
	checker := executor.NewIdempotencyChecker()

	if checker.IsDuplicate("order-1") {
		t.Error("expected order-1 to not be duplicate")
	}

	if checker.IsDuplicate("order-2") {
		t.Error("expected order-2 to not be duplicate")
	}
}

func TestIdempotencyChecker_DuplicateDetection(t *testing.T) {
	checker := executor.NewIdempotencyChecker()

	checker.Mark("order-1")

	if !checker.IsDuplicate("order-1") {
		t.Error("expected order-1 to be duplicate")
	}

	if checker.IsDuplicate("order-2") {
		t.Error("expected order-2 to not be duplicate")
	}
}

func TestIdempotencyChecker_MultipleOrders(t *testing.T) {
	checker := executor.NewIdempotencyChecker()

	checker.Mark("order-1")
	checker.Mark("order-2")
	checker.Mark("order-3")

	if !checker.IsDuplicate("order-1") {
		t.Error("expected order-1 to be duplicate")
	}
	if !checker.IsDuplicate("order-2") {
		t.Error("expected order-2 to be duplicate")
	}
	if !checker.IsDuplicate("order-3") {
		t.Error("expected order-3 to be duplicate")
	}
	if checker.IsDuplicate("order-4") {
		t.Error("expected order-4 to not be duplicate")
	}
}
