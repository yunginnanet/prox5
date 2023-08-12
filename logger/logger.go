package logger

type Logger interface {
	Printf(format string, a ...interface{})
	Errorf(format string, a ...interface{})
}
