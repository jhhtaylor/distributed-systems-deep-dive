package quorum

import (
	"errors"
	"fmt"
	"testing"
)

func names(prefix string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("%s-%02d", prefix, i)
	}
	return out
}

// Three nodes and a replication factor of 3 means every node holds every
// row, so emptying some of them is exactly "some replicas are new".
func threeReplicaRing(empty int) *Ring {
	r := NewRing(3, "a", "b", "c")
	r.Write("account-8f21", "£482.10", 1)
	for _, n := range r.nodes[:empty] {
		delete(n.data, "account-8f21")
	}
	return r
}

func TestCaseA_OneEmptyReplicaIsSurvivable(t *testing.T) {
	r := threeReplicaRing(1)

	got, err := r.Read("account-8f21", 2)

	if err != nil || got != "£482.10" {
		t.Fatalf("got %q, %v; want £482.10", got, err)
	}
	for _, n := range r.nodes {
		if !n.Has("account-8f21") {
			t.Errorf("node %s still empty after read repair", n.Name)
		}
	}
}

func TestCaseB_TwoEmptyReplicasSatisfyQuorumWithNoData(t *testing.T) {
	r := threeReplicaRing(2)

	_, err := r.Read("account-8f21", 2)

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
	if !r.nodes[2].Has("account-8f21") {
		t.Error("the one replica holding the row should still hold it; nothing asked it")
	}
}

func TestNewestTimestampWinsWhenRepliesDisagree(t *testing.T) {
	r := NewRing(3, "a", "b", "c")
	r.Write("k", "old", 1)
	r.nodes[0].data["k"] = Cell{Value: "new", WrittenAt: 2}

	got, err := r.Read("k", 3)

	if err != nil || got != "new" {
		t.Fatalf("got %q, %v; want the newer value", got, err)
	}
}

func clusterWithAccounts(t *testing.T, accounts int) *Ring {
	t.Helper()
	r := NewRing(3, names("node", 21)...)
	for i := 0; i < accounts; i++ {
		r.Write(fmt.Sprintf("account-%04d", i), "balance", 1)
	}
	return r
}

func readAll(r *Ring, accounts int) {
	for i := 0; i < accounts; i++ {
		r.Read(fmt.Sprintf("account-%04d", i), 2)
	}
}

func TestSixNodesJoiningWithoutBootstrapLoseSomeReads(t *testing.T) {
	const accounts = 2000
	r := clusterWithAccounts(t, accounts)
	for _, n := range names("new", 6) {
		r.Join(n, false)
	}

	readAll(r, accounts)

	if r.NotFound == 0 {
		t.Fatal("expected some accounts to read as not found with two empty replicas")
	}
	if r.NotFound == accounts {
		t.Fatal("expected most accounts to survive; only ranges with two new replicas should fail")
	}
}

func TestBootstrappingBeforeJoiningLosesNothing(t *testing.T) {
	const accounts = 2000
	r := clusterWithAccounts(t, accounts)
	for _, n := range names("new", 6) {
		r.Join(n, true)
	}

	readAll(r, accounts)

	if r.NotFound != 0 {
		t.Fatalf("%d accounts read as not found after a proper bootstrap, want 0", r.NotFound)
	}
}
