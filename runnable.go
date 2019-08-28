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
	SetDispatchFunc(f DispatchFunc)
	Start(ctx context.Context) error
	Status() string
	Stop()
}

type DispatchFunc func(m *Message)

type BaseRunnableOptions struct {
	Metadata Metadata
	OnStart  func(ctx context.Context) error
}

type BaseRunnable struct {
	cancel       context.CancelFunc
	ctx          context.Context
	dispatchFunc DispatchFunc
	o            BaseRunnableOptions
	oStart       *sync.Once
	oStop        *sync.Once
	status       string
}

func NewBaseRunnable(o BaseRunnableOptions) *BaseRunnable {
	return &BaseRunnable{
		o:      o,
		oStart: &sync.Once{},
		oStop:  &sync.Once{},
		status: StoppedStatus,
	}
}

func (r *BaseRunnable) Dispatch(m *Message) {
	if r.dispatchFunc != nil {
		r.dispatchFunc(m)
	}
}

func (r *BaseRunnable) Metadata() Metadata { return r.o.Metadata }

func (r *BaseRunnable) OnMessage(m *Message) (err error) { return }

func (r *BaseRunnable) SetDispatchFunc(f DispatchFunc) { r.dispatchFunc = f }

func (r *BaseRunnable) Status() string { return r.status }

func (r *BaseRunnable) Start(ctx context.Context) (err error) {
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

func (r *BaseRunnable) Stop() {
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
