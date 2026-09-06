# Tasks

## Features

Demonstrate features

```sh
cr env
cr arguments foo bar
echo Hello | cr pipe
cr exitStatus || echo "Recieved exit status $?"
echo ---- Feature: Custom executors
export TMPDIR=${TMPDIR-/tmp}
export MD_C="sh,-c,printf '%s' '{CODE}'>${TMPDIR}/a.c && cc ${TMPDIR}/a.c -o $TMPDIR/a && $TMPDIR/a"
cr c_hello
```

### Env

Print built-in env

```sh
echo ---- Feautre: Built-in env
echo CR=${CR}
echo CR_FILE=${CR_FILE}
```

### Arguments

Example to pass arguments

```sh
echo ---- Feature: Positional arguments
echo "Recieved arguments: $*"
```

### ExitStatus

Example with exit status

```sh
echo ---- Feature: ExitStatus
exit_code=$(shuf -i 1-255 -n 1)
echo "Script exit with code ${exit_code}"
exit ${exit_code}
```

### Pipe

Example to read stdin

```sh
echo ---- Feature: Pipe
echo "Recieved stdin: $(cat)"
```

### C_hello

C program example will be used later

```c
#include <stdio.h>

int main() {
    printf("Hello, 世界！ I am C.\n");
    return 0;
}
```

## Development

### Run

Zig run program

```sh
target=$(uname -m)-linux-musl
zig run -target ${target} -lc main.c -- "$@"
```

### Build

Build program

```sh
cc -o cr main.c "$@"
```

### Build:release

Build for release

```sh
cc -o cr main.c -static -s "$@"
```

### Zig-Build

Zig build program

```sh
target=$(uname -m)-linux-musl
zig cc -target ${target} -o cr main.c "$@"
```

### Zig-Build:release

Zig build for release

```sh
target=$(uname -m)-linux-musl
zig cc -target ${target} -o cr main.c -static -s "$@"
```

### Install

Install program

```sh
program=cr
if command -v sudo >/dev/null; then
    sudo install "${program}" "/usr/local/bin/${program}"
    if test -d /etc/bash_completion.d/; then
        sudo install completions/completion.sh /etc/bash_completion.d/${program}
    fi
elif test "${PREFIX+1}"; then
    install "${program}" "${PREFIX}/bin/${program}"
    if test -d "${PREFIX}/etc/bash_completion.d/"; then
        install completions/completion.sh "${PREFIX}/etc/bash_completion.d/${program}"
    fi
fi
```

### Test

Some tests

```sh
cr
cr -f c/main.c || true
cr -f test/test.md || true
cr -f LICENSE || true
```

#### Stat

Print status of program

```sh
du -ahd0 cr
file cr
llvm-objdump -p cr | grep LOAD
```

#### Benchmark

Some benchmarks

```sh
hyperfine "${CR} env" "$@"
```
