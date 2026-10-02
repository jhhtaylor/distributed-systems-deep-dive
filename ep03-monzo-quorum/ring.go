// Package quorum reconstructs the mechanism behind Monzo's July 2019
// Cassandra incident: nodes that join a token ring without streaming their
// data first still take ownership of key ranges, and because an empty node
// has nothing to read from disk it answers fastest. With a quorum of 2 out of
// 3 replicas, one empty replica is survivable (the newest timestamp wins and
// the empty one is repaired); two empty replicas satisfy the quorum with no
// data in it, and the read comes back "row not found".
//
// Quorum here counts replicas that ANSWER, not replicas that agree.
package quorum

import (
	"errors"
	"hash/fnv"
	"sort"
)

var ErrNotFound = errors.New("row not found")

type Cell struct {
	Value     string
	WrittenAt int64
}

type Node struct {
	Name  string
	token uint32
	data  map[string]Cell
}

// latencyMs models the asymmetry that made the outage possible: a replica
// holding the row has to read it from disk, an empty one replies at once.
func (n *Node) latencyMs(key string) int {
	if _, ok := n.data[key]; ok {
		return 14
	}
	return 2
}

func (n *Node) Has(key string) bool { _, ok := n.data[key]; return ok }

type Ring struct {
	ReplicationFactor int
	nodes             []*Node // sorted by token

	Reads    int
	NotFound int
}

func NewRing(replicationFactor int, names ...string) *Ring {
	r := &Ring{ReplicationFactor: replicationFactor}
	for _, name := range names {
		r.insert(name)
	}
	return r
}

func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func (r *Ring) insert(name string) *Node {
	n := &Node{Name: name, token: hash(name), data: map[string]Cell{}}
	r.nodes = append(r.nodes, n)
	sort.Slice(r.nodes, func(i, j int) bool { return r.nodes[i].token < r.nodes[j].token })
	return n
}

// ReplicasFor walks the ring clockwise from the key's token and returns the
// first ReplicationFactor nodes it meets.
func (r *Ring) ReplicasFor(key string) []*Node {
	t := hash(key)
	start := sort.Search(len(r.nodes), func(i int) bool { return r.nodes[i].token >= t })
	n := r.ReplicationFactor
	if n > len(r.nodes) {
		n = len(r.nodes)
	}
	out := make([]*Node, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, r.nodes[(start+i)%len(r.nodes)])
	}
	return out
}

func (r *Ring) Write(key, value string, at int64) {
	for _, n := range r.ReplicasFor(key) {
		n.data[key] = Cell{Value: value, WrittenAt: at}
	}
}

// Join adds a node to the ring. With bootstrap, the node is given a copy of
// every row it is about to own before it starts answering reads. Without it
// (auto_bootstrap: false), it owns those rows immediately and holds none.
func (r *Ring) Join(name string, bootstrap bool) {
	latest := map[string]Cell{}
	if bootstrap {
		for _, n := range r.nodes {
			for k, c := range n.data {
				if cur, ok := latest[k]; !ok || c.WrittenAt > cur.WrittenAt {
					latest[k] = c
				}
			}
		}
	}

	joined := r.insert(name)

	for k, c := range latest {
		for _, n := range r.ReplicasFor(k) {
			if n == joined {
				joined.data[k] = c
			}
		}
	}
}

// Read asks the key's replicas, takes the first `quorum` to reply, and
// returns the newest value among them. Responders that were missing the row
// or held an older value are repaired.
func (r *Ring) Read(key string, quorum int) (string, error) {
	replicas := r.ReplicasFor(key)
	sort.SliceStable(replicas, func(i, j int) bool {
		return replicas[i].latencyMs(key) < replicas[j].latencyMs(key)
	})
	responders := replicas[:quorum]

	var newest Cell
	found := false
	for _, n := range responders {
		if c, ok := n.data[key]; ok && (!found || c.WrittenAt > newest.WrittenAt) {
			newest, found = c, true
		}
	}

	r.Reads++
	if !found {
		r.NotFound++
		return "", ErrNotFound
	}
	for _, n := range responders {
		n.data[key] = newest
	}
	return newest.Value, nil
}

// NotFoundRate is the signal Monzo added alerting on afterwards: a sudden
// rise in "row not found" means reads are succeeding against nodes that
// don't have the data.
func (r *Ring) NotFoundRate() float64 {
	if r.Reads == 0 {
		return 0
	}
	return float64(r.NotFound) / float64(r.Reads)
}
