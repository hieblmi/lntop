package models

import (
	"context"
	"sort"
	"sync"

	netmodels "github.com/hieblmi/lntop/network/models"
)

// LoopSwapsSort is a comparator used by Sort. Identical pattern to
// PaymentsSort etc.
type LoopSwapsSort func(*netmodels.LoopSwap, *netmodels.LoopSwap) bool

// LoopSwaps holds the unified swap list backing the Swaps sub-tab.
type LoopSwaps struct {
	current *netmodels.LoopSwap
	list    []*netmodels.LoopSwap
	sort    LoopSwapsSort
	mu      sync.RWMutex
}

func (s *LoopSwaps) Current() *netmodels.LoopSwap { return s.current }
func (s *LoopSwaps) SetCurrent(index int)         { s.current = s.Get(index) }
func (s *LoopSwaps) List() []*netmodels.LoopSwap  { return s.list }
func (s *LoopSwaps) Len() int                     { return len(s.list) }
func (s *LoopSwaps) Swap(i, j int)                { s.list[i], s.list[j] = s.list[j], s.list[i] }
func (s *LoopSwaps) Less(i, j int) bool           { return s.sort(s.list[i], s.list[j]) }

func (s *LoopSwaps) Get(index int) *netmodels.LoopSwap {
	if index < 0 || index >= len(s.list) {
		return nil
	}
	return s.list[index]
}

func (s *LoopSwaps) Sort(c LoopSwapsSort) {
	if c == nil {
		return
	}
	s.sort = c
	sort.Sort(s)
}

// indexOf returns the position of a swap matching the same Type+ID, or -1.
func (s *LoopSwaps) indexOf(swap *netmodels.LoopSwap) int {
	if swap == nil {
		return -1
	}
	for i, existing := range s.list {
		if existing.Type == swap.Type && existing.ID == swap.ID {
			return i
		}
	}
	return -1
}

// Upsert inserts or updates a swap row. Used by both the bulk reload path
// (ApplyLoopSwaps) and the streaming Monitor handler.
func (s *LoopSwaps) Upsert(swap *netmodels.LoopSwap) {
	if swap == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if i := s.indexOf(swap); i >= 0 {
		s.list[i] = swap
	} else {
		s.list = append(s.list, swap)
	}
	if sameLoopSwap(s.current, swap) {
		s.current = swap
	}
	if s.sort != nil {
		sort.Sort(s)
	}
}

func (s *LoopSwaps) ReplaceAll(swaps []*netmodels.LoopSwap) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current := s.current
	s.list = make([]*netmodels.LoopSwap, 0, len(swaps))
	for _, swap := range swaps {
		if swap == nil {
			continue
		}
		s.list = append(s.list, swap)
	}
	if s.sort != nil {
		sort.Sort(s)
	}

	s.current = nil
	for _, swap := range s.list {
		if sameLoopSwap(current, swap) {
			s.current = swap
			break
		}
	}
}

func (s *LoopSwaps) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = nil
	s.list = nil
}

func sameLoopSwap(a, b *netmodels.LoopSwap) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Type == b.Type && a.ID == b.ID
}

// LoopDepositsSort comparator for the Deposits sub-tab.
type LoopDepositsSort func(*netmodels.LoopDeposit, *netmodels.LoopDeposit) bool

type LoopDeposits struct {
	current *netmodels.LoopDeposit
	list    []*netmodels.LoopDeposit
	sort    LoopDepositsSort
	mu      sync.RWMutex
}

func (d *LoopDeposits) Current() *netmodels.LoopDeposit { return d.current }
func (d *LoopDeposits) SetCurrent(index int)            { d.current = d.Get(index) }
func (d *LoopDeposits) List() []*netmodels.LoopDeposit  { return d.list }
func (d *LoopDeposits) Len() int                        { return len(d.list) }
func (d *LoopDeposits) Swap(i, j int) {
	d.list[i], d.list[j] = d.list[j], d.list[i]
}
func (d *LoopDeposits) Less(i, j int) bool { return d.sort(d.list[i], d.list[j]) }

func (d *LoopDeposits) Get(index int) *netmodels.LoopDeposit {
	if index < 0 || index >= len(d.list) {
		return nil
	}
	return d.list[index]
}

func (d *LoopDeposits) Sort(c LoopDepositsSort) {
	if c == nil {
		return
	}
	d.sort = c
	sort.Sort(d)
}

func (d *LoopDeposits) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.current = nil
	d.list = nil
}

func (d *LoopDeposits) ReplaceAll(deposits []*netmodels.LoopDeposit) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.list = deposits
	if d.sort != nil {
		sort.Sort(d)
	}
}

// LoopInfoModel wraps netmodels.LoopInfo so the rest of the UI layer can
// follow the same pointer-to-wrapper pattern used by Info, WalletBalance,
// etc.
type LoopInfoModel struct {
	*netmodels.LoopInfo
}

// Refresh* helpers are invoked from the model loaders/handleEvent paths.

func (m *Models) ApplyLoopInfo(info *netmodels.LoopInfo) {
	*m.LoopInfo = LoopInfoModel{info}
}

func (m *Models) ApplyLoopSwaps(swaps []*netmodels.LoopSwap) {
	m.LoopSwaps.ReplaceAll(swaps)
}

func (m *Models) ApplyLoopDeposits(deposits []*netmodels.LoopDeposit) {
	m.LoopDeposits.ReplaceAll(deposits)
}

func (m *Models) RefreshLoopSwap(update interface{}) func(context.Context) error {
	return func(ctx context.Context) error {
		swap, ok := update.(*netmodels.LoopSwap)
		if !ok || swap == nil {
			return nil
		}
		m.LoopSwaps.Upsert(swap)
		return nil
	}
}
