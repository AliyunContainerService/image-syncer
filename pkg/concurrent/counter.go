package concurrent

type Counter struct {
	c     chan struct{}
	count int
}

func NewCounter(count int) *Counter {
	return &Counter{
		c:     make(chan struct{}, 1),
		count: count,
	}
}

func (c *Counter) Decrease() {
	c.c <- struct{}{}
	defer func() {
		<-c.c
	}()

	if c.count > 0 {
		c.count--
	}
}

func (c *Counter) Value() int {
	c.c <- struct{}{}
	defer func() {
		<-c.c
	}()

	return c.count
}
