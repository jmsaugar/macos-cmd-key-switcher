package main

// Each new device event or wake gets three retries after the initial attempt.
// Delays are finite; successful operation leaves no timer running.
type retryBudget struct{ attempts int }

// next consumes one retry from the bounded exponential backoff budget.
//
// The receiver r is the budget to advance.
// It returns the delay in seconds and true, or zero and false after three retries.
func (r *retryBudget) next() (int, bool) {
	if r.attempts >= 3 {
		return 0, false
	}
	seconds := 1 << (r.attempts + 1)
	r.attempts++
	return seconds, true
}
