package main

import (
	"fmt"

	migration "github.com/jhhtaylor/outage-autopsies/ep02-dnb-gold-migration"
)

func vaults() (*migration.Vault, *migration.Vault) {
	var lots []migration.Lot
	for i := 0; i < 27; i++ {
		lots = append(lots, migration.Lot{ID: fmt.Sprintf("good-%02d", i), Tonnes: 1, Refiner: "approved-refiner-b"})
	}
	for i := 0; i < 59; i++ {
		lots = append(lots, migration.Lot{ID: fmt.Sprintf("old-%02d", i), Tonnes: 1, Refiner: "unlisted-refiner"})
	}
	return migration.NewVault("new-york-and-ottawa", lots...), migration.NewVault("london")
}

func main() {
	scenarios := []struct {
		name string
		c    migration.Conditions
	}{
		{"normal conditions", migration.Conditions{}},
		{"market freeze", migration.Conditions{MarketFrozen: true}},
		{"shipping route blocked", migration.Conditions{RouteBlocked: true}},
	}

	for _, s := range scenarios {
		src, dst := vaults()
		r := migration.Migrate(src, dst, s.c)
		fmt.Printf("%-24s lifted-and-shifted %2dt, re-derived %2dt, waiting at source %2dt, total still %dt\n",
			s.name+":", r.LiftedAndShifted, r.ReDerived, r.StillAtSource, src.Tonnes()+dst.Tonnes())
	}
}
