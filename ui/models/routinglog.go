package models

import (
	"sort"

	netmodels "github.com/hieblmi/lntop/network/models"
)

type RoutingSort func(*netmodels.RoutingEvent, *netmodels.RoutingEvent) bool

func (r *RoutingLog) Len() int {
	return len(r.Log)
}

func (r *RoutingLog) Swap(i, j int) {
	r.Log[i], r.Log[j] = r.Log[j], r.Log[i]
}

func (r *RoutingLog) Less(i, j int) bool {
	return r.sort(r.Log[i], r.Log[j])
}

func (r *RoutingLog) Sort(s RoutingSort) {
	if s == nil {
		return
	}
	r.sort = s
	sort.Sort(r)
}

func (r *RoutingLog) Upsert(event *netmodels.RoutingEvent) {
	if event == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	found := false
	for _, existing := range r.Log {
		if existing.Equals(event) {
			existing.Update(event)
			found = true
			break
		}
	}
	if !found {
		if len(r.Log) == MaxRoutingEvents {
			r.Log = r.Log[1:]
		}
		r.Log = append(r.Log, event)
	}
	if r.sort != nil {
		sort.Sort(r)
	}
}
