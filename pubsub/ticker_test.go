package pubsub

import (
	"testing"

	"github.com/hieblmi/lntop/network/models"
)

func TestChannelsChangedIgnoresStableState(t *testing.T) {
	base := []*models.Channel{
		{
			ChannelPoint:        "chan-1",
			LocalBalance:        10,
			RemoteBalance:       20,
			UnsettledBalance:    0,
			TotalAmountSent:     30,
			TotalAmountReceived: 40,
		},
	}

	if channelsChanged(base, cloneChannels(base)) {
		t.Fatalf("expected stable channel state to remain unchanged")
	}
}

func TestChannelsChangedDetectsTrafficCounters(t *testing.T) {
	old := []*models.Channel{
		{
			ChannelPoint:        "chan-1",
			TotalAmountSent:     30,
			TotalAmountReceived: 40,
		},
	}
	current := []*models.Channel{
		{
			ChannelPoint:        "chan-1",
			TotalAmountSent:     31,
			TotalAmountReceived: 42,
		},
	}

	if !channelsChanged(old, current) {
		t.Fatalf("expected sent/received counter changes to be detected")
	}
}

func TestChannelsChangedDetectsPendingHTLCUpdates(t *testing.T) {
	old := []*models.Channel{
		{
			ChannelPoint: "chan-1",
			PendingHTLC: []*models.HTLC{
				{
					Incoming:         true,
					Amount:           100,
					Hashlock:         []byte{0x01},
					ExpirationHeight: 1000,
				},
			},
		},
	}
	current := []*models.Channel{
		{
			ChannelPoint: "chan-1",
			PendingHTLC: []*models.HTLC{
				{
					Incoming:         true,
					Amount:           100,
					Hashlock:         []byte{0x02},
					ExpirationHeight: 1000,
				},
			},
		},
	}

	if !channelsChanged(old, current) {
		t.Fatalf("expected pending HTLC changes to be detected")
	}
}
