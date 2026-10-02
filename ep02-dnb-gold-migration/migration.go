// Package migration models DNB's 2026 gold relocation as the live data
// migration it resembles: two independent migration paths running at once,
// and an invariant that the total reserve never changes no matter which path
// fails.
//
//   - Lift-and-shift: lots that already meet the destination's standard are
//     moved as-is (DNB flew these via its own vault at Zeist to audit them).
//   - Re-derive at destination: lots that don't meet the standard are sold
//     at the source and an equivalent compliant lot is bought at the
//     destination, so nothing has to be converted in flight.
package migration

import (
	"fmt"
	"sort"
	"sync"
)

// Lot is a parcel of gold. Real Good Delivery bars are about 12.5kg; a whole
// tonne per lot keeps the numbers in the demo readable.
type Lot struct {
	ID      string
	Tonnes  int
	Refiner string
}

// ApprovedRefiners stands in for the London Good Delivery list: a bar can be
// pure gold and still be rejected if it wasn't cast by an approved refiner.
var ApprovedRefiners = map[string]bool{
	"approved-refiner-a": true,
	"approved-refiner-b": true,
}

func (l Lot) GoodDelivery() bool { return ApprovedRefiners[l.Refiner] }

type Vault struct {
	Name string
	mu   sync.Mutex
	lots map[string]Lot
}

func NewVault(name string, lots ...Lot) *Vault {
	v := &Vault{Name: name, lots: map[string]Lot{}}
	for _, l := range lots {
		v.lots[l.ID] = l
	}
	return v
}

func (v *Vault) put(l Lot) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.lots[l.ID] = l
}

func (v *Vault) take(id string) Lot {
	v.mu.Lock()
	defer v.mu.Unlock()
	l := v.lots[id]
	delete(v.lots, id)
	return l
}

func (v *Vault) Lots() []Lot {
	v.mu.Lock()
	defer v.mu.Unlock()
	out := make([]Lot, 0, len(v.lots))
	for _, l := range v.lots {
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (v *Vault) Tonnes() int {
	total := 0
	for _, l := range v.Lots() {
		total += l.Tonnes
	}
	return total
}

// Conditions lets a test or demo knock out either migration path, the
// failure DNB said it was rehearsing for.
type Conditions struct {
	MarketFrozen bool // the sell-and-rebuy path can't trade
	RouteBlocked bool // the physical path can't ship
}

type Report struct {
	LiftedAndShifted int // tonnes moved physically
	ReDerived        int // tonnes sold at source and rebought at destination
	StillAtSource    int // tonnes waiting for a path to come back
}

// Migrate moves every lot it can from src to dst. Both paths run at the same
// time and a failure on one never touches the lots the other is handling.
// Anything that can't move stays in src: no lot is ever lost in between.
func Migrate(src, dst *Vault, c Conditions) Report {
	var lift, rederive []Lot
	for _, l := range src.Lots() {
		if l.GoodDelivery() {
			lift = append(lift, l)
		} else {
			rederive = append(rederive, l)
		}
	}

	var (
		mu sync.Mutex
		r  Report
		wg sync.WaitGroup
	)
	add := func(field *int, n int) {
		mu.Lock()
		*field += n
		mu.Unlock()
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		for _, l := range lift {
			if c.RouteBlocked {
				continue
			}
			dst.put(src.take(l.ID))
			add(&r.LiftedAndShifted, l.Tonnes)
		}
	}()
	go func() {
		defer wg.Done()
		for _, l := range rederive {
			if c.MarketFrozen {
				continue
			}
			sold := src.take(l.ID)
			dst.put(Lot{
				ID:      fmt.Sprintf("%s-rebought", sold.ID),
				Tonnes:  sold.Tonnes,
				Refiner: "approved-refiner-a",
			})
			add(&r.ReDerived, sold.Tonnes)
		}
	}()
	wg.Wait()

	r.StillAtSource = src.Tonnes()
	return r
}
