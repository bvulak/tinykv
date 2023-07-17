package raft

import "sync"

type counter struct {
	sync.Mutex
	v  int
	ch chan bool
}

func NewCounter(v int) (*counter, <-chan bool) {
	var ch = make(chan bool)
	var ct = &counter{
		v:  v,
		ch: ch,
	}
	return ct, ch
}

func (ct *counter) Done() {
	ct.Lock()
	defer ct.Unlock()
	ct.v--
	if ct.v == 0 {
		ct.ch <- true
	}
}
