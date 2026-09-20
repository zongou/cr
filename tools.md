# Tools

## A

```sh
echo "A"
```

## B

```sh
echo "B"
```

## test

```sh
_cr(){
    cr -f ${CR_FILE} "$@"
}

_cr a
_cr b

alias cr="cr -f ${CR_FILE}"
cr a
cr b
```
