package clientcontroller

import (
	"time"

	"github.com/ds-test-framework/scheduler/types"
)

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
	Replica   types.ReplicaID        `json:"replica"`
	Message   string                 `json:"message"`
	Params    map[string]interface{} `json:"params"`
	Timestamp int64                  `json:"timestamp"`
}

func (c *ReplicaClient) Log(params map[string]interface{}, message string) {
	c.sendMasterMessage(&masterRequest{
		Type: "Log",
		Log: &log{
			Replica:   c.config.ReplicaID,
			Params:    params,
			Message:   message,
			Timestamp: time.Now().UTC().Unix(),
		},
	})
}

func (c *ReplicaClient) LogAsync(params map[string]interface{}, message string) {
	go c.Log(params, message)
}
