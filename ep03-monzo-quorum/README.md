# EP03: Phantom Nodes, Monzo's Cassandra Outage (2019)

On 29 July 2019 Monzo added six nodes to the 21-node Cassandra cluster holding most of its data, with `auto_bootstrap: false`. The new nodes took ownership of their share of the token ring straight away but hadn't been given any data yet. For about two hours, customers saw failed payments and wrong balances.

## What the code shows

Every row lives on three replicas, and a read needs two of them to answer. Quorum counts replicas that answer, not replicas that agree.

* **One empty replica (survivable):** the empty node answers first because it has nothing to read from disk, but the second answer has the data. The newest timestamp wins and the empty replica is repaired.
* **Two empty replicas (the outage):** both empty nodes answer first, the quorum of two is satisfied with no data in it, and the read returns "row not found". The one replica that does hold the row is never asked.
* **Bootstrapping first:** the same six nodes joining with their data streamed in beforehand lose nothing.
* **Alerting:** `NotFoundRate` is the signal Monzo added monitoring on afterwards.

```sh
go run ./ep03-monzo-quorum/cmd/demo
```

```
21 nodes, replication factor 3, quorum reads of 2; then 6 nodes join at once.
auto_bootstrap: false   374 of 2000 reads came back 'row not found' (18.7%)  ALERT: row-not-found rate above 1%
auto_bootstrap: true      0 of 2000 reads came back 'row not found' (0.0%)  ok
```

The exact count depends on where the new nodes' tokens happen to land. The model uses one token per node and fixed latencies (2ms empty, 14ms with data) to keep the mechanism visible.

## Reconstruction

Monzo's postmortem describes the configuration and the effect; the read path here is Cassandra's documented quorum and read-repair behaviour, not Monzo's code.

## Sources

* Monzo's public incident report on the 29 July 2019 Cassandra outage, published on monzo.com.
* Apache Cassandra documentation: consistency levels, read repair, `auto_bootstrap`.
