package main

import "testing"

// TestRetryBudgetStops verifies that retry delays stop after three attempts and a fresh budget
// resets the sequence.
//
// The parameter t runs the test and reports assertion failures.
//
// Failures are reported through t.
func TestRetryBudgetStops(t *testing.T) {
	var budget retryBudget
	for _, want := range []int{2, 4, 8} {
		if got, ok := budget.next(); !ok || got != want {
			t.Fatalf("got %d, %t; want %d", got, ok, want)
		}
	}
	for i := 0; i < 3; i++ {
		if _, ok := budget.next(); ok {
			t.Fatal("retry budget did not stop")
		}
	}
	budget = retryBudget{}
	if got, ok := budget.next(); !ok || got != 2 {
		t.Fatal("new event did not reset retry budget")
	}
}
