package testutil

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type FakeClock struct {
	t        testing.TB
	closed   chan struct{}
	goCount  int
	now      int64
	needTick atomic.Bool
	wg       sync.WaitGroup
	cnd      sync.Cond
}

func NewFakeClock(ctx context.Context, t testing.TB, goroutineCount int) *FakeClock {
	res := &FakeClock{
		t:       t,
		closed:  make(chan struct{}),
		goCount: goroutineCount,
		now:     time.Now().Unix(),
		cnd:     *sync.NewCond(&sync.Mutex{}),
	}
	res.wg.Add(goroutineCount)
	go res.tick(ctx)
	return res
}

func (c *FakeClock) Unix() int64 {
	return c.now
}

func (c *FakeClock) WaitForTick() {
	c.cnd.L.Lock()

	c.needTick.CompareAndSwap(false, true)
	curTime := c.now

	c.wg.Done()
	for curTime == c.now && c.now > 0 {
		c.cnd.Wait()
	}
	c.cnd.L.Unlock()
}

func (c *FakeClock) WaitUntilDone() {
	<-c.closed
}

func (c *FakeClock) tick(ctx context.Context) {
	c.t.Log("Start ticker!")
	for {
		select {
		case <-ctx.Done():
			// Context is cancelled, stop the ticker...
			c.cnd.L.Lock()
			c.now = 0
			c.cnd.L.Unlock()
			c.t.Log("Stop ticker!")
			close(c.closed)
			return
		default:
			if !c.needTick.Load() {
				continue
			}
			c.wg.Wait()
			c.cnd.L.Lock()
			c.now++
			c.needTick.Store(false)
			c.cnd.Broadcast()
			c.wg.Add(c.goCount)
			c.cnd.L.Unlock()
		}
	}
}
