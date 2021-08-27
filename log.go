package clientcontroller

import "time"

var (
	msgKey = "_msg"
)

// Logger interface for logging information
type Logger interface {
	Debug(msg string, keyvals ...interface{})
	Info(msg string, keyvals ...interface{})
	Error(msg string, keyvals ...interface{})
}

type silentLogger struct {
}

func newDefaultLogger() Logger {
	return &silentLogger{}
}

func (d *silentLogger) Info(msg string, keyvals ...interface{}) {
}
func (d *silentLogger) Debug(msg string, keyvals ...interface{}) {
}
func (d *silentLogger) Error(msg string, keyvals ...interface{}) {
}

type log struct {
	Replica   ReplicaID              `json:"replica"`
	Message   string                 `json:"message"`
	Params    map[string]interface{} `json:"params"`
	Timestamp int64                  `json:"timestamp"`
}

func (c *ClientController) Log(params map[string]interface{}, message string) {
	c.sendMasterMessage(&masterRequest{
		Type: "Log",
		Log: &log{
			Replica:   c.replicaID,
			Params:    params,
			Message:   message,
			Timestamp: time.Now().UTC().Unix(),
		},
	})
}

func (c *ClientController) LogAsync(params map[string]interface{}, message string) {
	go c.Log(params, message)
}
