package astibob

import (
	"context"
	"fmt"
	"sync"

	"github.com/asticode/go-astikit"
)

// Statuses
const (
	RunningStatus = "running"
	StoppedStatus = "stopped"
)

type Runnable interface {
	Metadata() Metadata
	OnMessage(m *Message) error
	SetDispatchFunc(f DispatchFunc)
	SetRootCtx(ctx context.Context)
	SetTaskFunc(f astikit.TaskFunc)
	Start(ctx context.Context) error
	Status() string
	Stop()
}

type DispatchFunc func(m *Message)

type BaseRunnableOptions struct {
	Logger    astikit.StdLogger
	Metadata  Metadata
	OnMessage func(m *Message) error
	OnStart   func(ctx context.Context) error
}

type BaseRunnable struct {
	dispatchFunc DispatchFunc
	l            astikit.SeverityLogger
	o            BaseRunnableOptions
	oStart       *sync.Once
	oStop        *sync.Once
	rootCtx      context.Context
	startCancel  context.CancelFunc
	startCtx     context.Context
	status       string
	taskFunc     astikit.TaskFunc
}

func NewBaseRunnable(o BaseRunnableOptions) *BaseRunnable {
	return &BaseRunnable{
		l:      astikit.AdaptStdLogger(o.Logger),
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

func (r *BaseRunnable) NewTask() *astikit.Task { return r.taskFunc() }

func (r *BaseRunnable) OnMessage(m *Message) (err error) {
	// We need to send a done message
	if m.ID > 0 {
		defer func() {
			// Create message
			m, err := NewRunnableDoneMessage(&Identifier{
				Name: astikit.StrPtr(m.From.WorkerName()),
				Type: WorkerIdentifierType,
			}, RunnableDone{
				ID:      m.ID,
				Success: err == nil,
			})
			if err != nil {
				r.l.Error(fmt.Errorf("astibob: creating runnable done message failed: %w", err))
				return
			}

			// Dispatch
			r.Dispatch(m)
		}()
	}

	// Custom
	if r.o.OnMessage != nil {
		if err = r.o.OnMessage(m); err != nil {
			err = fmt.Errorf("astibob: custom message handling failed: %w", err)
			return
		}
	}
	return
}

func (r *BaseRunnable) RootCtx() context.Context { return r.rootCtx }

func (r *BaseRunnable) SetDispatchFunc(f DispatchFunc) { r.dispatchFunc = f }

func (r *BaseRunnable) SetRootCtx(ctx context.Context) { r.rootCtx = ctx }

func (r *BaseRunnable) SetTaskFunc(f astikit.TaskFunc) { r.taskFunc = f }

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
				err = fmt.Errorf("astibob: OnStart failed: %w", err)
			}
		} else {
			<-r.startCtx.Done()
		}

		// Check context
		if r.startCtx.Err() != nil {
			err = r.startCtx.Err()
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
}
