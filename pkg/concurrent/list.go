package concurrent

import (
	"container/list"
	"sync"
)

type List struct {
	sync.Mutex
	items *list.List
}

func NewList() *List {
	return &List{
		items: list.New(),
	}
}

func (l *List) PopFront() any {
	l.Lock()
	defer l.Unlock()

	item := l.items.Front()
	if item != nil {
		l.items.Remove(item)
		return item.Value
	}

	return nil
}

func (l *List) PushBack(value any) {
	l.Lock()
	defer l.Unlock()

	l.items.PushBack(value)
}

func (l *List) PushFront(value any) {
	l.Lock()
	defer l.Unlock()

	l.items.PushFront(value)
}

func (l *List) PushBackList(other *List) {
	l.Lock()
	defer l.Unlock()

	l.items.PushBackList(other.GetItems())
}

func (l *List) GetItems() *list.List {
	l.Lock()
	defer l.Unlock()

	return l.items
}

func (l *List) Reset() {
	l.Lock()
	defer l.Unlock()

	l.items.Init()
}

func (l *List) Len() int {
	l.Lock()
	defer l.Unlock()

	return l.items.Len()
}
