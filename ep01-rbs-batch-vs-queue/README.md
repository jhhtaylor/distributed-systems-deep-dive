# EP01: The Night a Bank Forgot Everyone's Balance (RBS, 2012)

On 19 June 2012 a change to CA-7, the batch scheduler running RBS's overnight processing on an IBM mainframe, went wrong and the queue tracking which jobs had run was lost. Because everything depended on that one sequential schedule, payments for millions of RBS, NatWest and Ulster Bank customers stalled for days. The FCA and PRA fined RBS £56m.

## What the code shows

* `RunBatch` processes transactions strictly in order, like a nightly batch schedule. One corrupt transaction halts the run, and everything scheduled after it never happens.
* `RunQueue` processes the same transactions independently across a pool of workers. A message that keeps failing goes to a dead-letter queue, and the other 9,999 carry on.
* `Replicated` is PACELC's "else" branch: with no network partition at all, writing synchronously to a second node costs latency, and writing asynchronously leaves a short window where that node is stale.

```sh
go run ./ep01-rbs-batch-vs-queue/cmd/demo
```

```
== 2012: one sequential batch run ==
processed 4242, halted at txn-4242, never ran 5757

== Event-driven queue with a dead-letter queue ==
processed 9999, dead-lettered [txn-4242]
```

The dead-letter queue doesn't make the bad transaction go away. Someone still has to look at it, which is why the video pairs it with alerting on the dead-letter queue's size.

## Reconstruction

RBS's code isn't public. This models the failure shape described in the FCA's final notice and the reporting at the time, not CA-7 itself.

## Sources

* FCA, Final Notice to The Royal Bank of Scotland plc, National Westminster Bank plc and Ulster Bank Ltd, November 2014.
* PRA, Final Notice, November 2014.
