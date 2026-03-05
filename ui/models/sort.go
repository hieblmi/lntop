package models

import (
	"cmp"
	"strings"
	"time"
)

type Order int

const (
	Asc Order = iota
	Desc
)

// OrderedSort compares two ordered values according to the given sort order.
func OrderedSort[T cmp.Ordered](a, b T, o Order) bool {
	if o == Asc {
		return a < b
	}
	return a > b
}

// IntSort compares two int values according to the given sort order.
func IntSort(a, b int, o Order) bool { return OrderedSort(a, b, o) }

// Int32Sort compares two int32 values according to the given sort order.
func Int32Sort(a, b int32, o Order) bool { return OrderedSort(a, b, o) }

// Int64Sort compares two int64 values according to the given sort order.
func Int64Sort(a, b int64, o Order) bool { return OrderedSort(a, b, o) }

// Float64Sort compares two float64 values according to the given sort order.
func Float64Sort(a, b float64, o Order) bool { return OrderedSort(a, b, o) }

// UInt32Sort compares two uint32 values according to the given sort order.
func UInt32Sort(a, b uint32, o Order) bool { return OrderedSort(a, b, o) }

// UInt64Sort compares two uint64 values according to the given sort order.
func UInt64Sort(a, b uint64, o Order) bool { return OrderedSort(a, b, o) }

func DateSort(a, b *time.Time, o Order) bool {
	if o == Desc {
		if a == nil || b == nil {
			return b == nil
		}

		return a.After(*b)
	}

	if a == nil || b == nil {
		return a == nil
	}

	return a.Before(*b)
}

func StringSort(a, b string, o Order) bool {
	result := strings.Compare(a, b)
	if o == Asc {
		return result < 0
	}
	return result > 0
}

func BoolSort(a, b bool, o Order) bool {
	if o == Asc {
		return !a && b
	}
	return a && !b
}
