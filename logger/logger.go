package logger

type Logger interface {
	// Print implementations at this time are actually println
	Print(str string)
	Printf(format string, a ...interface{})
	Errorf(format string, a ...interface{})
}
