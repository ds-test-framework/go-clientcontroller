package clientcontroller

import "errors"

type State interface {
	Marshal() (string, error)
}

func (c *ClientController) UpdateState(s State) error {
	stateS, err := s.Marshal()
	if err != nil {
		return errors.New("failed to marshal state")
	}

	return c.sendMasterMessage(&masterRequest{
		Type: "StateUpdate",
		State: &state{
			ID:    c.peerID,
			State: stateS,
		},
	})
}

func (c *ClientController) UpdateStateAsync(s State) {
	go c.UpdateState(s)
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
