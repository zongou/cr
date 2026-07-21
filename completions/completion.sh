#!/bin/bash

# function_exists() {
#     declare -F $1 >/dev/null
#     return $?
# }

# function_exists __ltrim_colon_completions ||
#     __ltrim_colon_completions() {
#         if [[ "$1" == *:* && "$COMP_WORDBREAKS" == *:* ]]; then
#             # Remove colon-word prefix from COMPREPLY items
#             local colon_word=${1%${1##*:}}
#             local i=${#COMPREPLY[*]}
#             while [[ $((--i)) -ge 0 ]]; do
#                 COMPREPLY[$i]=${COMPREPLY[$i]#"$colon_word"}
#             done
#         fi
#     }

# _comp_ltrim_colon_completions() {
#     ((${#COMPREPLY[@]})) || return 0
#     _comp_compgen -c "$1" ltrim_colon "${COMPREPLY[@]}"
# }

# # Work-around bash_completion issue where bash interprets a colon
# # as a separator, borrowed from maven completion code which borrowed
# # it from darcs completion code :)
# local colonprefixes=${cur%"${cur##*:}"}
# printf "colonprefixes=%s\n" $colonprefixes >> /tmp/a.log
# local i=${#COMPREPLY[*]}
# while ((i-- > 0)); do
#     COMPREPLY[i]=${COMPREPLY[i]#"$colonprefixes"}
#     printf "cur=%s COMPREPLY[%d]=%s\n" "$cur" "$i" "${COMPREPLY[i]}" >>/tmp/a.log
# done

# _mycmd_completions is the function to generate the completions
_cr() {
    # local cur
    # COMPREPLY=()
    # cur="${COMP_WORDS[COMP_CWORD]}"

    # # Define options for the `mycmd` command
    # local options=$(cr -1)

    # Generate completions based on the options defined
    # COMPREPLY=($(compgen -W "${options}" -- "${cur}"))

    # case "$cur" in
    # -t | --tree | -c | --code)
    #     COMPREPLY=($(compgen -W "${options}" -- ${cur}))
    #     # COMPREPLY=($(compgen -W "-- -h --help -c --code -1 -t --tree -f --file -l --log-file" -- ${cur}))
    #     ;;
    # *)
    #     COMPREPLY=()
    #     ;;
    # esac

    local arg_1="${COMP_WORDS[1]}"
    local cur="${COMP_WORDS[COMP_CWORD]}"
    COMPREPLY=()

    case $arg_1 in
    -c | --code | -1 | -t | --tree)
        COMPREPLY=($(compgen -W "$(cr -1)" -- "${cur}"))
        ;;
    -f | --file | -l | --log-file)
        COMPREPLY=($(compgen -c -o nospace -f -- "${cur}"))
        ;;
    esac

    # if [[ "$subcommand" == "-t" ]]; then
    #     COMPREPLY=($(compgen -W "$(cr -1)" -- "${cur}"))
    # elif [[ "$subcommand" == "stop" ]]; then
    #     COMPREPLY=($(compgen -W "service1 service2" -- "${COMP_WORDS[COMP_CWORD]}"))
    # else
    #     COMPREPLY=($(compgen -W "start stop restart" -- "${COMP_WORDS[COMP_CWORD]}"))
    # fi
}

# Register the completion function for `mycmd`
complete -F _cr cr
