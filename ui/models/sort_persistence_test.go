package models

import (
	"testing"
	"time"

	netmodels "github.com/hieblmi/lntop/network/models"
)

func TestFwdingHistUpdateMaintainsSort(t *testing.T) {
	hist := &FwdingHist{}
	hist.Sort(func(a, b *netmodels.ForwardingEvent) bool {
		return a.EventTime.After(b.EventTime)
	})

	older := &netmodels.ForwardingEvent{EventTime: time.Unix(10, 0)}
	newer := &netmodels.ForwardingEvent{EventTime: time.Unix(20, 0)}

	hist.Update([]*netmodels.ForwardingEvent{older, newer})

	if got := hist.List()[0]; got != newer {
		t.Fatalf("first forwarding event = %v, want newest event first", got.EventTime)
	}
}

func TestChannelsUpdateMaintainsSortOnInsertAndMutation(t *testing.T) {
	chans := NewChannels()
	chans.Sort(func(a, b *netmodels.Channel) bool {
		return a.LocalBalance > b.LocalBalance
	})

	first := &netmodels.Channel{ChannelPoint: "a", LocalBalance: 100}
	second := &netmodels.Channel{ChannelPoint: "b", LocalBalance: 50}
	chans.Update(first)
	chans.Update(second)

	if got := chans.List()[0]; got.ChannelPoint != "a" {
		t.Fatalf("first channel after insert = %q, want %q", got.ChannelPoint, "a")
	}

	chans.Update(&netmodels.Channel{ChannelPoint: "b", LocalBalance: 150})

	if got := chans.List()[0]; got.ChannelPoint != "b" {
		t.Fatalf("first channel after mutation = %q, want %q", got.ChannelPoint, "b")
	}
}
