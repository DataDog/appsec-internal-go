// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

package apisec

import (
	"encoding/binary"
	"hash/fnv"
	"time"

	"github.com/DataDog/appsec-internal-go/apisec/internal/timed"
)

type (
	Sampler interface {
		DecisionFor(SamplingKey) bool
	}

	timedSetSampler timed.LRU

	SamplingKey struct {
		// Method is the value of the http.method span tag
		Method string
		// Route is the value of the http.route span tag
		Route string
		// StatusCode is the value of the http.status_code span tag
		StatusCode int
	}

	clockFunc = func() int64
)

// NewSamplerWithInterval returns a new [*Sampler] with the specified interval.
func NewSamplerWithInterval(interval time.Duration) Sampler {
	return newSampler(interval, timed.UnixTime)
}

// newSampler allows creating a new [*Sampler] with custom clock function,
// which is useful for testing.
func newSampler(interval time.Duration, clock clockFunc) Sampler {
	return (*timedSetSampler)(timed.NewLRU(interval, clock))
}

// DecisionFor makes a sampling decision for the provided [SamplingKey]. If it
// returns true, the request has been "sampled in" and the caller should proceed
// with the necessary actions. If it returns false, the request has been
// dropped, and the caller should short-circuit without extending further
// effort.
func (s *timedSetSampler) DecisionFor(key SamplingKey) bool {
	keyHash := key.hash()
	return (*timed.LRU)(s).Hit(keyHash)
}

// hash returns a hash of the key. Given the same seed, it always produces the
// same output. If the seed changes, the output is likely to change as well.
func (k SamplingKey) hash() uint64 {
	fnv := fnv.New64()

	_, _ = fnv.Write([]byte(k.Method))
	_, _ = fnv.Write([]byte(k.Route))

	var bytes [2]byte
	binary.NativeEndian.PutUint16(bytes[:], uint16(k.StatusCode))
	_, _ = fnv.Write(bytes[:])

	return fnv.Sum64()
}
