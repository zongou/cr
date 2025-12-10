# SCritps2

| Program | Description |
| ------- | ----------- |
| cr      | Code Runner |

| key | value |
| --- | ----- |
| A   | Apple |

- [ ] Apple
- [x] Banana

## tidy

```sh
go mod tidy
```

## Build

```sh
go build -ldflags="-s -w" -trimpath
```

```sh
du -ahd0 cr
file cr
```

### Win

```sh
export GOOS=windows
export GOARCH=amd64

${MD_EXE} build
```

## Run

```sh
go run . "$@"
```
