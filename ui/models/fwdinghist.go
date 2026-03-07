package models

import (
	"sort"
	"sync"
	"time"

	"github.com/hieblmi/lntop/network/models"
)

type FwdinghistSort func(*models.ForwardingEvent, *models.ForwardingEvent) bool

type FwdingHist struct {
	StartTime    string
	MaxNumEvents uint32
	current      *models.ForwardingEvent
	list         []*models.ForwardingEvent
	sort         FwdinghistSort
	mu           sync.RWMutex
}

type FwdingHistStats struct {
	Count                 int
	ForwardedTotal        uint64
	FeesTotalMsat         uint64
	LargestForward        uint64
	SmallestForward       uint64
	MostProfitableFeeMsat uint64
	HottestLinkInChanID   uint64
	HottestLinkOutChanID  uint64
	HottestLinkInAlias    string
	HottestLinkOutAlias   string
	FirstEventTime        time.Time
	LastEventTime         time.Time
}

func (t *FwdingHist) Current() *models.ForwardingEvent {
	return t.current
}

func (t *FwdingHist) SetCurrent(index int) {
	t.current = t.Get(index)
}

func (t *FwdingHist) List() []*models.ForwardingEvent {
	return t.list
}

func (t *FwdingHist) Len() int {
	return len(t.list)
}

func (t *FwdingHist) Swap(i, j int) {
	t.list[i], t.list[j] = t.list[j], t.list[i]
}

func (t *FwdingHist) Less(i, j int) bool {
	return t.sort(t.list[i], t.list[j])
}

func (t *FwdingHist) Sort(s FwdinghistSort) {
	if s == nil {
		return
	}
	t.sort = s
	sort.Sort(t)
}

func (t *FwdingHist) Get(index int) *models.ForwardingEvent {
	if index < 0 || index > len(t.list)-1 {
		return nil
	}

	return t.list[index]
}

func (t *FwdingHist) Update(events []*models.ForwardingEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.list = events
	if t.sort != nil {
		sort.Sort(t)
	}
}

func (t *FwdingHist) Stats() FwdingHistStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := FwdingHistStats{Count: len(t.list)}
	if len(t.list) == 0 {
		return stats
	}

	stats.FirstEventTime = t.list[0].EventTime
	stats.LastEventTime = t.list[0].EventTime
	stats.SmallestForward = t.list[0].AmtOut
	type routeKey struct {
		in  uint64
		out uint64
	}
	linkForwarded := make(map[routeKey]uint64)
	hottestForwarded := uint64(0)

	for _, event := range t.list {
		stats.ForwardedTotal += event.AmtOut
		stats.FeesTotalMsat += event.FeeMsat
		if event.AmtOut > stats.LargestForward {
			stats.LargestForward = event.AmtOut
		}
		if event.AmtOut < stats.SmallestForward {
			stats.SmallestForward = event.AmtOut
		}
		if event.FeeMsat > stats.MostProfitableFeeMsat {
			stats.MostProfitableFeeMsat = event.FeeMsat
		}
		if event.EventTime.Before(stats.FirstEventTime) {
			stats.FirstEventTime = event.EventTime
		}
		if event.EventTime.After(stats.LastEventTime) {
			stats.LastEventTime = event.EventTime
		}
		key := routeKey{in: event.ChanIdIn, out: event.ChanIdOut}
		linkForwarded[key] += event.AmtOut
		if linkForwarded[key] > hottestForwarded {
			hottestForwarded = linkForwarded[key]
			stats.HottestLinkInChanID = event.ChanIdIn
			stats.HottestLinkOutChanID = event.ChanIdOut
			stats.HottestLinkInAlias = event.PeerAliasIn
			stats.HottestLinkOutAlias = event.PeerAliasOut
		}
	}

	return stats
}
