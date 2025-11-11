# CR

Codeblock Runner for Markdown

Example: To run code blocks under heading [Echo](#echo) with arguments `Hello World!`.

```sh
cr echo Hello World!
```

`echo` indicates the heading, and `Hello World!` will be passed to codeblocks as arguments.

Or you can call with heading path start from h2 and with separator `/`

```sh
cr examples/echo Hello World!
```

Run without arguments will give you a hint of available headings.

## Supported languages

- shellscript
- awk
- javascript
- python
- ruby
- php
- batch
- powershell

To handle unsupported languages, for example a c hello world program, see [c_hello](#c_hello)

## How does this work?

1. Find makrdown document.
   1. Find markdown doc by name in the order of `scripts.md`, `.scripts.md`, `README.md`.
   2. Find in the current directory and upper directories.

2. Parsing markdown doc into nodes.
3. Find the first matched node start from h2 nodes.
4. Execute code blocks in the found node.

## Env

> Prefixed env

| key     | description           |
| ------- | --------------------- |
| MD_EXE  | path to `<program>`   |
| MD_FILE | path to markdown file |

> Custom env

You can defind env by creating a table with header name `key` and `value`:

| key     | value | description     |
| ------- | ----- | --------------- |
| heading | Env   | Current heading |

You can also define boolean env map by creating a task list:

- [x] item_1
- [ ] item_2

Print env

```sh
for env in \
    MD_EXE \
    MD_FILE \
    heading \
    item_1 \
    item_2; do

    eval echo "${env}=\${${env}}"
done
```

### ChildEnv

Test scoped env

| key     | value    |
| ------- | -------- |
| heading | ChildEnv |

```sh
echo "heading=${heading}"
```

## Build

Build this program

```sh
go build . "$@"
```

### win

| key    | value   |
| ------ | ------- |
| GOOS   | windows |
| GOARCH | amd64   |

```sh
${MD_EXE} build "$@"
```

### Run

Build and run

```sh
go run . "$@"
```

### Install

Build and install

```sh
${MD_EXE} --file=${MD_FILE} build "$@"
if command -v sudo >/dev/null; then
    sudo install "${program}" "/usr/local/bin/${program}"
elif test "${PREFIX+1}"; then
    install "${program}" "${PREFIX}/bin/${program}"
fi
```

## Test

```sh
${MD_EXE} test/parsing
${MD_EXE} test/commands
```

### Parsing

Test parsing files

```sh
./${MD_EXE}
./${MD_EXE} -f main.c || true
./${MD_EXE} -f LICENSE || true
```

### Commands

Test commands

```sh
./${MD_EXE} echo Hello World!
./${MD_EXE} --file test/test.md -v
./${MD_EXE} --file test/test.md -v b1/c1/d1
```

### Stat

Build and print stat

```sh
du -ahd0 ./${MD_EXE}
file ./${MD_EXE}
llvm-objdump -p ./${MD_EXE} | grep LOAD
```

### Benchmark

Benchmark this program

```sh
hyperfine "${MD_EXE} env" "$@"
```

<!-- ```sh
export CC=/opt/android-ndk-r29/toolchains/llvm/prebuilt/linux-x86_64/bin/aarch64-linux-android35-clang
cr stat -static -s
# hyperfine "./${MD_EXE} -v env" --show-output
file cr
./${MD_EXE} benchmark
./${MD_EXE} -v echo Hello World!
``` -->

## Examples

Setting env

| key     | value |
| ------- | ----- |
| heading | Test  |

```sh
${MD_EXE} env
${MD_EXE} examples/env/childEnv
${MD_EXE} examples/arguments -- foo bar
echo Hello | ${MD_EXE} examples/stdin
${MD_EXE} examples/c_hello
```

### Echo

Print input

```sh
echo "$@"
```

### Arguments

Example to pass arguments

```sh
echo "shellscript with arguments: $*"
```

```js
console.log(`nodejs with arguments: ${process.argv}`);
```

```python
import sys

print("python with arguments: %s" %(sys.argv))
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
echo "stdin: $(cat)"
```

### c_hello

Example to handle C program

```sh
TMPDIR=${TMPDIR:-/tmp}
$MD_EXE -ac examples/c_hello_source >"${TMPDIR}/hello.c"
cc -o "${TMPDIR}/a" "${TMPDIR}/hello.c"
"${TMPDIR}/a"
```

### c_hello_source

C Hello World

```c
#include <stdio.h>
int main() {
    printf("Hello, World! C.\n");
    return 0;
}
```

---

Inspired by [mask](https://github.com/jacobdeichert/mask) and [xc](https://github.com/joerdav/xc).
