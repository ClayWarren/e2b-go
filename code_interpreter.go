package e2b

import (
	"context"
	"fmt"
)

// CodeInterpreter is a specialized sandbox for executing AI-generated code.
// It uses a Jupyter-like notebook environment.
type CodeInterpreter struct {
	*Sandbox
}

// Result represents an output from a cell execution.
type Result struct {
	IsMainResult bool              `json:"isMainResult"`
	Data         map[string]string `json:"data"`
	Formats      []string          `json:"formats"`
}

// Text returns the plain text representation of the result if available.
func (r *Result) Text() string {
	if r.Data == nil {
		return ""
	}
	return r.Data["text/plain"]
}

// ExecutionLogs contains the stdout and stderr from a cell execution.
type ExecutionLogs struct {
	Stdout []string `json:"stdout"`
	Stderr []string `json:"stderr"`
}

// ExecutionError contains the error information from a cell execution.
type ExecutionError struct {
	Name      string `json:"name"`
	Value     string `json:"value"`
	Traceback string `json:"traceback"`
}

// Execution represents the full result of a cell execution.
type Execution struct {
	Results []Result        `json:"results"`
	Logs    ExecutionLogs   `json:"logs"`
	Error   *ExecutionError `json:"error"`
}

// Text returns the text representation of the main result.
func (e *Execution) Text() string {
	for _, r := range e.Results {
		if r.IsMainResult {
			return r.Text()
		}
	}
	return ""
}

const (
	// DefaultCodeInterpreterTemplate is the default template for CodeInterpreter.
	DefaultCodeInterpreterTemplate SandboxTemplate = "code-interpreter-v1"
)

// NewCodeInterpreter creates a new CodeInterpreter sandbox.
// By default, it uses the "code-interpreter-v1" template.
func NewCodeInterpreter(ctx context.Context, apiKey string, opts ...Option) (*CodeInterpreter, error) {
	// Apply default template if none provided
	sOpts := append([]Option{WithTemplate(DefaultCodeInterpreterTemplate)}, opts...)

	sb, err := NewSandbox(ctx, apiKey, sOpts...)
	if err != nil {
		return nil, err
	}

	return &CodeInterpreter{sb}, nil
}

// ExecCell executes a code cell in the notebook environment.
func (ci *CodeInterpreter) ExecCell(ctx context.Context, code string) (*Execution, error) {
	respCh := make(chan []byte)

	// notebook_execCell params: [code, kernelID]
	// kernelID is optional and usually empty for the default kernel
	err := ci.writeRequest(ctx, notebookExecCell, []any{code}, respCh)
	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case body := <-respCh:
		res, err := decodeResponse[Execution, APIError](body)
		if err != nil {
			return nil, err
		}
		if res.Error.Code != 0 {
			return nil, fmt.Errorf("notebook execution failed (%d): %s", res.Error.Code, res.Error.Message)
		}
		return &res.Result, nil
	}
}

// RunCode is an alias for ExecCell, matching the TS SDK naming.
func (ci *CodeInterpreter) RunCode(ctx context.Context, code string) (*Execution, error) {
	return ci.ExecCell(ctx, code)
}
