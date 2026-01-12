package e2b

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type (
	// Event is a file system event.
	Event struct {
		Path      string      `json:"path"`      // Path is the path of the event.
		Name      string      `json:"name"`      // Name is the name of file or directory.
		Timestamp int64       `json:"timestamp"` // Timestamp is the timestamp of the event.
		Error     string      `json:"error"`     // Error is the possible error of the event.
		Params    EventParams `json:"params"`    // Params is the parameters of the event.
	}

	// EventParams is the params for subscribing to a process event.
	EventParams struct {
		Subscription string      `json:"subscription"` // Subscription is the subscription id of the event.
		Result       EventResult `json:"result"`       // Result is the result of the event.
	}

	// EventResult is a file system event response.
	EventResult struct {
		Type        string `json:"type"`
		Line        string `json:"line"`
		Timestamp   int64  `json:"timestamp"`
		IsDirectory bool   `json:"isDirectory"`
		Error       string `json:"error"`
	}

	// LsResult is a result of the list request.
	LsResult struct {
		Name  string `json:"name"`  // Name is the name of the file or directory.
		IsDir bool   `json:"isDir"` // isDir is true if the entry is a directory.
	}
)

const (
	filesystemWrite      Method = "filesystem_write"
	filesystemRead       Method = "filesystem_read"
	filesystemList       Method = "filesystem_list"
	filesystemRemove     Method = "filesystem_remove"
	filesystemMakeDir    Method = "filesystem_makeDir"
	filesystemReadBytes  Method = "filesystem_readBase64"
	filesystemWriteBytes Method = "filesystem_writeBase64"
	filesystemSubscribe  Method = "filesystem_subscribe"
)

// Mkdir makes a directory in the sandbox file system.
func (s *Sandbox) Mkdir(ctx context.Context, path string) error {
	respCh := make(chan []byte)
	err := s.writeRequest(ctx, filesystemMakeDir, []any{path}, respCh)
	if err != nil {
		return err
	}
	select {
	case body := <-respCh:
		resp, err := decodeResponse[string, APIError](body)
		if err != nil {
			return fmt.Errorf("failed to mkdir: %w", err)
		}
		if resp.Error.Code != 0 {
			return fmt.Errorf("failed to write to file: %s", resp.Error.Message)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Ls lists the files and/or directories in the sandbox file system at
// the given path.
func (s *Sandbox) Ls(ctx context.Context, path string) ([]LsResult, error) {
	respCh := make(chan []byte)
	defer close(respCh)
	err := s.writeRequest(ctx, filesystemList, []any{path}, respCh)
	if err != nil {
		return nil, err
	}
	select {
	case body := <-respCh:
		res, err := decodeResponse[[]LsResult, string](body)
		if err != nil {
			return nil, err
		}
		return res.Result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Read reads a file from the sandbox file system.
func (s *Sandbox) Read(
	ctx context.Context,
	path string,
) (string, error) {
	respCh := make(chan []byte)
	err := s.writeRequest(ctx, filesystemRead, []any{path}, respCh)
	if err != nil {
		return "", err
	}
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case body := <-respCh:
		res, err := decodeResponse[string, string](body)
		if err != nil {
			return "", err
		}
		if res.Error != "" {
			return "", fmt.Errorf("failed to read file: %s", res.Error)
		}
		return res.Result, nil
	}
}

// Write writes to a file to the sandbox file system.
func (s *Sandbox) Write(ctx context.Context, path string, data []byte) error {
	respCh := make(chan []byte)
	err := s.writeRequest(ctx, filesystemWrite, []any{path, string(data)}, respCh)
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case resp := <-respCh:
		err = json.Unmarshal(resp, &Request{})
		if err != nil {
			return err
		}
		return nil
	}
}

// WriteBytes writes bytes to a file in the sandbox file system.
func (s *Sandbox) WriteBytes(ctx context.Context, path string, data []byte) error {
	respCh := make(chan []byte)
	sEnc := base64.StdEncoding.EncodeToString(data)
	err := s.writeRequest(ctx, filesystemWriteBytes, []any{path, sEnc}, respCh)
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case resp := <-respCh:
		err = json.Unmarshal(resp, &Request{})
		if err != nil {
			return err
		}
		return nil
	}
}

// ReadBytes reads a file from the sandbox file system.
func (s *Sandbox) ReadBytes(ctx context.Context, path string) ([]byte, error) {
	resCh := make(chan []byte)
	defer close(resCh)
	err := s.writeRequest(ctx, filesystemReadBytes, []any{path}, resCh)
	if err != nil {
		return nil, err
	}
	select {
	case body := <-resCh:
		res, err := decodeResponse[string, string](body)
		if err != nil {
			return nil, err
		}
		sDec, err := base64.StdEncoding.DecodeString(res.Result)
		if err != nil {
			return nil, err
		}
		return sDec, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Watch watches a directory in the sandbox file system.
//
// This is intended to be run in a goroutine as it will block until the
// connection is closed, an error occurs, or the context is canceled.
//
// While blocking, filesystem events will be written to the provided channel.
func (s *Sandbox) Watch(
	ctx context.Context,
	path string,
	eCh chan<- Event,
) error {
	respCh := make(chan []byte)
	defer close(respCh)
	err := s.writeRequest(ctx, filesystemSubscribe, []any{"watchDir", path}, respCh)
	if err != nil {
		return err
	}
	res, err := decodeResponse[string, string](<-respCh)
	if err != nil {
		return err
	}
	s.Map.Store(res.Result, eCh)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var event Event
				err := json.Unmarshal(<-respCh, &event)
				if err != nil {
					return
				}
				if event.Error != "" {
					return
				}
				if event.Params.Subscription != path {
					continue
				}
				eCh <- event
			}
		}
	}()
	return nil
}
