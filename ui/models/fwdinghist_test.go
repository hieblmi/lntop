package models

import (
	"testing"
	"time"

	netmodels "github.com/hieblmi/lntop/network/models"
)

func TestFwdingHistStats(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	hist := &FwdingHist{}
	hist.Update([]*netmodels.ForwardingEvent{
		{
			ChanIdIn:     11,
			ChanIdOut:    22,
			PeerAliasIn:  "alice",
			PeerAliasOut: "bob",
			AmtOut:       1000,
			Fee:          10,
			FeeMsat:      10_000,
			EventTime:    now.Add(-2 * time.Hour),
		},
		{
			ChanIdIn:     11,
			ChanIdOut:    22,
			PeerAliasIn:  "alice",
			PeerAliasOut: "bob",
			AmtOut:       5000,
			Fee:          40,
			FeeMsat:      40_000,
			EventTime:    now,
		},
	})

	stats := hist.Stats()
	if stats.Count != 2 {
		t.Fatalf("Count = %d, want 2", stats.Count)
	}
	if stats.ForwardedTotal != 6000 {
		t.Fatalf("ForwardedTotal = %d, want 6000", stats.ForwardedTotal)
	}
	if stats.FeesTotalMsat != 50_000 {
		t.Fatalf("FeesTotalMsat = %d, want 50000", stats.FeesTotalMsat)
	}
	if stats.LargestForward != 5000 {
		t.Fatalf("LargestForward = %d, want 5000", stats.LargestForward)
	}
	if stats.SmallestForward != 1000 {
		t.Fatalf("SmallestForward = %d, want 1000", stats.SmallestForward)
	}
	if stats.MostProfitableFeeMsat != 40_000 {
		t.Fatalf("MostProfitableFeeMsat = %d, want 40000", stats.MostProfitableFeeMsat)
	}
	if stats.HottestLinkInChanID != 11 {
		t.Fatalf("HottestLinkInChanID = %d, want 11", stats.HottestLinkInChanID)
	}
	if stats.HottestLinkOutChanID != 22 {
		t.Fatalf("HottestLinkOutChanID = %d, want 22", stats.HottestLinkOutChanID)
	}
	if stats.HottestLinkInAlias != "alice" {
		t.Fatalf("HottestLinkInAlias = %q, want %q", stats.HottestLinkInAlias, "alice")
	}
	if stats.HottestLinkOutAlias != "bob" {
		t.Fatalf("HottestLinkOutAlias = %q, want %q", stats.HottestLinkOutAlias, "bob")
	}
	if !stats.FirstEventTime.Equal(now.Add(-2 * time.Hour)) {
		t.Fatalf("FirstEventTime = %v, want %v", stats.FirstEventTime, now.Add(-2*time.Hour))
	}
	if !stats.LastEventTime.Equal(now) {
		t.Fatalf("LastEventTime = %v, want %v", stats.LastEventTime, now)
	}
}
