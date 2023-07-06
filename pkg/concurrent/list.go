package concurrent

import (
	"container/list"
)

type List struct {
	c     chan struct{}
	items *list.List
}

func NewList() *List {
	return &List{
		c:     make(chan struct{}, 1),
		items: list.New(),
	}
}

func (l *List) PopFront() any {
	l.c <- struct{}{}
	defer func() {
		<-l.c
	}()

	item := l.items.Front()
	if item != nil {
		l.items.Remove(item)
		return item.Value
	}

	return nil
}

func (l *List) PushBack(value any) {
	l.c <- struct{}{}
	defer func() {
		<-l.c
	}()

	l.items.PushBack(value)
}

func (l *List) PushBackList(other *List) {
	l.c <- struct{}{}
	defer func() {
		<-l.c
	}()

	l.items.PushBackList(other.GetItems())
}

func (l *List) GetItems() *list.List {
	l.c <- struct{}{}
	defer func() {
		<-l.c
	}()

	return l.items
}

func (l *List) Reset() {
	close(l.c)
	l.c = make(chan struct{}, 1)
	l.items.Init()
}

func (l *List) Len() int {
	return l.items.Len()
}
