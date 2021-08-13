package clientcontroller

import (
	"time"

	"github.com/ds-test-framework/scheduler/types"
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

func (c *ClientController) Log(params map[string]string, message string) {
	c.sendMasterMessage(&masterRequest{
		Type: "Log",
		Log: &types.ReplicaLog{
			Replica:   types.ReplicaID(c.replicaID),
			Params:    params,
			Message:   message,
			Timestamp: time.Now().UTC().Unix(),
		},
	})
}

func (c *ClientController) LogAsync(params map[string]string, message string) {
	go c.Log(params, message)
}
