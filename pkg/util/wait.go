package util

import (
	"time"

	"github.com/pkg/errors"
)

type RetryOptions struct {
	// Immediate when true runs the condition first & then waits
	// if required
	Immediate bool
	Interval  time.Duration
	Timeout   time.Duration
}

// Retry will execute the condition & repeatedly in intervals
// till this condition returns true or times out
//
// Note: It is valid for a condition to return true with error
// Note: Original error throws by the condition is preserved
func Retry(opts RetryOptions, cond func() (bool, error)) error {
	var count int = 1
	start := time.Now()
	for {
		if !opts.Immediate {
			// first wait then run the condition
			time.Sleep(opts.Interval)
		}
		done, err := cond()
		if done {
			// this may or may not be nil
			return err
		}
		elaspsed := time.Since(start)
		if elaspsed > opts.Timeout {
			return errors.Errorf("timed out after %s with %d retries: %s", elaspsed, count, err)
		}
		count++
		if opts.Immediate {
			// first run the condition then wait
			time.Sleep(opts.Interval)
		}
	}
}
