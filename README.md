# CR

MarkDown CodeBlock runner.

## Echo

A simple example.

```sh
echo "$@"
```

## Quick Start

To execute code blocks under the heading [Echo](#echo) with arguments `Hello, 世界！`.

```shell
cr echo Hello, 世界！
```

## Features

### Built-in supported codeblock types

- sh
- bash
- zsh
- fish
- dash
- ksh
- ash
- awk
- js
- javascript
- py
- python
- rb
- ruby
- php
- cmd
- batch
- ps2
- powershell

### Handle any codeblock

Set env `MD_ALIA=foo,bar`

For example:

```shell
export MD_PYTHON="python3,-c,{CODE}"
export MD_PY="${MD_PYTHON}"
export MD_C="sh,-c,printf '%s' '{CODE}'>/tmp/a.c && cc /tmp/a.c -o /tmp/a && /tmp/a"
export MD_CPP="sh,-c,printf '%s' '{CODE}'>/tmp/a.cpp && c++ /tmp/a.cpp -o /tmp/a && /tmp/a"
export MD_CXX="${MD_CPP}"
export MD_C++="${MD_CPP}"
export MD_RUST="sh,-c,printf '%s' '{CODE}'>/tmp/a.rs && rustc /tmp/a.rs -o /tmp/a && /tmp/a"
export MD_RS="${MD_RUST}"
export MD_ZIG="sh,-c,printf '%s' '{CODE}'>/tmp/a.zig && zig run -lc /tmp/a.zig"
```

### Env

Print built-in env.

```sh
echo CR=${CR}
echo CR_FILE=${CR_FILE}
```

### Arguments

Example to pass arguments.

```sh
echo "Recieved arguments: $*"
```

### ExitStatus

Example with exit status.

```sh
exit_code=$(shuf -i 1-255 -n 1)
echo "Script exit with code ${exit_code}"
exit ${exit_code}
```

### Pipe

Example to read stdin.

```sh
echo "Recieved stdin: $(cat)"
```

### C_Hello

C program example will be used later.

```c
#include <stdio.h>

int main() {
    printf("Hello, 世界！ I am C.\n");
    return 0;
}
```

### Examples

Demonstrate these features.

```sh
${CR} env
${CR} arguments foo bar
echo Hello | ${CR} pipe
${CR} exitStatus || echo "Recieved exit status $?"
export TMPDIR=${TMPDIR-/tmp}
export MD_C="sh,-c,printf '%s' '{CODE}'>${TMPDIR}/a.c && cc ${TMPDIR}/a.c -o $TMPDIR/a && $TMPDIR/a"
${CR} c_hello
```

## Dev

### Run

#### Run:go

Run Go version.

```sh
go run . "$@"
```

#### Run:c

Run C version.

```sh
zig run -lc c/main.c -- "$@"
```

### Build

Choose and build

```sh
${CR} $(${CR} -1 build|gum choose)
```

#### Build:go

Build Go version for debug.

```sh
go build "$@" .
```

#### Build:c

Build C version for debug.

```sh
target=$(uname -m)
zig cc -target ${target}-linux-musl -o cr c/main.c "$@"
```

#### Build:Release:go

Build Go version for Release.

```sh
go build -ldflags="-w -s" . "$@"
```

#### Build:Release:c

Build C version for Release.

```sh
target=$(uname -m)
zig cc -target ${target}-linux-musl -o cr c/main.c -static -s "$@"
```

### Install

Install what is built.

```sh
program=cr
if command -v sudo >/dev/null; then
    sudo install "${program}" "/usr/local/bin/${program}"
elif test "${PREFIX+1}"; then
    install "${program}" "${PREFIX}/bin/${program}"
fi
```

### Test

Some tests.

```sh
${CR}
${CR} -f c/main.c || true
${CR} -f test/test.md || true
${CR} -f LICENSE || true
```

#### Stat

Print status of program.

```sh
du -ahd0 ${CR}
file ${CR}
llvm-objdump -p ${CR} | grep LOAD
```

#### Benchmark

Some benchmarks.

```sh
hyperfine "${CR} env" "$@"
```

---

Inspired by [mask](https://github.com/jacobdeichert/mask) and [xc](https://github.com/joerdav/xc).
