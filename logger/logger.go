package logger

type Logger interface {
	Debug(message string, vars ...interface{})
	Info(message string, vars ...interface{})
	Error(message string, vars ...interface{})
}
