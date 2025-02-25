// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

package apisec

import (
	"encoding/binary"
	"hash/fnv"
	"math/rand/v2"
	"sync/atomic"
	"time"

	"github.com/DataDog/appsec-internal-go/apisec/internal/ccache"
)

const (
	maxItemCount    = 4_096
	defaultInterval = 30 * time.Second
)

type (
	Sampler struct {
		// msSinceEpoch returns the current timestamp, as milliseconds since Epoch.
		msSinceEpoch msSinceEpochFunc
		// timeSince returns the elapsed duration since the provided timestamp,
		// which is expressed as milliseconds since Epoch.
		timeSince timeSinceMsSinceEpochFunc

		// times holds the last timestamp (as milliseconds since Epoch) when a
		// sample was taken for a [SamplingKey] that hashed to this particular slot.
		times *ccache.Cache[uint64, *atomic.Int64]

		// seed is used to seed the CRC64 hash function, so that different instances
		// slot samples based on different hashes. It is randomly assigned by
		// [NewSampler] and remains constant for the lifetime of the sampler.
		seed uint64

		// interval controls the minimum duration between two sambles being taken.
		interval time.Duration
	}

	SamplingKey struct {
		// Method is the value of the http.method span tag
		Method string
		// Route is the value of the http.route span tag
		Route string
		// StatusCode is the value of the http.status_code span tag
		StatusCode int
	}

	msSinceEpochFunc          = func() int64
	timeSinceMsSinceEpochFunc = func(int64) time.Duration
)

// NewSampler returns a new [*Sampler] with the default clock functions based on
// [time.Now] and [time.Since].
func NewSampler() *Sampler {
	return newSampler(30*time.Second, msSinceEpoch, timeSinceMsSinceEpoch)
}

// NewSamplerWithInterval returns a new [*Sampler] with the specified interval
// instead of the default of 30 seconds.
func NewSamplerWithInterval(interval time.Duration) *Sampler {
	return newSampler(interval, msSinceEpoch, timeSinceMsSinceEpoch)
}

// newSampler allows creating a new [*Sampler] with custom clock functions,
// which is useful for testing.
func newSampler(interval time.Duration, msSinceEpoch msSinceEpochFunc, timeSince timeSinceMsSinceEpochFunc) *Sampler {
	return &Sampler{
		msSinceEpoch: msSinceEpoch,
		timeSince:    timeSince,
		times:        ccache.New[uint64, *atomic.Int64](),
		seed:         rand.Uint64(),
		interval:     interval,
	}
}

// Close releases all resources associated with this [Sampler]. It must be
// called before disposing of the instance. A [Sampler] can no longer be used
// after [Sampler.Close] has been called.
func (s *Sampler) Close() {
	s.times.Close()
	s.times = nil
}

// DecisionFor makes a sampling decision for the provided [SamplingKey]. If it
// returns true, the request has been "sampled in" and the caller should proceed
// with the necessary actions. If it returns false, the request has been
// dropped, and the caller should short-circuit without extending further
// effort.
func (s *Sampler) DecisionFor(key SamplingKey) bool {
	keyHash := key.hash(s.seed)

	now := msSinceEpoch()

	item, loaded := s.times.GetOrStore(keyHash, func() *atomic.Int64 {
		return &atomic.Int64{}
	})
	timer := item.Value()
	if !loaded {
		// If we're the one swapping 0 out, we're the first to sample this new key.
		// Otherwise, a concurrent goroutine did it, so we should not sample again.
		return timer.CompareAndSwap(0, now)
	}

	lastSampleSecs := timer.Load()
	if s.timeSince(lastSampleSecs) < s.interval {
		// Too soon to sample again
		return false
	}

	if !timer.CompareAndSwap(lastSampleSecs, now) {
		// Another goroutine has sampled at the same time
		return false
	}

	// Move to the front of the recency queue, as this is not the last sampled
	// item.
	s.times.MoveToFront(item)
	return true
}

// hash returns a hash of the key. Given the same seed, it always produces the
// same output. If the seed changes, the output is likely to change as well.
func (k SamplingKey) hash(seed uint64) uint64 {
	fnv := fnv.New64()

	// First, hash the seed/salt.
	var bytes [8]byte
	binary.NativeEndian.PutUint64(bytes[:], seed)
	_, _ = fnv.Write(bytes[:])

	// Then hash the method followed by a 0-byte
	_, _ = fnv.Write([]byte(k.Method))
	_, _ = fnv.Write([]byte{0})

	// Then hash the route followed by a 0-byte
	_, _ = fnv.Write([]byte(k.Route))
	_, _ = fnv.Write([]byte{0})

	// Finally, hash the status code (assumed to fit in 16 bits, as these are HTTP
	// status codes, so they all should be >=100 <600).
	binary.NativeEndian.PutUint16(bytes[:2], uint16(k.StatusCode))
	_, _ = fnv.Write(bytes[:2])

	return fnv.Sum64()
}

func msSinceEpoch() int64 {
	return time.Now().UnixMilli()
}

func timeSinceMsSinceEpoch(ms int64) time.Duration {
	t := time.UnixMilli(ms)
	return time.Since(t)
}
