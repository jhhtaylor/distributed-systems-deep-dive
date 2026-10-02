package main

import (
	"fmt"
	"time"

	rbs "github.com/jhhtaylor/distributed-systems-deep-dive/ep01-rbs-batch-vs-queue"
)

func main() {
	const n, poison = 10_000, 4_242
	txns := make([]rbs.Txn, n)
	for i := range txns {
		txns[i] = rbs.Txn{ID: fmt.Sprintf("txn-%04d", i), Amount: 5000, Corrupt: i == poison}
	}

	fmt.Println("== 2012: one sequential batch run ==")
	b := rbs.RunBatch(txns)
	fmt.Printf("processed %d, halted at %s, never ran %d\n\n", len(b.Processed), b.HaltedAt, len(b.Skipped))

	fmt.Println("== Event-driven queue with a dead-letter queue ==")
	q := rbs.RunQueue(txns, 8, 3)
	fmt.Printf("processed %d, dead-lettered %v\n\n", len(q.Processed), q.DeadLettered)

	fmt.Println("== PACELC 'else': latency vs consistency, no partition needed ==")
	for _, sync := range []bool{true, false} {
		r := &rbs.Replicated{Synchronous: sync, ReplDelay: 30 * time.Millisecond}
		waited := r.Deposit(5000)
		fmt.Printf("synchronous=%-5v  caller waited %-6v  secondary reads £%.2f straight away\n",
			sync, waited.Round(time.Millisecond), float64(r.ReadSecondary())/100)
		r.WaitForReplication()
	}
}
