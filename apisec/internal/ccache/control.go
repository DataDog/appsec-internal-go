// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

package ccache

import "time"

type (
	// control is a channel used to manage the run-time state of the [Cache] via
	// the [Cache.worker] goroutine.
	control chan controlMessage

	// controlMessage is the type of value sent over the [control] channel. It is
	// a simple marker interface to make mis-use more complicated.
	controlMessage interface{ controlMessage() }

	// controlSyncUpdate is used to process all pending items from
	// [Cache.deletables] and [Cache.promotables] before sending a value to the
	// [controlSyncUpdates.done] channel.
	controlSyncUpdates struct{ done chan<- struct{} }
	// controlStop is used to instruct the [Cache.worker] goroutine to exit while
	// allowing further [Cache.deletables] and [Cache.promotables] to be processed
	// for up to [controlStop.timeout].
	controlStop struct{ timeout time.Duration }
)

// newControl creates a new [control] channel with an appropriately sized
// buffer.
func newControl() control {
	return make(control, 2)
}

// Close is the same as [control.CloseWithTimeout] with the default timeout of
// 5 seconds.
func (c control) Close() {
	c.CloseWithTimeout(5 * time.Second)
}

// CloseWithTimeout sends a signal on the control channel to stop the
// [Cache.worker] goroutine after the timeout has passed. Past this delay, the
// [Cache.promotables] and [Cache.deletables] channels will no longer be read
// from, which will cause deletions from the cache (including those induced by
// [Cache.Set]) to permanently block once the buffer is full.
func (c control) CloseWithTimeout(timeout time.Duration) {
	c.syncUpdates()
	c <- controlStop{timeout}
	close(c)
}

// syncUpdates waits until all pending updates are processed. This is useful for
// testing, as it allows one to deterministically wait for all side-effects to
// have been processed before asserting them.
func (c control) syncUpdates() {
	done := make(chan struct{})
	c <- controlSyncUpdates{done: done}
	<-done
}

func (controlSyncUpdates) controlMessage() {}
func (controlStop) controlMessage()        {}
