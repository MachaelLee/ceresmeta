// Copyright 2022 CeresDB Project Authors. Licensed under Apache-2.0.

package id

import (
	"context"
	"sort"
	"sync"
)

type ReusableAllocatorImpl struct {
	// Mutex is used to protect following fields.
	lock sync.Mutex

	minID    uint64
	existIDs *OrderedList
}

type OrderedList struct {
	sorted []uint64
}

// FindMinHoleValueAndIndex Find the minimum hole value and its index.
// If the list is empty, then return min value and 0 as index;
// If no hole is found, then return the `last_value + 1` in the list and l.Len() as the index;
func (l *OrderedList) FindMinHoleValueAndIndex(min uint64) (uint64, int) {
	if len(l.sorted) == 0 {
		return min, 0
	}
	if l.sorted[0] > min {
		return min, 0
	}
	if len(l.sorted) == 1 {
		return l.sorted[0] + 1, 1
	}

	s := l.sorted
	for i := 0; i < len(l.sorted)-1; i++ {
		if s[i]+1 != s[i+1] {
			return s[i] + 1, i + 1
		}
	}

	return s[len(s)-1] + 1, len(s)
}

// Insert the value at the idx whose correctness should be ensured by the caller.
func (l *OrderedList) Insert(v uint64, i int) {
	if len(l.sorted) == i {
		l.sorted = append(l.sorted, v)
	} else {
		l.sorted = append(l.sorted[:i+1], l.sorted[i:]...)
		l.sorted[i] = v
	}
}

func (l *OrderedList) Remove(v uint64) int {
	removeIndex := -1
	for i, value := range l.sorted {
		if value == v {
			removeIndex = i
		}
	}
	l.sorted = append(l.sorted[:removeIndex], l.sorted[removeIndex+1:]...)
	return removeIndex
}

func NewReusableAllocatorImpl(existIDs []uint64, minID uint64) Allocator {
	sort.Slice(existIDs, func(i, j int) bool {
		return existIDs[i] < existIDs[j]
	})
	return &ReusableAllocatorImpl{minID: minID, existIDs: &OrderedList{sorted: existIDs}}
}

func (a *ReusableAllocatorImpl) Alloc(_ context.Context) (uint64, error) {
	a.lock.Lock()
	defer a.lock.Unlock()
	// Find minimum unused ID bigger than minID
	v, i := a.existIDs.FindMinHoleValueAndIndex(a.minID)
	a.existIDs.Insert(v, i)
	return v, nil
}

func (a *ReusableAllocatorImpl) Collect(_ context.Context, id uint64) error {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.existIDs.Remove(id)
	return nil
}
