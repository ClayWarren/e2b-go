# e2b-go

Unofficial Go SDK for [E2B](https://e2b.dev/) - Cloud runtime for AI agents.
Inspired by the [official E2B Code Interpreter SDKs](https://github.com/e2b-dev/code-interpreter).

E2B is an open-source infrastructure that allows you to run AI-generated code in secure isolated sandboxes in the cloud.

## Installation

```bash
go get github.com/ClayWarren/e2b-go
```

## Quick Start

### 1. Get your E2B API key
1. Sign up to E2B [here](https://e2b.dev/).
2. Get your API key [here](https://e2b.dev/docs/getting-started/api-key).

### 2. Execute code with Code Interpreter
The `CodeInterpreter` provides a stateful Jupyter-like environment.

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ClayWarren/e2b-go"
)

func main() {
	ctx := context.Background()

	// Create a new code interpreter sandbox
	// By default it uses the "code-interpreter-v1" template
	sbx, err := e2b.NewCodeInterpreter(ctx, "your-api-key")
	if err != nil {
		log.Fatal(err)
	}
	defer sbx.Close(ctx)

	// Execute code
	_, err = sbx.RunCode(ctx, "x = 1")
	if err != nil {
		log.Fatal(err)
	}

	// State is preserved between calls
	execution, err := sbx.RunCode(ctx, "x += 1; x")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(execution.Text()) // Outputs: 2
}
```

## Features

- **Code Interpreter**: Stateful execution of Python/JS code with rich output support (charts, images).
- **Sandbox Lifecycle**: Create, keep alive, reconnect, and stop sandboxes.
- **Filesystem Operations**: Read, write, list, mkdir, and watch for changes.
- **Process Execution**: Start processes with environment variables and working directory.
- **Event Streaming**: Subscribe to stdout, stderr, and exit events.

## Parity with JS/Python SDK

This SDK is designed to be a 1:1 Go implementation of the E2B V2 specialized SDKs. It supports the same JSON-RPC methods and abstractions found in `@e2b/code-interpreter`.

## Documentation

- [E2B Documentation](https://e2b.dev/docs)

## E2B Cookbook
Visit the [E2B Cookbook](https://github.com/e2b-dev/cookbook) to get inspired by examples with different LLMs and AI frameworks.

## License

MIT