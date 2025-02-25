// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

// Package ccache is a "trimmed down" version of [github.com/karlseguin/ccache/v3]
// that only has the simple features (no custom expiry times, no hooks, no
// tracking, etc...) and that is adapted to work off integer keys instead of
// strings.
// It has also been adapted slightly for the specific use-case at hand,
// including:
//   - introduction of [Cache.MoveToFront] method to promote items to the font of
//     the recency queue without incurring a deletion operation;
//   - introduction of [Cache.GetOrStore] method to atomically check for existence
//     of an item, and creating of a new if needed; reducing the risk of
//     dramatically over-sampling if a given key suddenly becomes wildly popular.
package ccache

import (
	"context"
	"time"
)

const (
	maxCount           = 4_096         // Maximum number of items held in the cache
	itemsToPrune       = maxCount / 4  // Number of items to prune when the cache is full
	promoteBufferCount = maxCount / 16 // Size of the promotion queue. If full, promotions are not done
	deleteBufferCount  = maxCount / 4  // Size of the deletion queue. If full, calls to [Cache.Set] will block when replacing values
)

type (
	Cache[K cacheKey, V any] struct {
		control                      // Control channel for the [Cache]
		deletables  chan *Item[K, V] // Ask [Cache.worker] to remove an item from the recency list
		promotables chan *Item[K, V] // Ask [Cache.worker] to move an item to the front of the recency list
		buckets     []bucket[K, V]   // Buckets for the cache's backing store

		// Maintained by [Cache.worker] only
		list  *linkedList[*Item[K, V]] // Recency list
		count uint                     // Current item count

		// Constant for operations
		bucketMask uint32
	}

	cacheKey interface {
		int32 | int64 | uint32 | uint64 | int | uint | uintptr
	}
)

// New creates a new cache and starts its worker goroutine. [Cache.Close] must
// be called after it ceases to be useful so that the worker goroutine exits.
// The cache will be sharded in 2^4=16 buckets, to use a different shard count,
// use [NewWithShardBits] function instead.
func New[K cacheKey, V any]() *Cache[K, V] {
	return NewWithShardBits[K, V](4)
}

// New creates a new cache and starts its worker goroutine. [Cache.Close] must
// be called after it ceases to be useful so that the worker goroutine exits.
// The cache will be sharded in 2^bucketCountPow buckets.
func NewWithShardBits[K cacheKey, V any](bucketCountPow uint8) *Cache[K, V] {
	bucketCount := uint32(1) << bucketCountPow
	bucketMask := bucketCount - 1 // Effectively the bucketCountPow low bits

	cache := &Cache[K, V]{
		control:     newControl(),
		list:        newList[*Item[K, V]](),
		buckets:     make([]bucket[K, V], bucketCount),
		deletables:  make(chan *Item[K, V], deleteBufferCount),
		promotables: make(chan *Item[K, V], promoteBufferCount),
		bucketMask:  bucketMask,
	}

	for i := range bucketCount {
		cache.buckets[i].lookup = make(map[K]*Item[K, V])
	}

	go cache.worker()

	return cache
}

// Get retrieves the [Item] associated with the supplied key. If no such item
// exist, nil is returned. This does not move the item to the front of the
// recency queue.
func (c *Cache[K, V]) Get(key K) *Item[K, V] {
	return c.bucket(key).get(key)
}

// GetOrStore retrieves the [Item] associated with the supplied key. If no such
// item exist, the value returned by the provided load callback is stored, and
// the new [Item] is returned. Use of the callback allows avoiding unnecessary
// allocations. Existing items are not moved to the front of the recency queue,
// but new items are added at the front of the recency queue.
func (c *Cache[K, V]) GetOrStore(key K, load func() V) (*Item[K, V], bool) {
	item, found := c.bucket(key).getOrStore(key, load)
	if !found {
		// This is a new item, need to add it ahead of the recency queue
		c.MoveToFront(item)
	}
	return item, found
}

// MoveToFront moves the supplied item to the front of the recency list; meaning
// it becomes the last item eligible for pruning. If the promotion queue is
// full, this does nothing and immediately returns false.
func (c *Cache[K, V]) MoveToFront(item *Item[K, V]) bool {
	select {
	case c.promotables <- item:
		// There was space in the queue, so we've successfully registered!
		return true

	default:
		// There was no space in the queue, so we're ignoring this request...
		return false
	}
}

