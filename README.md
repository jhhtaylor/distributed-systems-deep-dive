# Outage Autopsies

Code for my YouTube videos, where I take apart real software outages and banking stories and look at what actually broke underneath. One folder per episode, each with a small runnable model of the mechanism and tests that pin down the behaviour shown in the video.

Channel: [youtube.com/@jhhtaylor](https://www.youtube.com/@jhhtaylor)

| Episode | Story | Folder | What the code shows |
|---|---|---|---|
| 01 | The night RBS forgot everyone's balance (2012) | [ep01-rbs-batch-vs-queue](ep01-rbs-batch-vs-queue) | A sequential batch run halting on one bad job, vs an event-driven queue with a dead-letter queue; plus PACELC's latency vs consistency trade-off |
| 02 | Why a central bank relocated $10bn of gold (DNB, 2026) | [ep02-dnb-gold-migration](ep02-dnb-gold-migration) | A live migration with two independent paths (lift-and-shift, re-derive at destination) and a total that never changes when one path fails |
| 03 | Phantom nodes: Monzo's Cassandra outage (2019) | [ep03-monzo-quorum](ep03-monzo-quorum) | A token ring where nodes join without their data, and why quorum reads then return "row not found" |

## Running it

Needs Go 1.21 or later.

```sh
go test ./...                              # every episode's tests
go run ./ep03-monzo-quorum/cmd/demo        # one episode's demo
```

## A note on accuracy

None of these companies' real code is public, so each model is a reconstruction of the mechanism described in their postmortems and public reporting, simplified to what's needed to see the failure. Each episode's README links to the sources.
