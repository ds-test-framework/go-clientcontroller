package godriver

import (
	"sync"
	"time"
)

// TimeoutInfo encapsulates the timeout information that needs to be scheduled
type TimeoutInfo interface {
	// Key returns a unique key for the given timeout. Only one timeout for a specific key can be running at any given time.
	Key() string
	// Duration of the timeout
	Duration() time.Duration
}

type timer struct {
	timeouts map[string]TimeoutInfo
	outChan  chan TimeoutInfo
	lock     *sync.Mutex
}

func newTimer() *timer {
	return &timer{
		timeouts: make(map[string]TimeoutInfo),
		outChan:  make(chan TimeoutInfo, 10),
		lock:     new(sync.Mutex),
	}
}

func (t *timer) AddTimeout(info TimeoutInfo) {
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
		go func(i TimeoutInfo) { t.outChan <- i }(info)
	}
}

type timeout struct {
	Type     string `json:"type"`
	Duration int    `json:"duration"`
	Peer     PeerID `json:"peer"`
}

// StartTimer schedules a timer for the given TimerInfo
// Note: The timers are implemented as message sends and receives that are to be scheduled by the
// testing strategy. If you do not want to instrument timers as message send/receives then do not use this function.
func (c *ClientController) StartTimer(i TimeoutInfo) {
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

// TimeoutChan returns the channel on which timeouts are delivered.
func (c *ClientController) TimeoutChan() chan TimeoutInfo {
	return c.timer.outChan
}
