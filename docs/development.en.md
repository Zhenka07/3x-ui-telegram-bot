# Development

[Russian version](development.md)

## Environment Requirements

Building and developing the project requires Go 1.22+ (Go 1.25 recommended), Git, and access to a 3x-ui panel instance for integration testing:

```bash
git clone https://github.com/Zhenka07/3x-ui-telegram-bot.git
cd 3x-ui-telegram-bot
```

## Testing and Checks

The project includes automated unit tests that run independently of external network services:

```bash
go test -v -count=1 ./...
```

Static analysis is performed using the standard `go vet` tool:

```bash
go vet ./...
```

## Building Binaries

Build a static standalone binary without CGO for the Linux amd64 target architecture:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o admin-bot ./cmd/admin
```

## Codebase Standards

- all functions and methods must have standard Go doc comments in English (`// FunctionName ...`);
- inline explanatory comments inside function bodies are strictly forbidden;
- database operations must use the pure-Go `modernc.org/sqlite` driver without CGO dependencies;
- Telegram messages are formatted using HTML with user-supplied inputs sanitized via `escapeHTML`.
