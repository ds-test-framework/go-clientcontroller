package godriver

import (
	"sync"
	"time"
)

type TimerInfo interface {
	Key() string
	Duration() time.Duration
}

type timer struct {
	timeouts map[string]TimerInfo
	outChan  chan TimerInfo
	lock     *sync.Mutex
}

func newTimer() *timer {
	return &timer{
		timeouts: make(map[string]TimerInfo),
		outChan:  make(chan TimerInfo, 10),
		lock:     new(sync.Mutex),
	}
}

func (t *timer) AddTimeout(info TimerInfo) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.timeouts[info.Key()] = info
}

func (t *timer) FireTimeout(key string) {
	t.lock.Lock()
	info, ok := t.timeouts[key]
	t.lock.Unlock()
	if ok {
		t.lock.Lock()
		delete(t.timeouts, key)
		t.lock.Unlock()
		go func(i TimerInfo) { t.outChan <- i }(info)
	}
}

type timeout struct {
	Type     string `json:"type"`
	Duration int    `json:"duration"`
	Peer     PeerID `json:"peer"`
}

func (c *ClientController) StartTimer(i TimerInfo) {
	c.timer.AddTimeout(i)

	tMsg := &timeout{
		Type:     i.Key(),
		Duration: int(i.Duration().Milliseconds()),
		Peer:     c.peerID,
	}
	c.sendMasterMessage(&masterRequest{
		Type:    "TimeoutMessage",
		Timeout: tMsg,
	})
}

func (c *ClientController) TimeoutChan() chan TimerInfo {
	return c.timer.outChan
}