// Set adds or replaces an item to the [Cache], and returns the associated
// [Item]. [Cache.MoveToFront] is called on the returned [Item], promoting it
// to most recently used unless the promotion queue is already full. If an
// existing value is being replaced, its [Item] is added to the deletion queue,
// blocking if it is already full.
func (c *Cache[K, V]) Set(key K, value V) *Item[K, V] {
	item, oldItem := c.bucket(key).set(key, value)
	if oldItem != nil {
		// We replaced an existing item, so need to remove the old one from the
		// recency queue, as it's no longer in store.
		c.deletables <- oldItem
	}
	c.MoveToFront(item)
	return item
}

// bucket returns the bucket assoeciated with the supplied key.
func (c *Cache[K, V]) bucket(key K) *bucket[K, V] {
	slot := uint32(key) & c.bucketMask
	return &c.buckets[slot]
}

// doDelete performs the deletion of the provided [Item]. This must only be
// called from the [Cache.worker] goroutine.
func (c *Cache[K, V]) doDelete(item *Item[K, V]) {
	item.deleted = true
	if item.node == nil {
		// That was already deleted (or never inserted in the first place)
		return
	}
	c.count--
	c.list.Remove(item.node)
	item.node = nil
}

// doPromote performs the promotion of the provided [Item], placing it ahead of
// the recency list. This must only be called from the [Cache.worker] goroutine.
// Returns true if the item was added to the recency list, false if it was
// already there, or has already been evicted.
func (c *Cache[K, V]) doPromote(item *Item[K, V]) bool {
	if item.deleted {
		// Already deleted, not promoting anymore...
		return false
	}

	if item.node != nil {
		// Not a new item, so we just move it to the front of the list.
		c.list.MoveToFront(item.node)
		return false
	}

	// New item, so we insert it right at the front of the queue.
	c.count++
	item.node = c.list.Insert(item)
	return true
}

// gc prunes old items from the cache, making space for new items. This must
// only be called from the [Cache.worker] goroutine.
func (c *Cache[K, V]) gc() int {
	dropped := 0
	node := c.list.Tail

	itemsToPrune := uint(itemsToPrune)
	if delta := c.count - maxCount; delta > itemsToPrune {
		itemsToPrune = delta
	}

	for range itemsToPrune {
		if node == nil {
			break
		}

		item := node.Value
		c.bucket(item.key).delete(item.key)
		c.count--
		item.deleted = true
		c.list.Remove(node)
		item.node = nil
		dropped++

		node = node.Prev
	}

	return dropped
}

// worker is the goroutine that maintains the cache. It receives from
// [Cache.deletables], [Cache.promotables], and processes messages from
// [Cache.control]. It exits after [Cache.Stop] is called; once outstanding
// queued items are processed.
func (c *Cache[K, V]) worker() {
	dropped := 0

	promoteItem := func(item *Item[K, V]) {
		if c.doPromote(item) && c.count > maxCount {
			dropped += c.gc()
		}
	}

	drain := func(timeout time.Duration) {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// Start by completely draining the promotion and deletion queues... This is
		// done in an IIFE to make breaking out of the for loop easier.
		func() {
			for {
				select {
				case item := <-c.deletables:
					c.doDelete(item)
				case item := <-c.promotables:
					promoteItem(item)
				default:
					return
				}
			}
		}()

		// Now, contribue processing items from the promotion and deletion queues
		// until the context expires.
		for {
			select {
			case item := <-c.deletables:
				c.doDelete(item)
			case item := <-c.promotables:
				promoteItem(item)
			case <-ctx.Done():
				return
			}
		}
	}

	for {
		select {
		case item := <-c.deletables:
			// Actually delete old items from the recency queue
			c.doDelete(item)

		case item := <-c.promotables:
			// Add new items (or move existing items) to the front of the recency queue
			promoteItem(item)

		case ctrl := <-c.control:
			switch ctrl := ctrl.(type) {
			case controlStop:
				// [control.Close] has been called, stop operations...
				drain(ctrl.timeout)
				return // Goroutine exits after draining

			case controlSyncUpdates:
				// [control.syncUpdates] was called, process all pending promotions &
				// deletions synchronously. This is done in an IIFE to make breaking out
				// of that for loop easier.
				func() {
					for {
						select {
						case item := <-c.deletables:
							c.doDelete(item)

						case item := <-c.promotables:
							promoteItem(item)

						default:
							ctrl.done <- struct{}{}
							return
						}
					}
				}()
			}
		}
	}
}
