package flow

import "fmt"

type Logger interface {
	Info(msg string)
	Warn(msg string)
}

type defaultLogger struct{}

func (l *defaultLogger) Info(msg string) {
	fmt.Println("[INFO] flow:", msg)
}

func (l *defaultLogger) Warn(msg string) {
	fmt.Println("[WARN] flow:", msg)
}
