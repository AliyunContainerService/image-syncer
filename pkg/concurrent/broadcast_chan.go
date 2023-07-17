package concurrent

type BroadcastChan struct {
	c             chan struct{}
	subscriberNum int

	hungCounter   *Counter
	totalHungChan chan struct{}
}

func NewBroadcastChan(subscriberNum int) *BroadcastChan {
	return &BroadcastChan{
		c:             make(chan struct{}, subscriberNum),
		subscriberNum: subscriberNum,

		hungCounter:   NewCounter(0, subscriberNum),
		totalHungChan: make(chan struct{}, 1),
	}
}

func (b *BroadcastChan) Close() {
	close(b.c)
}

func (b *BroadcastChan) Broadcast() {
	for i := 0; i < b.subscriberNum; i++ {
		select {
		case b.c <- struct{}{}:
		default:
			continue
		}
	}
}

func (b *BroadcastChan) Wait() bool {
	value, _ := b.hungCounter.Increase()
	if value == b.subscriberNum {
		select {
		case b.totalHungChan <- struct{}{}:
		default:
		}
	}

	defer func() {
		b.hungCounter.Decrease()
	}()

	_, ok := <-b.c
	return !ok
}

func (b *BroadcastChan) TotalHungChan() <-chan struct{} {
	return b.totalHungChan
}
