package e2b

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
)

type (
	// ProcessEvents is a process event type.
	ProcessEvents string

	// Process is a process in the sandbox.
	Process struct {
		id  string            // ID is process id.
		cmd string            // cmd is process's command.
		Cwd string            // cwd is process's current working directory.
		sb  *Sandbox          // sb is the sandbox the process belongs to.
		Env map[string]string // env is process's environment variables.
	}

	// ProcessOption is an option for the process.
	ProcessOption func(*Process)
)

const (
	// OnStdout is the event for the stdout.
	OnStdout ProcessEvents = "onStdout"
	// OnStderr is the event for the stderr.
	OnStderr ProcessEvents = "onStderr"
	// OnExit is the event for the exit.
	OnExit ProcessEvents = "onExit"

	processSubscribe   Method = "process_subscribe"
	processUnsubscribe Method = "process_unsubscribe"
	processStart       Method = "process_start"

	charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// NewProcess creates a new process startable in the sandbox.
func (s *Sandbox) NewProcess(
	cmd string,
	opts ...ProcessOption,
) (*Process, error) {
	b := make([]byte, 12)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	proc := &Process{
		id:  string(b),
		sb:  s,
		cmd: cmd,
	}
	for _, opt := range opts {
		opt(proc)
	}
	if proc.Cwd == "" {
		proc.Cwd = s.Cwd
	}
	return proc, nil
}

// Start starts a process in the sandbox.
func (p *Process) Start(ctx context.Context) (err error) {
	if p.Env == nil {
		p.Env = map[string]string{"PYTHONUNBUFFERED": "1"}
	}
	respCh := make(chan []byte)
	err = p.sb.writeRequest(
		ctx,
		processStart,
		[]any{p.id, p.cmd, p.Env, p.Cwd},
		respCh,
	)
	if err != nil {
		return err
	}
	select {
	case body := <-respCh:
		res, err := decodeResponse[string, APIError](body)
		if err != nil {
			return err
		}
		if res.Error.Code != 0 {
			return fmt.Errorf("process start failed(%d): %s", res.Error.Code, res.Error.Message)
		}
		if res.Result == "" || len(res.Result) == 0 {
			return fmt.Errorf("process start failed got empty result id")
		}
		if p.id != res.Result {
			return fmt.Errorf("process start failed got wrong result id; want %s, got %s", p.id, res.Result)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Done returns a channel that is closed when the process is done.
func (p *Process) Done() <-chan struct{} {
	rCh, ok := p.sb.Map.Load(p.id)
	if !ok {
		return nil
	}
	return rCh.(<-chan struct{})
}

// SubscribeStdout subscribes to the process's stdout.
func (p *Process) SubscribeStdout(ctx context.Context) (chan Event, chan error) {
	return p.subscribe(ctx, OnStdout)
}

// SubscribeStderr subscribes to the process's stderr.
func (p *Process) SubscribeStderr(ctx context.Context) (chan Event, chan error) {
	return p.subscribe(ctx, OnStderr)
}

// SubscribeExit subscribes to the process's exit.
func (p *Process) SubscribeExit(ctx context.Context) (chan Event, chan error) {
	return p.subscribe(ctx, OnExit)
}

// Subscribe subscribes to a process event.
//
// It creates a go routine to read the process events into the provided channel.
func (p *Process) subscribe(
	ctx context.Context,
	event ProcessEvents,
) (chan Event, chan error) {
	events := make(chan Event)
	errs := make(chan error)
	go func(errCh chan error) {
		respCh := make(chan []byte)
		defer close(respCh)
		err := p.sb.writeRequest(ctx, processSubscribe, []any{event, p.id}, respCh)
		if err != nil {
			errCh <- err
		}
		res, err := decodeResponse[string, any](<-respCh)
		if err != nil {
			errCh <- err
		}
		p.sb.Map.Store(res.Result, respCh)
	loop:
		for {
			select {
			case eventBd := <-respCh:
				var event Event
				_ = json.Unmarshal(eventBd, &event)
				if event.Error != "" {
					p.sb.logger.Error("failed to read event", "error", event.Error)
					continue
				}
				events <- event
			case <-ctx.Done():
				break loop
			case <-p.Done():
				break loop
			}
		}

		p.sb.Map.Delete(res.Result)
		finishCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		p.sb.logger.Debug("unsubscribing from process", "event", event, "id", res.Result)
		_ = p.sb.writeRequest(finishCtx, processUnsubscribe, []any{res.Result}, respCh)
		unsubRes, _ := decodeResponse[bool, string](<-respCh)
		if unsubRes.Error != "" || !unsubRes.Result {
			p.sb.logger.Debug("failed to unsubscribe from process", "error", unsubRes.Error)
		}
	}(errs)
	return events, errs
}
