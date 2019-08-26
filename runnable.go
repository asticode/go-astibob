package astibob

import (
	"context"
	"sync"

	"github.com/pkg/errors"
)

// Statuses
const (
	RunningStatus = "running"
	StoppedStatus = "stopped"
)

// Errors
var (
	ErrContextCancelled = errors.New("astibob: context cancelled")
)

type Runnable interface {
	Metadata() Metadata
	OnMessage(m *Message) error
	Start(ctx context.Context) error
	Status() string
	Stop()
}

type RunnableOptions struct {
	Metadata  Metadata
	OnMessage func(m *Message) error
	OnStart   func(ctx context.Context) error
}

type runnable struct {
	cancel context.CancelFunc
	ctx    context.Context
	o      RunnableOptions
	oStart *sync.Once
	oStop  *sync.Once
	status string
}

func NewRunnable(o RunnableOptions) Runnable {
	return &runnable{
		o:      o,
		oStart: &sync.Once{},
		oStop:  &sync.Once{},
		status: StoppedStatus,
	}
}

func (r *runnable) Metadata() Metadata {
	return r.o.Metadata
}

func (r *runnable) Status() string {
	return r.status
}

func (r *runnable) Start(ctx context.Context) (err error) {
	// Make sure it's started only once
	r.oStart.Do(func() {
		// Create context
		r.ctx, r.cancel = context.WithCancel(ctx)

		// Reset once
		r.oStop = &sync.Once{}

		// Update status
		r.status = RunningStatus

		// Start
		if r.o.OnStart != nil {
			if err = r.o.OnStart(r.ctx); err != nil {
				err = errors.Wrap(err, "astibob: OnStart failed")
			}
		} else {
			<-r.ctx.Done()
		}

		// Check context
		if r.ctx.Err() != nil {
			err = ErrContextCancelled
		}

		// Update status
		r.status = StoppedStatus
	})
	return
}

func (r *runnable) Stop() {
	// Make sure it's stopped only once
	r.oStop.Do(func() {
		// Cancel context
		if r.cancel != nil {
			r.cancel()
		}

		// Reset once
		r.oStart = &sync.Once{}
	})
	return
}

func (r *runnable) OnMessage(m *Message) (err error) {
	// No handler
	if r.o.OnMessage == nil {
		return
	}

	// Check status
	if r.status != RunningStatus {
		return
	}

	// Custom
	if err = r.o.OnMessage(m); err != nil {
		err = errors.Wrap(err, "astibob: OnMessage failed")
		return
	}
	return
}
