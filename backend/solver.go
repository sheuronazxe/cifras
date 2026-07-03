package main

import (
	"context"
	"fmt"
	"math"
	"strings"
)

type OpResult struct {
	value int
	steps []string
}

type solverState struct {
	target   int
	maxSteps int64
	counter  int64
	bestDiff int
	bestSteps []string
}

func SolveCifras(ctx context.Context, numbers []int, target int, maxSteps int) (bool, string) {
	var state []OpResult
	for _, n := range numbers {
		state = append(state, OpResult{value: n})
	}

	s := &solverState{
		target:   target,
		maxSteps: int64(maxSteps),
		bestDiff: math.MaxInt,
	}

	foundExact := s.dfs(ctx, state)

	finalExpr := strings.Join(s.bestSteps, "\n")
	return foundExact, finalExpr
}

func (s *solverState) dfs(ctx context.Context, current []OpResult) bool {
	select {
	case <-ctx.Done():
		return false
	default:
	}

	s.counter++
	if s.counter > s.maxSteps {
		return false
	}

	for _, op := range current {
		diff := int(math.Abs(float64(op.value - s.target)))
		if diff < s.bestDiff {
			s.bestDiff = diff
			if len(op.steps) > 0 {
				s.bestSteps = op.steps
			} else {
				s.bestSteps = []string{fmt.Sprintf("%d", op.value)}
			}
		}
		if s.bestDiff == 0 {
			return true
		}
	}

	n := len(current)
	if n == 1 {
		return false
	}

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			a := current[i]
			b := current[j]

			next := make([]OpResult, 0, n-1)
			for k := 0; k < n; k++ {
				if k != i && k != j {
					next = append(next, current[k])
				}
			}

			// Addition: a + b
			{
				val := a.value + b.value
				step := fmt.Sprintf("%d + %d = %d", a.value, b.value, val)
				newSteps := mergeSteps(a.steps, b.steps, step)
				if s.dfs(ctx, append(next, OpResult{val, newSteps})) {
					return true
				}
			}

			// Multiplication: a * b
			if a.value > 1 && b.value > 1 {
				val := a.value * b.value
				step := fmt.Sprintf("%d * %d = %d", a.value, b.value, val)
				newSteps := mergeSteps(a.steps, b.steps, step)
				if s.dfs(ctx, append(next, OpResult{val, newSteps})) {
					return true
				}
			}

			// Subtraction: a - b or b - a
			if a.value > b.value {
				val := a.value - b.value
				step := fmt.Sprintf("%d - %d = %d", a.value, b.value, val)
				newSteps := mergeSteps(a.steps, b.steps, step)
				if s.dfs(ctx, append(next, OpResult{val, newSteps})) {
					return true
				}
			} else if b.value > a.value {
				val := b.value - a.value
				step := fmt.Sprintf("%d - %d = %d", b.value, a.value, val)
				newSteps := mergeSteps(b.steps, a.steps, step)
				if s.dfs(ctx, append(next, OpResult{val, newSteps})) {
					return true
				}
			}

			// Division: a / b or b / a
			if b.value > 1 && a.value%b.value == 0 {
				val := a.value / b.value
				step := fmt.Sprintf("%d / %d = %d", a.value, b.value, val)
				newSteps := mergeSteps(a.steps, b.steps, step)
				if s.dfs(ctx, append(next, OpResult{val, newSteps})) {
					return true
				}
			} else if a.value > 1 && b.value%a.value == 0 {
				val := b.value / a.value
				step := fmt.Sprintf("%d / %d = %d", b.value, a.value, val)
				newSteps := mergeSteps(b.steps, a.steps, step)
				if s.dfs(ctx, append(next, OpResult{val, newSteps})) {
					return true
				}
			}
		}
	}
	return false
}

func mergeSteps(a, b []string, step string) []string {
	result := make([]string, 0, len(a)+len(b)+1)
	result = append(result, a...)
	result = append(result, b...)
	result = append(result, step)
	return result
}
