package godriver

const (
	// Directive action to start the replica
	startAction = "START"

	// Directive action to stop the replica
	stopAction = "STOP"

	// Directive action to restart the replica
	restartAction = "RESTART"

	// Directive action querying if the replica is ready
	isReadyAction = "ISREADY"
)

// directiveMessage contains the action that is to be performed by the replica
type directiveMessage struct {
	Action string `json:"action"`
}

// DirectiveHandler is used to perform action on the current replica such as start, stop or restart the replica.
// This is implementation specific and hence the interface to encapsulate the actions
type DirectiveHandler interface {
	Stop() error
	Start() error
	Restart() error
}
