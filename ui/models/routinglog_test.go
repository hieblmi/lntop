package models

import (
	"testing"
	"time"

	netmodels "github.com/hieblmi/lntop/network/models"
)

func TestRoutingLogUpsertMaintainsSort(t *testing.T) {
	log := &RoutingLog{
		Log: []*netmodels.RoutingEvent{
			{IncomingChannelId: 1, IncomingHtlcId: 1, LastUpdate: time.Unix(10, 0)},
			{IncomingChannelId: 2, IncomingHtlcId: 1, LastUpdate: time.Unix(20, 0)},
		},
	}

	log.Sort(func(a, b *netmodels.RoutingEvent) bool {
		at, bt := a.LastUpdate, b.LastUpdate
		return DateSort(&at, &bt, Desc)
	})

	log.Upsert(&netmodels.RoutingEvent{
		IncomingChannelId: 3,
		IncomingHtlcId:    1,
		LastUpdate:        time.Unix(30, 0),
	})

	if got := log.Log[0].IncomingChannelId; got != 3 {
		t.Fatalf("first routing event channel = %d, want 3", got)
	}
}
