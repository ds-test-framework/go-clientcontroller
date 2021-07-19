package clientcontroller

import "time"

type event struct {
	Type      string            `json:"type"`
	Replica   PeerID            `json:"replica"`
	Params    map[string]string `json:"params"`
	Timestamp int64             `json:"timestamp"`
}

func (c *ClientController) PublishEvent(t string, params map[string]string) {
	c.sendMasterMessage(&masterRequest{
		Type: "Event",
		Event: &event{
			Type:      t,
			Replica:   c.peerID,
			Params:    params,
			Timestamp: time.Now().Unix(),
		},
	})
}
func (c *ClientController) PublishEventAsync(t string, params map[string]string) {
	go c.PublishEvent(t, params)
}

type Log struct {
	ID     PeerID                 `json:"id"`
	Params map[string]interface{} `json:"params"`
}

func (c *ClientController) Log(params map[string]interface{}) {
	c.sendMasterMessage(&masterRequest{
		Type: "Log",
		Log: &Log{
			ID:     c.peerID,
			Params: params,
		},
	})
}

func (c *ClientController) LogAsync(params map[string]interface{}) {
	go c.Log(params)
}
