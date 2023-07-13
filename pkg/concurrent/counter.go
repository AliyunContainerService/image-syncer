package concurrent

type Counter struct {
	c     chan struct{}
	count int
	total int
}

func NewCounter(count, total int) *Counter {
	return &Counter{
		c:     make(chan struct{}, 1),
		count: count,
		total: total,
	}
}

func (c *Counter) Decrease() (int, int) {
	c.c <- struct{}{}
	defer func() {
		<-c.c
	}()

	if c.count > 0 {
		c.count--
	}
	return c.count, c.total
}

func (c *Counter) Increase() (int, int) {
	c.c <- struct{}{}
	defer func() {
		<-c.c
	}()

	if c.count < c.total {
		c.count++
	}
	return c.count, c.total
}

func (c *Counter) IncreaseTotal() (int, int) {
	c.c <- struct{}{}
	defer func() {
		<-c.c
	}()

	c.total++
	return c.count, c.total
}

// Value return count and total
func (c *Counter) Value() (int, int) {
	c.c <- struct{}{}
	defer func() {
		<-c.c
	}()

	return c.count, c.total
}
