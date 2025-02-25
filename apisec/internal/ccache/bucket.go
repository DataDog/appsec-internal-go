// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

package ccache

import (
	"sync"
)

type bucket[K cacheKey, V any] struct {
	sync.RWMutex
	lookup map[K]*Item[K, V]
}

// delete removes the item with the given key from the bucket and returns it.
func (b *bucket[K, V]) delete(key K) *Item[K, V] {
	b.Lock()
	item := b.lookup[key]
	delete(b.lookup, key)
	b.Unlock()
	return item
}

// get returns the item with the given key from the bucket, or nil if no such
// item exists.
func (b *bucket[K, V]) get(key K) *Item[K, V] {
	b.RLock()
	item := b.lookup[key]
	b.RUnlock()
	return item
}

// getOrStore returns the item with the given key from the bucket if one exists,
// or inserts a new item with the given key and the value obtained from the load
// callback into the bucket and returns the newly created item.
func (b *bucket[K, V]) getOrStore(key K, load func() V) (item *Item[K, V], existed bool) {
	item = b.get(key)
	if item != nil {
		return item, true
	}

	b.Lock()
	defer b.Unlock()

	// Another goroutine might have stored an item while we were waiting for the
	// lock, so we check again here...
	item = b.lookup[key]
	if item != nil {
		return item, true
	}

	item = newItem(key, load())
	b.lookup[key] = item

	return item, false
}

// set adds a new item to the bucket, or replaces an existing item with the new
// one. It returns the newly inserted item, as well as the previous item if one
// existed.
func (b *bucket[K, V]) set(key K, value V) (item *Item[K, V], oldItem *Item[K, V]) {
	item = newItem(key, value)

	b.Lock()
	oldItem = b.lookup[key]
	b.lookup[key] = item
	b.Unlock()

	return item, oldItem
}
