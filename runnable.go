package astibob

import (
	"context"
	"sync"

	astiworker "github.com/asticode/go-astitools/worker"
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
	SetRootCtx(ctx context.Context)
	SetTaskFunc(f astiworker.TaskFunc)
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
	dispatchFunc DispatchFunc
	o            BaseRunnableOptions
	oStart       *sync.Once
	oStop        *sync.Once
	rootCtx      context.Context
	startCancel  context.CancelFunc
	startCtx     context.Context
	status       string
	taskFunc     astiworker.TaskFunc
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

func (r *BaseRunnable) NewTask() *astiworker.Task { return r.taskFunc() }

func (r *BaseRunnable) OnMessage(m *Message) (err error) { return }

func (r *BaseRunnable) RootCtx() context.Context { return r.rootCtx }

func (r *BaseRunnable) SetDispatchFunc(f DispatchFunc) { r.dispatchFunc = f }

func (r *BaseRunnable) SetRootCtx(ctx context.Context) { r.rootCtx = ctx }

func (r *BaseRunnable) SetTaskFunc(f astiworker.TaskFunc) { r.taskFunc = f }

func (r *BaseRunnable) Status() string { return r.status }

func (r *BaseRunnable) Start(ctx context.Context) (err error) {
	// Make sure it's started only once
	r.oStart.Do(func() {
		// Create context
		r.startCtx, r.startCancel = context.WithCancel(ctx)

		// Reset once
		r.oStop = &sync.Once{}

		// Update status
		r.status = RunningStatus

		// Start
		if r.o.OnStart != nil {
			if err = r.o.OnStart(r.startCtx); err != nil {
				err = errors.Wrap(err, "astibob: OnStart failed")
			}
		} else {
			<-r.startCtx.Done()
		}

		// Check context
		if r.startCtx.Err() != nil {
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
		if r.startCancel != nil {
			r.startCancel()
		}

		// Reset once
		r.oStart = &sync.Once{}
	})
	return
}
