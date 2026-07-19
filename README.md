# CR

MarkDown CodeBlock runner.

## Quick Start

To execute code blocks under a heading named [Echo](#echo) with arguments `Hello, 世界！`.

```shell
cr echo Hello, 世界！
```

`echo` indicates the heading, and `Hello World!` will be passed as arguments.

Run without arguments will give you a hint of all available headings.

To print codeblocks under the heading [echo]

```shell
cr -c echo
```

## Code Block Support

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

To handle codeblock type not in the predescribed list or to override,  
you can export env in the format of `MD_LANG=foo,bar`

For examples:

```shell
export MD_PYTHON="python3,-c,{CODE}"
export MD_C="sh,-c,printf '%s' '{CODE}'>/tmp/a.c && cc /tmp/a.c -o /tmp/a && /tmp/a"
export MD_CPP="sh,-c,printf '%s' '{CODE}'>/tmp/a.cpp && c++ /tmp/a.cpp -o /tmp/a && /tmp/a"
export MD_RUST="sh,-c,printf '%s' '{CODE}'>/tmp/a.rs && rustc /tmp/a.rs -o /tmp/a && /tmp/a"
export MD_ZIG="sh,-c,printf '%s' '{CODE}'>/tmp/a.zig && zig run -lc /tmp/a.zig"
${CR_EXE} -f test/hellos.md $@

```

## Env

Prefixed env

```sh
# Path to program
echo CR_EXE=${CR_EXE}
# Path to markdown file
echo CR_FILE=${CR_FILE}
```

## Run

```sh
go run . "$@"
```

## Run_C

```sh
zig run -lc c/main.c -- "$@"
```

## Build

Build this program

```sh
go build . "$@"
```

## Build_C

```sh
target=$(uname -m)
zig cc -target ${target}-linux-musl -o cr c/main.c "$@"
```

### Release

Build staticlly and stripped

```sh
go build -ldflags="-w -s" . "$@"
```

### Install

Build and install

```sh
program=cr
${CR_EXE} --file=${CR_FILE} release
if command -v sudo >/dev/null; then
    sudo install "${program}" "/usr/local/bin/${program}"
elif test "${PREFIX+1}"; then
    install "${program}" "${PREFIX}/bin/${program}"
fi
```

## Test

Run some tests

```sh
${CR_EXE}
${CR_EXE} -f c/main.c || true
${CR_EXE} -f test/test.md || true
${CR_EXE} -f LICENSE || true
```

### Stat

Build and print stat

```sh
du -ahd0 ${CR_EXE}
file ${CR_EXE}
llvm-objdump -p ${CR_EXE} | grep LOAD
```

### Benchmark

Benchmark this program

```sh
hyperfine "${CR_EXE} env" "$@"
```

## Examples

Run example code blocks

```sh
${CR_EXE} env
${CR_EXE} args -- foo bar
echo Hello | ${CR_EXE} stdin
```

### Echo

Print input

```sh
echo "$@"
```

### Args

Example to pass arguments

```sh
echo "Shellscript with arguments: $*"
```

### Error

Example with exit code

```sh
exit_code=$(shuf -i 1-255 -n 1)
echo "Script exits with code ${exit_code}"
exit ${exit_code}
```

### stdin

Example to read stdin

```sh
echo "Catched stdin: $(cat)"
```

---

Inspired by [mask](https://github.com/jacobdeichert/mask) and [xc](https://github.com/joerdav/xc).
