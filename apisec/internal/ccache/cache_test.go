// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

package ccache

import (
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCache(t *testing.T) {
	t.Run("Get", func(t *testing.T) {
		t.Run("Miss", func(t *testing.T) {
			cache := New[int, int]()
			defer cache.Close()

			assert.Nil(t, cache.Get(1337))
		})

		t.Run("Hit", func(t *testing.T) {
			cache := New[int, int]()
			defer cache.Close()

			setItem := cache.Set(1337, 42)

			getItem := cache.Get(1337)
			require.NotNil(t, getItem)
			require.Equal(t, setItem, getItem)
		})
	})

	t.Run("GetOrStore", func(t *testing.T) {
		cache := New[int, int]()
		defer cache.Close()

		item, found := cache.GetOrStore(1337, func() int { return 42 })
		require.NotNil(t, item)
		assert.False(t, found)
		assert.Equal(t, 42, item.Value())

		item, found = cache.GetOrStore(1337, func() int { return -58008 })
		require.NotNil(t, item)
		assert.True(t, found)
		assert.Equal(t, 42, item.Value())
	})

	t.Run("Set", func(t *testing.T) {
		cache := New[int, int]()
		defer cache.Close()

		item := cache.Set(1337, 42)
		require.NotNil(t, item)
		assert.Equal(t, 1337, item.Key())
		assert.Equal(t, 42, item.Value())

		t.Run("Overwrite", func(t *testing.T) {
			item := cache.Set(1337, -58008)
			require.NotNil(t, item)
			assert.Equal(t, 1337, item.Key())
			assert.Equal(t, -58008, item.Value())
		})
	})

	t.Run("MoveToFront", func(t *testing.T) {
		cache := New[int, int]()
		defer cache.Close()

		var item0 *Item[int, int]
		for i := range maxCount + itemsToPrune {
			item := cache.Set(i, i)
			if i == 0 {
				item0 = item
				continue
			}
			// Keep the item 0 atop the freshness list
			cache.MoveToFront(item0)
		}

		item := cache.Get(0)
		assert.Equal(t, item0, item)
	})

	t.Run("Close", func(t *testing.T) {
		cache := New[int, int]()

		var (
			goroutineCount = runtime.GOMAXPROCS(0) * 10
			barrier        sync.WaitGroup
			closeChan      = make(chan struct{})
			wg             sync.WaitGroup
		)
		barrier.Add(goroutineCount + 1)
		for range goroutineCount {
			wg.Add(1)
			go func() {
				defer wg.Done()

				// Synchronize the start of all the goroutines
				barrier.Done()
				barrier.Wait()

				for {
					// Check whether this is our last action or not... This is done this
					// way to ensure we slot one more request after the cache has been
					// closed before stopping to issue new requests.
					var exit bool
					select {
					case <-closeChan:
						exit = true
					default:
						// Don't wait
					}

					key := rand.Intn(2 * maxCount)
					switch rand.Intn(3) {
					case 0:
						_ = cache.Set(key, key)
					case 1:
						_ = cache.Get(key)
					case 2:
						_, _ = cache.GetOrStore(key, func() int { return key })
					}

					if exit {
						return
					}
				}
			}()
		}
		barrier.Done()

		// In 100 ms, close the cache, and let further requests proceed...
		time.Sleep(10 * time.Millisecond)
		cache.Close()
		close(closeChan)

		// Wait until all goroutines have finished...
		wg.Wait()
	})

	t.Run("gc", func(t *testing.T) {
		cache := New[int, int]()
		defer cache.Close()

		// Insert more items than are allowed to be retained...
		for i := range maxCount + itemsToPrune {
			_ = cache.Set(i, i)
			if i%(promoteBufferCount/2) == 0 {
				// Make sure we don't cause the promotion buffer to become full...
				cache.syncUpdates()
			}
		}

		// Wait for all pending promotes/deletes to have been flushed
		cache.syncUpdates()

		// Now, verify only the last maxCount items are in the cache...
		for i := range maxCount + itemsToPrune {
			item := cache.Get(i)
			if i < itemsToPrune {
				assert.Nil(t, item, "item %d should have been pruned", i)
				continue
			}
			require.NotNil(t, item, "item %d should not have been pruned", i)
			assert.Equal(t, i, item.Value())
		}
	})
}
