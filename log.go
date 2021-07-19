package clientcontroller

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
