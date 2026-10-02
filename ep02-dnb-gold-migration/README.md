# EP02: Why a Central Bank Just Relocated $10 Billion in Gold (DNB, 2026)

Over five months in 2026, De Nederlandsche Bank moved 86 tonnes of gold from New York and Ottawa to London. 27 tonnes already met the London Good Delivery standard and were flown via DNB's own vault at Zeist. The other 59 tonnes didn't, so DNB sold them in New York and bought an equivalent amount of compliant gold in London. Running both methods at once meant a problem with one, like a market freeze or a blocked route, couldn't stall the whole move.

That's the same playbook as a zero-downtime migration of a production system.

## What the code shows

* **Lift-and-shift:** records already valid at the destination are moved as they are.
* **Re-derive at destination:** records that aren't valid are recreated in the destination's own format from the same source of truth, rather than converted in flight.
* **Parallel paths:** both run at the same time, and knocking out one leaves the other's work untouched.
* **The invariant:** the total across both vaults is 86 tonnes before, during and after, whichever path fails. Nothing is lost in between.

```sh
go run ./ep02-dnb-gold-migration/cmd/demo
```

```
normal conditions:       lifted-and-shifted 27t, re-derived 59t, waiting at source  0t, total still 86t
market freeze:           lifted-and-shifted 27t, re-derived  0t, waiting at source 59t, total still 86t
shipping route blocked:  lifted-and-shifted  0t, re-derived 59t, waiting at source 27t, total still 86t
```

## Reconstruction

DNB published what it did, not how. The lots here are a tonne each to keep the numbers readable; real Good Delivery bars are about 12.5kg.

## Sources

* De Nederlandsche Bank press release, "DNB improves tradability of gold reserves", September 2026.
* LBMA, Good Delivery Rules.
