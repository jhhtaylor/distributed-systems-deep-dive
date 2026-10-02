// Package rbs models the two architectures compared in the RBS episode: a
// strictly sequential batch run, where one bad job stops everything queued
// behind it, and an event-driven queue, where a bad message is set aside in a
// dead-letter queue and everything else carries on.
package rbs

import (
	"errors"
	"sort"
	"sync"
)

// ErrCorruptState is what a transaction returns when its state can't be
// trusted, like the jobs CA-7 could no longer place after its tracking data
// was wiped.
var ErrCorruptState = errors.New("corrupt transaction state")

type Txn struct {
	ID      string
	Amount  int // pence
	Corrupt bool
}

func process(t Txn) error {
	if t.Corrupt {
		return ErrCorruptState
	}
	return nil
}

type BatchResult struct {
	Processed []string
	HaltedAt  string // empty if the whole batch completed
	Skipped   []string
}

// RunBatch processes every transaction in strict order. The first failure
// halts the run, so every transaction scheduled after it never happens.
func RunBatch(txns []Txn) BatchResult {
	var r BatchResult
	for i, t := range txns {
		if err := process(t); err != nil {
			r.HaltedAt = t.ID
			for _, rest := range txns[i+1:] {
				r.Skipped = append(r.Skipped, rest.ID)
			}
			return r
		}
		r.Processed = append(r.Processed, t.ID)
	}
	return r
}

type QueueResult struct {
	Processed    []string
	DeadLettered []string
}

// RunQueue processes transactions independently across a pool of workers.
// A message that still fails after maxAttempts goes to the dead-letter queue
// for a human to look at; it never blocks the others.
func RunQueue(txns []Txn, workers, maxAttempts int) QueueResult {
	queue := make(chan Txn)
	var (
		mu sync.Mutex
		r  QueueResult
		wg sync.WaitGroup
	)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range queue {
				var err error
				for attempt := 1; attempt <= maxAttempts; attempt++ {
					if err = process(t); err == nil {
						break
					}
				}
				mu.Lock()
				if err != nil {
					r.DeadLettered = append(r.DeadLettered, t.ID)
				} else {
					r.Processed = append(r.Processed, t.ID)
				}
				mu.Unlock()
			}
		}()
	}

	for _, t := range txns {
		queue <- t
	}
	close(queue)
	wg.Wait()

	sort.Strings(r.Processed)
	sort.Strings(r.DeadLettered)
	return r
}
