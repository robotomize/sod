package iqueue

import (
	"container/list"
)

func New() *Queue {
	return &Queue{
		queue: list.New(),
		send:  make(chan interface{}, 1),
		recv:  make(chan interface{}, 1),
	}
}

type Queue struct {
	queue *list.List
	send  chan interface{}
	recv  chan interface{}
}

func (iq *Queue) Init() {
	iq.queue = list.New()
	iq.send = make(chan interface{}, 1)
	iq.recv = make(chan interface{}, 1)
}

func (iq *Queue) Send(v interface{}) {
	iq.send <- v
}

func (iq *Queue) Receive() <-chan interface{} {
	return iq.recv
}

func (iq *Queue) Len() int {
	return iq.queue.Len()
}

func (iq *Queue) Queue() *list.List {
	return iq.queue
}

func (iq *Queue) Close() {
	close(iq.recv)
	close(iq.send)
}

func (iq *Queue) Loop() {
	for {
		front := iq.queue.Front()
		if front != nil {
			select {
			case iq.recv <- front.Value:
				iq.queue.Remove(front)
			case value, ok := <-iq.send:
				if ok {
					iq.queue.PushBack(value)
				} else {
					iq.send = nil
				}
			}
			continue
		}

		if iq.send == nil {
			close(iq.recv)
			return
		}
		value, ok := <-iq.send
		if !ok {
			return
		}
		iq.queue.PushBack(value)
	}
}
