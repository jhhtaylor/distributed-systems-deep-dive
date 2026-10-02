package migration

import (
	"fmt"
	"testing"
)

// dnbVaults builds the 2026 shape: 86 tonnes overseas, 27 already at the
// London standard and 59 not.
func dnbVaults() (*Vault, *Vault) {
	var lots []Lot
	for i := 0; i < 27; i++ {
		lots = append(lots, Lot{ID: fmt.Sprintf("good-%02d", i), Tonnes: 1, Refiner: "approved-refiner-b"})
	}
	for i := 0; i < 59; i++ {
		lots = append(lots, Lot{ID: fmt.Sprintf("old-%02d", i), Tonnes: 1, Refiner: "unlisted-refiner"})
	}
	return NewVault("new-york-and-ottawa", lots...), NewVault("london")
}

func TestBothPathsMoveEverythingAndTheTotalNeverChanges(t *testing.T) {
	src, dst := dnbVaults()

	r := Migrate(src, dst, Conditions{})

	if r.LiftedAndShifted != 27 || r.ReDerived != 59 || r.StillAtSource != 0 {
		t.Fatalf("got %+v, want 27 lifted, 59 re-derived, 0 left", r)
	}
	if got := src.Tonnes() + dst.Tonnes(); got != 86 {
		t.Errorf("total reserve %dt, want 86t", got)
	}
}

func TestEverythingThatArrivesMeetsTheStandard(t *testing.T) {
	src, dst := dnbVaults()
	Migrate(src, dst, Conditions{})

	for _, l := range dst.Lots() {
		if !l.GoodDelivery() {
			t.Errorf("lot %s arrived in London without meeting the standard", l.ID)
		}
	}
}

func TestAMarketFreezeOnlyStallsThePathThatDependsOnIt(t *testing.T) {
	src, dst := dnbVaults()

	r := Migrate(src, dst, Conditions{MarketFrozen: true})

	if r.LiftedAndShifted != 27 {
		t.Errorf("lifted %dt, want the physical path's 27t to still move", r.LiftedAndShifted)
	}
	if r.ReDerived != 0 || r.StillAtSource != 59 {
		t.Errorf("got %+v, want the 59t that needs the market to wait at source", r)
	}
	if got := src.Tonnes() + dst.Tonnes(); got != 86 {
		t.Errorf("total reserve %dt, want 86t: nothing should be lost mid-migration", got)
	}
}

func TestWithBothPathsDownNothingMovesAndNothingIsLost(t *testing.T) {
	src, dst := dnbVaults()

	r := Migrate(src, dst, Conditions{MarketFrozen: true, RouteBlocked: true})

	if r.StillAtSource != 86 || dst.Tonnes() != 0 {
		t.Fatalf("got %+v with %dt in London, want everything still at source", r, dst.Tonnes())
	}
}
