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
${MD_EXE} -f test/hellos.md $@

```

## Env

Prefixed env

```sh
# Path to program
echo MD_EXE=${MD_EXE}
# Path to markdown file
echo MD_EXE=${MD_FILE}
```

## Build

Build this program

```sh
go build . "$@"
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
${MD_EXE} --file=${MD_FILE} release
if command -v sudo >/dev/null; then
    sudo install "${program}" "/usr/local/bin/${program}"
elif test "${PREFIX+1}"; then
    install "${program}" "${PREFIX}/bin/${program}"
fi
```

## Test

Run some tests

```sh
${MD_EXE}
${MD_EXE} -f c/main.c || true
${MD_EXE} -f test/test.md || true
${MD_EXE} -f LICENSE || true
```

### Stat

Build and print stat

```sh
du -ahd0 ${MD_EXE}
file ${MD_EXE}
llvm-objdump -p ${MD_EXE} | grep LOAD
```

### Benchmark

Benchmark this program

```sh
hyperfine "${MD_EXE} env" "$@"
```

## Examples

Run example code blocks

```sh
${MD_EXE} env
${MD_EXE} args -- foo bar
echo Hello | ${MD_EXE} stdin
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
