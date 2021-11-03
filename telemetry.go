package clientcontroller

import (
	"time"

	"github.com/ds-test-framework/scheduler/types"
)

var (
	MessageSendEventType    = "MessageSend"
	MessageReceiveEventType = "MessageReceive"
	TimeoutStartEventType   = "TimeoutStart"
	TimeoutEndEventType     = "TimeoutEnd"
)

type event struct {
	Type      string            `json:"type"`
	Replica   types.ReplicaID   `json:"replica"`
	Params    map[string]string `json:"params"`
	Timestamp int64             `json:"timestamp"`
}

func (c *ClientController) PublishEvent(t string, params map[string]string) {
	c.sendMasterMessage(&masterRequest{
		Type: "Event",
		Event: &event{
			Type:      t,
			Replica:   types.ReplicaID(c.replicaID),
			Params:    params,
			Timestamp: time.Now().Unix(),
		},
	})
}
func (c *ClientController) PublishEventAsync(t string, params map[string]string) {
	go c.PublishEvent(t, params)
}
