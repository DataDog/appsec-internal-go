// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

package timed

import (
	"container/heap"
	"sync/atomic"

	"github.com/DataDog/appsec-internal-go/apisec/internal/config"
)

type (
	// table is a simple open-addressing hash table implementation that uses a
	// fixed array of items.
	table struct {
		// entries is the set of items contained in the table. The last entry is
		// reserved for cases where all slots are taken before a rebuild is
		// complete (it saves us from having to write code to deal with the
		// impossibility to find an empty slot, as we always have a slot to return.
		// We could return a throw-away slot but this would incur a heap allocation,
		// which we can spare by doing this).
		entries [capacity + 1]entry
		// count is the number of items currently stored in the table.
		count atomic.Int32
	}

	// entry is a single item in the open-addressing hash table.
	entry struct {
		// Key is the Key of the entry. A zero Key indicates that the entry is
		// currently free.
		Key atomic.Uint64
		// Data is the Data associated with the entry.
		Data atomicEntryData
	}

	// atomicEntryData is an atomic version of [entryData].
	atomicEntryData atomic.Uint64
	// entryData is a 64-bit value that represents the last time an entry was
	// accessed paired together with the last time this value was sampled.
	entryData uint64

	// copiableEntry is a copy-able version of entryData, which is used for
	// sorting entries by recency using a heap when re-building the table.
	copiableEntry struct {
		// Key is the Key of the entry.
		Key uint64
		// Data is the Data associated with the entry.
		Data entryData
	}
	// sortedEntries is a [heap.Interface] used to sort [copiableEntry] when
	// rebuilding the table in order to evict older items.
	sortedEntries []copiableEntry
)

// FindEntry locates the correct entry for use in the table. If an entry already
// exists for the given key, it is returned with true. If not, the first blank
// entry is returned with false.
func (t *table) FindEntry(key uint64) (*entry, bool) {
	origIdx := key % capacity
	idx := origIdx

	for {
		entry := &t.entries[idx]
		if curKey := entry.Key.Load(); curKey == 0 || curKey == key {
			// This is either the entry we're looking for, or an empty slot we can
			// claim for this key.
			return entry, curKey == key
		}
		idx = (idx + 1) % capacity
		if idx == origIdx {
			// We are back at the original index, meaning the map is full.
			break
		}
	}
	// We have gone full circle without finding a blank slot, so we give up and
	// return our last resort slot that is reserved for this situation.
	return &t.entries[capacity], true
}

// PrunedCopy creates a copy of this table with expired items removed, retaining
// up to the [config.MaxItemCount]*2/3 most recent items from the original.
func (t *table) PrunedCopy(threshold uint32) *table {
	// Sort the existing entries (most recent at the top)
	newEntries := make(sortedEntries, 0, len(t.entries))
	heap.Init(&newEntries)
	for i := range t.entries {
		entry := &t.entries[i]
		if entry.Key.Load() == 0 || entry.Data.Load().SampleTime() < threshold {
			// Entry is blank or expired, not carrying it around...
			continue
		}
		heap.Push(&newEntries, entry.Copyable())
	}

	// Insert up to [config.MaxItemCount]*2/3 items into the new table
	var count int32
	t = &table{}
	for newEntries.Len() != 0 && count < config.MaxItemCount*2/3 {
		entry, _ := heap.Pop(&newEntries).(copiableEntry)
		slot, _ := t.FindEntry(entry.Key)
		slot.Key.Store(entry.Key)
		slot.Data.Store(entry.Data)
		count++
	}
	t.count.Store(count)

	return t
}

// Copyable returns a [copyableEntry] version of this entry.
func (e *entry) Copyable() copiableEntry {
	return copiableEntry{
		Key:  e.Key.Load(),
		Data: e.Data.Load(),
	}
}

// Load returns the current value held by this atomic.
func (d *atomicEntryData) Load() entryData {
	return entryData((*atomic.Uint64)(d).Load())
}

// CompareAndSwap atomically compares the current value held by this atomic with
// the old value, and if they match replaes it with the new value. Returns true
// if the swap happened.
func (d *atomicEntryData) CompareAndSwap(old entryData, new entryData) (swapped bool) {
	return (*atomic.Uint64)(d).CompareAndSwap(uint64(old), uint64(new))
}

// Store atomically stores the given value in this atomic.
func (d *atomicEntryData) Store(new entryData) {
	(*atomic.Uint64)(d).Store(uint64(new))
}

// newEntryData creates a new [entryData] value from the given access and sample
// times.
func newEntryData(atime uint32, stime uint32) entryData {
	return entryData(uint64(atime)<<32 | uint64(stime))
}

// AccessTime is the access time part of the [entryData].
func (d entryData) AccessTime() uint32 {
	return uint32(d >> 32)
}

// SampleTime is the sample time part of the [entryData].
func (d entryData) SampleTime() uint32 {
	return uint32(d)
}

// LastAccessKept returns true if the last access to this entry resulted in a
// decision to keep the sample. This is true of the access time is not 0 and is
// equal to the sample time.
func (d entryData) LastAccessKept() bool {
	return d.AccessTime() != 0 && d.AccessTime() == d.SampleTime()
}

// WithAccessTime returns a new [entryData] by copying the receiver and
// replacing the access time portion with the specified value.
func (d entryData) WithAccessTime(atime uint32) entryData {
	return (d & 0x00000000_FFFFFFFF) | (entryData(atime) << 32)
}

var _ heap.Interface = (*sortedEntries)(nil)

// Len implements [heap.Interface.Len].
func (n *sortedEntries) Len() int {
	return len(*n)
}

// Less implements [heap.Interface.Less].
func (n *sortedEntries) Less(i int, j int) bool {
	// We are sorting "most recent first", so higher access time is lower in the
	// heap's definition.
	iatime := (*n)[i].Data.AccessTime()
	jatime := (*n)[j].Data.AccessTime()
	return iatime > jatime
}

// Swap implements [heap.Interface.Swap].
func (n *sortedEntries) Swap(i int, j int) {
	(*n)[i], (*n)[j] = (*n)[j], (*n)[i]
}

// Push implements [heap.Interface.Push].
func (n *sortedEntries) Push(x any) {
	*n = append(*n, x.(copiableEntry))
}

// Pop implements [heap.Interface.Pop].
func (n *sortedEntries) Pop() any {
	var res any
	res, *n = (*n)[len(*n)-1], (*n)[:len(*n)-1]
	return res
}
