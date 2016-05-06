package di

import "log"

// Logger is the interface used to log errors
// that occurred while an object is built or closed.
type Logger interface {
	Error(args ...interface{})
}

// DefaultLogger is a Logger that uses log.Println
// to write the error on the standard output.
type DefaultLogger struct{}

func (l DefaultLogger) Error(args ...interface{}) {
	log.Println(args)
}

// MuteLogger is a Logger that doesn't log anything.
type MuteLogger struct{}

func (l MuteLogger) Error(args ...interface{}) {}
