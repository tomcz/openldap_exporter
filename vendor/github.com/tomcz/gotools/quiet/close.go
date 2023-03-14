package quiet

import (
	"context"
	"io"
	"time"
)

// Logger allows logging of errors and panics.
type Logger interface {
	Error(err error)
	Panic(p any)
}

var log Logger = noopLogger{}

// SetLogger sets a panic & error logger for the package
// rather than the default noop logger. Passing in a nil
// logger will reset the package logger to default.
func SetLogger(logger Logger) {
	if logger == nil {
		log = noopLogger{}
	} else {
		log = logger
	}
}

// Close quietly invokes the closer.
// Any errors or panics will be logged by the package logger.
func Close(closer io.Closer) {
	defer func() {
		if p := recover(); p != nil {
			log.Panic(p)
		}
	}()
	if err := closer.Close(); err != nil {
		log.Error(err)
	}
}

// CloseFunc quietly invokes the given function.
// Any panics will be logged by the package logger.
func CloseFunc(close func()) {
	Close(&quietCloser{close})
}

// CloseFuncE quietly invokes the given function.
// Any errors or panics will be logged by the package logger.
func CloseFuncE(close func() error) {
	Close(&quietCloserE{close})
}

// CloseWithTimeout provides a closer for graceful service shutdown.
// Any errors or panics will be logged by the package logger.
func CloseWithTimeout(close func(ctx context.Context) error, timeout time.Duration) {
	Close(&timeoutCloser{close: close, timeout: timeout})
}

type quietCloserE struct {
	close func() error
}

func (c *quietCloserE) Close() error {
	return c.close()
}

type quietCloser struct {
	close func()
}

func (c *quietCloser) Close() error {
	c.close()
	return nil
}

type timeoutCloser struct {
	close   func(context.Context) error
	timeout time.Duration
}

func (c *timeoutCloser) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	return c.close(ctx)
}

type noopLogger struct{}

func (n noopLogger) Error(error) {}

func (n noopLogger) Panic(any) {}
