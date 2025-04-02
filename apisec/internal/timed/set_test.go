// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

package timed

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DataDog/appsec-internal-go/apisec/internal/config"
	"github.com/stretchr/testify/require"
)

func TestSet(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		require.PanicsWithError(t, "NewSet: interval must be at least 1s, got 0s", func() { NewSet(0, UnixTime) })
		require.PanicsWithError(t, "NewSet: interval must be at least 1s, got 10ms", func() { NewSet(10*time.Millisecond, UnixTime) })
		require.PanicsWithError(t, "NewSet: interval must not exceed 30s, got 1m0s", func() { NewSet(time.Minute, UnixTime) })
	})

	t.Run("Hit", func(t *testing.T) {
		fakeTime := time.Now().Unix()
		fakeClock := func() int64 { return fakeTime }

		subject := NewSet(config.Interval, fakeClock)

		require.True(t, subject.Hit(1337))
		for range config.Interval / time.Second {
			require.False(t, subject.Hit(1337))
			fakeTime++
		}
		require.True(t, subject.Hit(1337))

		t.Run("zero", func(t *testing.T) {
			require.True(t, subject.Hit(0))

			// Keys are slotted via [% capacity], so if we don't properly encode
			// 0-values, the new slot will inherit the previously set sample time, and
			// the assertion will fail as a result.
			zeroSlot := uint64(capacity)
			if zeroSlot == subject.zeroKey {
				// There is a very small chance that the zero key has been set to
				// [capacity], in which case we'll just double it to escape the
				// collision and get a fresh new hit.
				zeroSlot *= 2
			}
			require.True(t, subject.Hit(zeroSlot))
		})
	})

	t.Run("rebuild", func(t *testing.T) {
		var fakeTime atomic.Int64
		fakeTime.Store(time.Now().Unix())
		fakeClock := func() int64 { return fakeTime.Load() }

		subject := NewSet(config.Interval, fakeClock)

		var (
			goCount       = runtime.GOMAXPROCS(0) * 10
			startBarrier  sync.WaitGroup
			finishBarrier sync.WaitGroup
		)
		startBarrier.Add(goCount + 1)
		finishBarrier.Add(goCount)
		for g := range goCount {
			go func() {
				defer finishBarrier.Done()
				startBarrier.Done()
				startBarrier.Wait()

				for key := range uint64(config.MaxItemCount * 4) {
					_ = subject.Hit(key)
					if g == 0 {
						fakeTime.Add(1)
					}
				}
			}()
		}

		startBarrier.Done()
		finishBarrier.Wait()

		// Wiat for an in-progress rebuild to finish...
		for subject.rebuilding.Load() {
			runtime.Gosched()
		}

		// Check the final table has a reasonable content...
		table := subject.table.Load()
		count := 0
		for i := range table.entries {
			entry := &table.entries[i]
			if entry.Key.Load() == 0 {
				continue
			}
			// Since we ran through the keys sequentially, we should not have kept any
			// of the first [config.MaxItemCount] keys in any case.
			require.Less(t, uint64(config.MaxItemCount), entry.Key.Load())
			count++
		}
		// We shoudl not have more than [maxItemCount] items left in the map...
		require.LessOrEqual(t, count, config.MaxItemCount)
	})
}
