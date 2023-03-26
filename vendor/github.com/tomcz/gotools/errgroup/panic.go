package errgroup

import (
	"context"
	"fmt"
	"runtime/debug"

	"golang.org/x/sync/errgroup"
)

// Group provides an interface compatible with golang.org/x/sync/errgroup
// for instances that enhance the capabilities of Groups.
type Group interface {
	Go(f func() error)
	Wait() error
}

// PanicHandler processes the recovered panic.
type PanicHandler func(p any) error

// Opt is a configuration option.
type Opt func(g *panicGroup)

type panicGroup struct {
	group  *errgroup.Group
	handle PanicHandler
}

// WithPanicHandler overrides the default panic handler.
func WithPanicHandler(ph PanicHandler) Opt {
	return func(p *panicGroup) {
		p.handle = ph
	}
}

// New creates a panic-handling Group,
// without any context cancellation.
func New(opts ...Opt) Group {
	group := &errgroup.Group{}
	pg := &panicGroup{group: group}
	pg.configure(opts)
	return pg
}

// NewContext creates a panic-handling Group.
// The returned context is cancelled on first error,
// first panic, or when the Wait function exits.
func NewContext(ctx context.Context, opts ...Opt) (Group, context.Context) {
	group, ctx := errgroup.WithContext(ctx)
	pg := &panicGroup{group: group}
	pg.configure(opts)
	return pg, ctx
}

func (p *panicGroup) configure(opts []Opt) {
	for _, opt := range opts {
		opt(p)
	}
	if p.handle == nil {
		p.handle = defaultPanicHandler
	}
}

func (p *panicGroup) Wait() error {
	return p.group.Wait()
}

func (p *panicGroup) Go(f func() error) {
	p.group.Go(func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = p.handle(r)
			}
		}()
		return f()
	})
}

func defaultPanicHandler(p any) error {
	stack := string(debug.Stack())
	if ex, ok := p.(error); ok {
		return fmt.Errorf("panic: %w\nstack: %s", ex, stack)
	}
	return fmt.Errorf("panic: %+v\nstack: %s", p, stack)
}
