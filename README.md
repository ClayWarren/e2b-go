# e2b-go

Go SDK for [E2B](https://e2b.dev/) - Cloud runtime for AI agents.

## Installation

```bash
go get github.com/ClayWarren/e2b-go
```

## Usage

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

	// Create a new sandbox
	sb, err := e2b.NewSandbox(ctx, "your-api-key")
	if err != nil {
		log.Fatal(err)
	}
	defer sb.Stop(ctx)

	// List files
	files, err := sb.Ls(ctx, "/")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Files:", files)

	// Write a file
	err = sb.Write(ctx, "/tmp/hello.txt", []byte("Hello, World!"))
	if err != nil {
		log.Fatal(err)
	}

	// Read a file
	content, err := sb.Read(ctx, "/tmp/hello.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Content:", content)

	// Run a process
	proc, err := sb.NewProcess("echo 'Hello from E2B!'")
	if err != nil {
		log.Fatal(err)
	}

	err = proc.Start(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Subscribe to stdout
	stdout, errCh := proc.SubscribeStdout(ctx)
	select {
	case event := <-stdout:
		fmt.Println("Output:", event.Params.Result.Line)
	case err := <-errCh:
		log.Fatal(err)
	}
}
```

## Features

- **Sandbox Lifecycle**: Create, keep alive, reconnect, and stop sandboxes
- **Filesystem Operations**: Read, write, list, mkdir, and watch for changes
- **Process Execution**: Start processes with environment variables and working directory
- **Event Streaming**: Subscribe to stdout, stderr, and exit events

## License

MIT
