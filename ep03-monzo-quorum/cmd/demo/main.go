package main

import (
	"fmt"

	quorum "github.com/jhhtaylor/distributed-systems-deep-dive/ep03-monzo-quorum"
)

const accounts = 2000

func run(bootstrap bool) {
	nodes := make([]string, 21)
	for i := range nodes {
		nodes[i] = fmt.Sprintf("node-%02d", i)
	}
	r := quorum.NewRing(3, nodes...)
	for i := 0; i < accounts; i++ {
		r.Write(fmt.Sprintf("account-%04d", i), "balance", 1)
	}

	for i := 0; i < 6; i++ {
		r.Join(fmt.Sprintf("new-%02d", i), bootstrap)
	}

	for i := 0; i < accounts; i++ {
		r.Read(fmt.Sprintf("account-%04d", i), 2)
	}

	status := "ok"
	if r.NotFoundRate() > 0.01 {
		status = "ALERT: row-not-found rate above 1%"
	}
	fmt.Printf("auto_bootstrap: %-5v  %4d of %d reads came back 'row not found' (%.1f%%)  %s\n",
		bootstrap, r.NotFound, r.Reads, 100*r.NotFoundRate(), status)
}

func main() {
	fmt.Println("21 nodes, replication factor 3, quorum reads of 2; then 6 nodes join at once.")
	run(false)
	run(true)
}
