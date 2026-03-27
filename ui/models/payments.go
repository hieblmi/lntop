package models

import (
	"context"
	"sort"
	"strconv"
	"sync"

	netmodels "github.com/hieblmi/lntop/network/models"
)

type PaymentsSort func(*netmodels.Payment, *netmodels.Payment) bool

type Payments struct {
	current *netmodels.Payment
	list    []*netmodels.Payment
	sort    PaymentsSort
	mu      sync.RWMutex
}

func (p *Payments) Current() *netmodels.Payment { return p.current }
func (p *Payments) SetCurrent(index int)        { p.current = p.Get(index) }
func (p *Payments) List() []*netmodels.Payment  { return p.list }
func (p *Payments) Len() int                    { return len(p.list) }
func (p *Payments) Swap(i, j int)               { p.list[i], p.list[j] = p.list[j], p.list[i] }
func (p *Payments) Less(i, j int) bool          { return p.sort(p.list[i], p.list[j]) }

func (p *Payments) Sort(s PaymentsSort) {
	if s == nil {
		return
	}
	p.sort = s
	sort.Sort(p)
}

func (p *Payments) Get(index int) *netmodels.Payment {
	if index < 0 || index > len(p.list)-1 {
		return nil
	}

	return p.list[index]
}

func (p *Payments) paymentKey(payment *netmodels.Payment) string {
	if payment == nil {
		return ""
	}
	if payment.PaymentIndex != 0 {
		return stringKeyUint64(payment.PaymentIndex)
	}
	return payment.PaymentHash
}

func (p *Payments) Contains(payment *netmodels.Payment) bool {
	key := p.paymentKey(payment)
	if key == "" {
		return false
	}
	for _, item := range p.list {
		if p.paymentKey(item) == key {
			return true
		}
	}
	return false
}

func (p *Payments) Add(payment *netmodels.Payment) {
	if payment == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.Contains(payment) {
		return
	}

	p.list = append(p.list, payment)
	if p.sort != nil {
		sort.Sort(p)
	}
}

func (p *Payments) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = nil
	p.list = nil
}

func (m *Models) RefreshPaymentsFromNetwork(ctx context.Context) error {
	payments, err := m.network.ListPayments(ctx)
	if err != nil {
		return err
	}

	m.Payments.Reset()
	for _, payment := range payments {
		m.Payments.Add(payment)
	}

	return nil
}

func stringKeyUint64(v uint64) string {
	return strconv.FormatUint(v, 10)
}
