package views

import (
	"strings"
	"testing"

	"github.com/hieblmi/lntop/config"
	netmodels "github.com/hieblmi/lntop/network/models"
	uimodels "github.com/hieblmi/lntop/ui/models"
)

func TestLoopHeaderShowsFullStaticAddress(t *testing.T) {
	address := "bcrt1p3mjfq8x9w7v7ktv5c9r0nqg9t0g6phd3n2h4q6d8y0x9z7a6r5qsnw7t4"
	view := NewLoop(nil, nil, &uimodels.LoopInfoModel{
		LoopInfo: &netmodels.LoopInfo{
			Version:       "0.30.0",
			Network:       "regtest",
			StaticAddress: address,
		},
	}, nil, nil)

	header := view.renderHeader(120)

	if !strings.Contains(header, address) {
		t.Fatalf("header does not contain full static address:\n%s", header)
	}
}

func TestLoopStaticSwapCostColumnsRenderDash(t *testing.T) {
	table := newLoopSwapsTable(&config.View{Columns: []string{
		"COST_SRV", "COST_CHAIN", "COST_OFFCH",
	}}, &uimodels.LoopSwaps{})
	swap := &netmodels.LoopSwap{Type: netmodels.LoopSwapTypeStaticIn}

	for _, col := range table.columns {
		got := strings.TrimSpace(stripAnsi(col.display(swap)))
		if got != "-" {
			t.Fatalf("static swap cost column %q = %q, want dash", col.name, got)
		}
	}
}
