package clientcontroller

import (
	"fmt"
	"sync"

	"github.com/ds-test-framework/scheduler/types"
)

type Counter struct {
	counter int
	mtx     *sync.Mutex
}

func NewCounter() *Counter {
	return &Counter{
		counter: 0,
		mtx:     new(sync.Mutex),
	}
}

func (id *Counter) Next() int {
	id.mtx.Lock()
	defer id.mtx.Unlock()

	cur := id.counter
	id.counter = id.counter + 1

	return cur
}

func (id *Counter) Reset() {
	id.mtx.Lock()
	defer id.mtx.Unlock()

	id.counter = 0
}

func (id *Counter) NextID(from, to types.ReplicaID) string {
	return fmt.Sprintf("%s_%s_%d", from, to, id.Next())
}
