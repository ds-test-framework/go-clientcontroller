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
			State: stateS,
		},
	})
}

func (c *ClientController) UpdateStateAsync(s State) {
	go c.UpdateState(s)
}

type Log struct {
	Params map[string]interface{} `json:"params"`
}

func (c *ClientController) Log(l *Log) {
	c.sendMasterMessage(&masterRequest{
		Type: "Log",
		Log:  l,
	})
}

func (c *ClientController) LogAsync(l *Log) {
	go c.Log(l)
}
