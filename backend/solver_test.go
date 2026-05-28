package main

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestSolveCifrasExactTwoNumbers(t *testing.T) {
	ctx := context.Background()
	found, expr := SolveCifras(ctx, []int{2, 3}, 6, 1_000_000)
	if !found {
		t.Fatalf("expected exact solution for 2,3 -> 6, got found=%v", found)
	}
	if !strings.Contains(expr, "6") {
		t.Fatalf("expected result 6 in expression, got: %s", expr)
	}
}

func TestSolveCifrasExactThreeNumbers(t *testing.T) {
	ctx := context.Background()
	found, expr := SolveCifras(ctx, []int{10, 5, 2}, 100, 1_000_000)
	if !found {
		t.Fatalf("expected exact solution for 10,5,2 -> 100, got found=%v", found)
	}
	if !strings.Contains(expr, "100") {
		t.Fatalf("expected result 100 in expression, got: %s", expr)
	}
}

func TestSolveCifrasContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	time.Sleep(5 * time.Millisecond)
	found, _ := SolveCifras(ctx, []int{25, 50, 75, 100, 3, 6}, 952, 1_000_000)
	if found {
		t.Log("solver found exact solution before context expired (acceptable)")
	}
}

func TestSolveCifrasNoExact(t *testing.T) {
	ctx := context.Background()
	found, expr := SolveCifras(ctx, []int{1, 1, 1, 1, 1, 1}, 999, 1_000_000)
	if found {
		t.Fatalf("expected no exact solution for all ones -> 999, got found=%v", found)
	}
	if expr == "" {
		t.Fatalf("expected some approximation expression even if not exact")
	}
}

func TestSolveCifrasMaxStepsLimit(t *testing.T) {
	ctx := context.Background()
	// Use a very small maxSteps to force early termination
	found, _ := SolveCifras(ctx, []int{25, 50, 75, 100, 3, 6}, 952, 10)
	// Should return quickly without necessarily finding exact
	_ = found
}

func TestSolveCifrasSubtracBug(t *testing.T) {
	ctx := context.Background()
	found, expr := SolveCifras(ctx, []int{10, 50}, 40, 1000)
	if !found {
		t.Fatalf("expected exact solution for 10, 50 -> 40, got found=%v", found)
	}
	if !strings.Contains(expr, "50 - 10 = 40") {
		t.Fatalf("expected 50 - 10 = 40, got: %s", expr)
	}
}

func TestSolveCifrasDivisionBug(t *testing.T) {
	ctx := context.Background()
	found, expr := SolveCifras(ctx, []int{10, 50}, 5, 1000)
	if !found {
		t.Fatalf("expected exact solution for 10, 50 -> 5, got found=%v", found)
	}
	if !strings.Contains(expr, "50 / 10 = 5") {
		t.Fatalf("expected 50 / 10 = 5, got: %s", expr)
	}
}

