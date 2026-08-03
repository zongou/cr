# CR

CR is a CLI markdown codeblock runner.

## Concept

- **Documentation first, commands second.**

- **Does not aim to work like a script language.**

- **Does not want to break our writting style.**

## Usage

### Supported file names

this program looks for files with the following names, in order of priority:

- taskfile.md
- .taskfile.md
- README.md

### Create a Taskfile

Create a simple `taskfile.md` in your project.

````markdown
# Tasks

<!-- A heading defines the command's name -->

## Build

<!-- A blockparagraph defines the command's description -->

Builds my project

<!-- A code block defines the script to be executed -->

```sh
echo "building project..."
```

## Test

Tests my project

You can also write documentation anywhere you want. Only certain types of markdown patterns
are parsed to determine the command structure.

```js
console.log("running tests...");
```
````

### Run your commands

Try running one of your commands!

```SH
cr build
cr test
```

### Running from a subdirectory

If a Taskfile cannot be found in the current working directory, this program will walk up the file tree until it finds one (similar to how git works). When running from a subdirectory like this, it will behave as if you ran it from the directory containing the Taskfile.

### Language executor

Built-in supported codeblock languages list:

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

To set an executor for a codeblock in language `lang`, set env `MD_LANG=progarm,arg1,arg2...`  
You cannot set executor for a codeblock which's language contains any uppercased charactor.

Placeholders:

- {LANG}: codeblock lang
- {CODE}: codeblock code

placeholder will be replaced with it's corresponding value.

Some examples:

```SH
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

### Hide Codeblocks

- A codeblock which's language doesn't have an executor is hidden.
- A codeblock which's language contains any uppercased character is always hidden.

### Env

Print built-in env

```sh
echo CR=${CR}
echo CR_FILE=${CR_FILE}
```

### Arguments

Example to pass arguments

```sh
echo "Recieved arguments: $*"
```

### ExitStatus

Example with exit status

```sh
exit_code=$(shuf -i 1-255 -n 1)
echo "Script exit with code ${exit_code}"
exit ${exit_code}
```

### Pipe

Example to read stdin

```sh
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

### Examples

Demonstrate features

```sh
cr env
cr arguments foo bar
echo Hello | cr pipe
cr exitStatus || echo "Recieved exit status $?"
export TMPDIR=${TMPDIR-/tmp}
export MD_C="sh,-c,printf '%s' '{CODE}'>${TMPDIR}/a.c && cc ${TMPDIR}/a.c -o $TMPDIR/a && $TMPDIR/a"
cr c_hello
```

## Development

### Run

#### Run:go

Run Go version

```sh
cd go
go run . "$@"
```

#### Run:c

Run C version with zig

```sh
cd c
target=$(uname -m)-linux-musl
zig run -target ${target} -lc main.c -- "$@"
```

#### Run:rust

Run rust version

```sh
cd rust
cargo run -- "$@"
```

### Build

Choose one to build

```sh
opt=$(cr -1 build | gum choose --header="Choose one to build")
cr ${opt}
```

#### Build:go

Build Go version

```sh
cd go
go build -o ../cr "$@" .
```

#### Build:go:release

Build Go release version

```sh
cd go
go build -o ../cr -ldflags="-w -s" "$@" .
```

#### Build:c

Build C version

```sh
cd c
cc -o ../cr main.c "$@"
```

#### Build:c:release

Build C release version

```sh
cd c
cc -o ../cr main.c -static -s "$@"
```

#### Build:c_zig

Build C with zig

```sh
cd c
target=$(uname -m)-linux-musl
zig cc -target ${target} -o ../cr main.c "$@"
```

#### Build:c_zig:release

Build C release with zig

```sh
cd c
target=$(uname -m)-linux-musl
zig cc -target ${target} -o ../cr main.c -static -s "$@"
```

#### Build:rust

```sh
cd rust
cargo build "$@"
cp target/debug/cr ../cr
```

#### Build:rust:release

```sh
cd rust
cargo build --release "$@"
cp target/release/cr ../cr
```

### Install

Install what is built

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
