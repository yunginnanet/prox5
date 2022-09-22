package logger

type Logger interface {
	Print(str string)
	Printf(format string, a ...interface{})
}
