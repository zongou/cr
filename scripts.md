# Tasks For My Project

## run

```sh
zig run -target x86_64-linux-musl -lc c/main.c -- "$@"
```

## build

```sh
zig cc -target x86_64-linux-musl -lc -o cr c/main.c
```
