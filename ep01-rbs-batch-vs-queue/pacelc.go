package rbs

import (
	"sync"
	"time"
)

// Replicated models PACELC's "else" branch from the RBS episode: with no
// network partition at all, a system still has to choose between latency and
// consistency every time it writes.
//
// Synchronous: Deposit waits for the secondary to confirm, so a read from
// either node straight afterwards is correct, but the caller pays the
// replication delay. Asynchronous: Deposit returns immediately and the
// secondary catches up in the background, so for a short window a read from
// the secondary is stale.
type Replicated struct {
	Synchronous bool
	ReplDelay   time.Duration

	mu        sync.Mutex
	primary   int
	secondary int
	pending   sync.WaitGroup
}

// Deposit adds amount (in pence) and reports how long the caller waited.
func (r *Replicated) Deposit(amount int) time.Duration {
	start := time.Now()

	r.mu.Lock()
	r.primary += amount
	r.mu.Unlock()

	replicate := func() {
		time.Sleep(r.ReplDelay)
		r.mu.Lock()
		r.secondary += amount
		r.mu.Unlock()
	}

	if r.Synchronous {
		replicate()
	} else {
		r.pending.Add(1)
		go func() {
			defer r.pending.Done()
			replicate()
		}()
	}
	return time.Since(start)
}

func (r *Replicated) ReadPrimary() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.primary
}

func (r *Replicated) ReadSecondary() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.secondary
}

// WaitForReplication blocks until background replication has caught up.
func (r *Replicated) WaitForReplication() { r.pending.Wait() }
