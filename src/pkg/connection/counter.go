package connection

import "sync/atomic"

var (
	// globalCounter is the global connection counter
	GlobalCounter = &Counter{count: 0}
)

// Counter is a connection counter
type Counter struct {
	count int64
}

// Inc increments the counter by 1
func (c *Counter) Inc() {
	atomic.AddInt64(&c.count, 1)
}

// Dec decrements the counter by 1
func (c *Counter) Dec() {
	atomic.AddInt64(&c.count, -1)
}

// Count returns the current count
func (c *Counter) Count() int64 {
	return atomic.LoadInt64(&c.count)
}
