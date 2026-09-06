# CR

A CLI markdown defind task runner.

- **Documentation first, commands second.**

- **Does not aim to work like a script language.**

- **Does not want to break our writting style.**

## Usage

### Supported file names

this program looks for files with the following names, in order of priority:

- taskfile.md
- .taskfile.md

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
export MD_RUST="sh,-c,printf '%s' '{CODE}'>/tmp/a.rs && rustc /tmp/a.rs -o /tmp/a && /tmp/a"
export MD_RS="${MD_RUST}"
export MD_ZIG="sh,-c,printf '%s' '{CODE}'>/tmp/a.zig && zig run -lc /tmp/a.zig"
```

### Hide Codeblocks

- A codeblock which's language doesn't have an executor is hidden.
- A codeblock which's language contains any uppercased character is always hidden.
