// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

package ccache

// Item is an entry in the cache; tracking a key and value together with the
// necessary state.
type Item[K cacheKey, V any] struct {
	node    *listNode[*Item[K, V]] // The recency list node for this item (if it's in the list)
	key     K                      // The key of this item
	value   V                      // The value of this item
	deleted bool                   // True if the item was deleted already
}

// newItem initializes a new [Item] with the given key and value.
func newItem[K cacheKey, V any](key K, value V) *Item[K, V] {
	return &Item[K, V]{
		key:   key,
		value: value,
	}
}

// Key returns the key associated with this [Item].
func (i *Item[K, V]) Key() K {
	return i.key
}

// Key returns the value associated with this [Item].
func (i *Item[K, V]) Value() V {
	return i.value
}
