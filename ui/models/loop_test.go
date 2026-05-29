package models

import (
	"testing"

	netmodels "github.com/hieblmi/lntop/network/models"
)

func TestLoopSwapsReplaceAllPreservesCurrent(t *testing.T) {
	swaps := &LoopSwaps{}
	swaps.ReplaceAll([]*netmodels.LoopSwap{
		{Type: netmodels.LoopSwapTypeOut, ID: "old"},
		{Type: netmodels.LoopSwapTypeStaticIn, ID: "selected", State: "old"},
	})
	swaps.SetCurrent(1)

	swaps.ReplaceAll([]*netmodels.LoopSwap{
		{Type: netmodels.LoopSwapTypeStaticIn, ID: "selected", State: "new"},
	})

	current := swaps.Current()
	if current == nil {
		t.Fatalf("current swap was cleared")
	}
	if current.ID != "selected" || current.State != "new" {
		t.Fatalf("current swap = %#v, want refreshed selected swap", current)
	}
}

func TestLoopSwapsUpsertRefreshesCurrent(t *testing.T) {
	swaps := &LoopSwaps{}
	swaps.Upsert(&netmodels.LoopSwap{
		Type:  netmodels.LoopSwapTypeOut,
		ID:    "selected",
		State: "old",
	})
	swaps.SetCurrent(0)

	swaps.Upsert(&netmodels.LoopSwap{
		Type:  netmodels.LoopSwapTypeOut,
		ID:    "selected",
		State: "new",
	})

	current := swaps.Current()
	if current == nil {
		t.Fatalf("current swap was cleared")
	}
	if current.State != "new" {
		t.Fatalf("current state = %q, want new", current.State)
	}
}
