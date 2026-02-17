# REM

Белешке за rem пројекат.

## Брзи старт

1. Покрени:
   - rem list
   - rem build
2. Погледај граф:
   - rem graph
3. Покрени конкретан target:
   - rem run test

## Remfile синтакса (TOML)

```toml
default = "build"

[vars]
APP_NAME = "myapp"
VERSION = "${VERSION:-dev}"

[task.build]
desc = "Компилација бинарног фајла"
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

## Команде

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

## Напомене

- Експанзија променљивих: ${VAR} и ${VAR:-fallback}
- Task без outputs се понаша као phony target
- rem format пише канонски TOML и може да промени распоред/коментаре
- rem doctor проверава основно окружење и здравље Remfile-а
- Task shell прати $SHELL; постави REM_SHELL=/path/to/shell за форсирање shell-а
