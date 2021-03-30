package godriver

var (
	StartAction   = "START"
	StopAction    = "STOP"
	RestartAction = "RESTART"
	IsReadyAction = "ISREADY"
)

type DirectiveMessage struct {
	Action string `json:"action"`
}

type DirectiveHandler interface {
	Stop() error
	Start() error
	Restart() error
}
