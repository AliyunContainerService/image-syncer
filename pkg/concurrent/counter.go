package concurrent

import "sync"

type Counter struct {
	sync.Mutex
	count int
	total int
}

func NewCounter(count, total int) *Counter {
	return &Counter{
		count: count,
		total: total,
	}
}

func (c *Counter) Decrease() (int, int) {
	c.Lock()
	defer c.Unlock()

	if c.count > 0 {
		c.count--
	}
	return c.count, c.total
}

func (c *Counter) Increase() (int, int) {
	c.Lock()
	defer c.Unlock()

	if c.count < c.total {
		c.count++
	}
	return c.count, c.total
}

func (c *Counter) IncreaseTotal() (int, int) {
	c.Lock()
	defer c.Unlock()

	c.total++
	return c.count, c.total
}

// Value return count and total
func (c *Counter) Value() (int, int) {
	c.Lock()
	defer c.Unlock()

	return c.count, c.total
}
