package rbs

import (
	"fmt"
	"testing"
	"time"
)

func txns(n, poisonAt int) []Txn {
	out := make([]Txn, n)
	for i := range out {
		out[i] = Txn{ID: fmt.Sprintf("txn-%04d", i), Amount: 5000, Corrupt: i == poisonAt}
	}
	return out
}

func TestBatchHaltsEverythingAfterThePoisonJob(t *testing.T) {
	r := RunBatch(txns(10, 3))

	if r.HaltedAt != "txn-0003" {
		t.Fatalf("halted at %q, want txn-0003", r.HaltedAt)
	}
	if len(r.Processed) != 3 {
		t.Errorf("processed %d, want 3", len(r.Processed))
	}
	if len(r.Skipped) != 6 {
		t.Errorf("skipped %d, want 6", len(r.Skipped))
	}
}

func TestBatchCompletesWithNoPoisonJob(t *testing.T) {
	r := RunBatch(txns(10, -1))
	if r.HaltedAt != "" || len(r.Processed) != 10 {
		t.Fatalf("got %+v, want all 10 processed", r)
	}
}

func TestQueueDeadLettersThePoisonMessageAndCarriesOn(t *testing.T) {
	r := RunQueue(txns(10_000, 4_242), 8, 3)

	if len(r.DeadLettered) != 1 || r.DeadLettered[0] != "txn-4242" {
		t.Fatalf("dead-lettered %v, want [txn-4242]", r.DeadLettered)
	}
	if len(r.Processed) != 9_999 {
		t.Errorf("processed %d, want 9999", len(r.Processed))
	}
}

func TestSynchronousReplicationIsConsistentButSlower(t *testing.T) {
	r := &Replicated{Synchronous: true, ReplDelay: 20 * time.Millisecond}

	waited := r.Deposit(5000)

	if waited < r.ReplDelay {
		t.Errorf("waited %v, want at least the replication delay %v", waited, r.ReplDelay)
	}
	if got := r.ReadSecondary(); got != 5000 {
		t.Errorf("secondary read %d straight after a synchronous deposit, want 5000", got)
	}
}

func TestAsynchronousReplicationIsFastButBrieflyStale(t *testing.T) {
	r := &Replicated{Synchronous: false, ReplDelay: 50 * time.Millisecond}

	r.Deposit(5000)

	if got := r.ReadPrimary(); got != 5000 {
		t.Errorf("primary read %d, want 5000", got)
	}
	if got := r.ReadSecondary(); got != 0 {
		t.Errorf("secondary read %d straight after an async deposit, want the stale 0", got)
	}

	r.WaitForReplication()
	if got := r.ReadSecondary(); got != 5000 {
		t.Errorf("secondary read %d after replication caught up, want 5000", got)
	}
}
