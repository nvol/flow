package flow

import "fmt"

type Logger interface {
	Info(msg string)
}

type defaultLogger struct{}

func (l *defaultLogger) Info(msg string) {
	fmt.Println("flow:", msg)
}
