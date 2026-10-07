package main

// retryBudget permits three retries with delays of 2, 4, and 8 seconds.
// Its zero value starts a new budget; it does not manage timers itself.
type retryBudget struct{ attempts int }

// next consumes a retry and returns its delay in seconds.
// After the 2, 4, and 8 second retries, it returns zero and false.
func (r *retryBudget) next() (int, bool) {
	if r.attempts >= 3 {
		return 0, false
	}
	seconds := 1 << (r.attempts + 1)
	r.attempts++
	return seconds, true
}
