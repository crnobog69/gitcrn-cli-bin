# REM

Project notes for rem.

## Quick Start

1. Run:
   - rem list
   - rem build
2. Inspect graph:
   - rem graph
3. Run a specific target:
   - rem run test

## Remfile Syntax (TOML)

```toml
default = "build"

[vars]
APP_NAME = "myapp"
VERSION = "${VERSION:-dev}"

[task.build]
desc = "Build binary"
inputs = ["cmd/myapp/main.go", "go.mod"]
outputs = ["bin/${APP_NAME}"]
cmds = [
  "mkdir -p bin",
  "go build -ldflags \"-X main.version=${VERSION}\" -o bin/${APP_NAME} ./cmd/myapp",
]

[task.test]
deps = ["build"]
cmds = ["go test ./..."]
```

## Commands

- rem init
- rem doctor
- rem list -D VERSION=v0.1.0
- rem graph -D APP_NAME=rem
- rem build [target]
- rem run <target>
- rem run -D VERSION=v1.0.0 production
- rem run -D RELEASE_VERSION=v1.0.0 release
- rem format
- rem format --check

## Notes

- Variable expansion: ${VAR} and ${VAR:-fallback}
- Tasks without outputs behave like phony targets
- rem format writes canonical TOML and may rewrite layout/comments
- rem doctor checks basic environment and Remfile health
- Task shell follows $SHELL; set REM_SHELL=/path/to/shell to force shell
