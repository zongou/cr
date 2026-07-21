#!/bin/bash

# #BASH_COMPLETION
# - This variable defines the path to the bash completion script. If you install additional completion scripts, you can add their paths to this variable.

# #BASH_COMPLETION_VERSIONS
# - Used to specify the version of Bash completion scripts to load. This can be useful if you are maintaining scripts for different versions of bash.

# #COMP_CWORD
# - This variable contains the index of the current word being completed in the command line. It is useful for determining context when defining custom completions.

# #COMP_LINE
# - The entire current command line. This can be helpful for your completion functions if you want to analyze the full command string.

# #COMP_POINT
# - The position of the cursor in the command line, which is an index into COMPREPLY
# - This is an array used by completion functions to provide possible completions. You will populate this array with the suggestions you want to offer when the user triggers completion.

# #COMP_WORDS
# - An array containing the individual words in the current command line. This can be used to decide how to complete based on prior words.

### Enabling Bash Completion

# To enable Bash completion in your shell, make sure to include the following lines in your $(.bashrc) or $(.bash_profile):

_cr() {
    _log_write() {
        printf "%s\n" "$*" >>/tmp/a.log
    }

    _log_write "---------------------------------------------"

    COMPREPLY=()                             # Reset the COMPREPLY array
    local cur=${COMP_WORDS[COMP_CWORD]}      # Current word
    local prev=${COMP_WORDS[COMP_CWORD - 1]} # Previous word

    _log_write cur=$cur
    _log_write prev=$prev

    if test ${COMP_CWORD} -eq 1; then
        _log_write empty
        COMPREPLY=($(compgen -W "-h --help -f --file -c --code -1 -t --tree -l --log-file $(cr -1)" -- ${cur}))
    else
        local i=1
        local fileOpt=""
        local mdcmds
        local builtinOpts="-h --help -f --file -c --code -1 -t --tree -l --log-file"
        while test $i -le ${COMP_CWORD}; do
            _log_write arg[$i]=${COMP_WORDS[i]}

            if test -f "${fileOpt}"; then
                mdcmds=$(cr -1 -f ${fileOpt} 2>/dev/null)
            else
                mdcmds=$(cr -1)
            fi

            case "${COMP_WORDS[i]}" in
            -h | --help)
                _log_write is_help
                ;;
            -c | --code)
                _log_write is_code
                COMPREPLY=($(compgen -W $mdcmds -- "${cur}"))
                ;;
            -t | --tree)
                _log_write is_tree
                COMPREPLY=($(compgen -W $mdcmds -- "${cur}"))
                ;;
            -1)
                _log_write is_one
                COMPREPLY=($(compgen -W $mdcmds -- "${cur}"))
                ;;
            -f | --file)
                COMPREPLY=($(compgen -f -- "${cur}"))
                fileOpt=${COMP_WORDS[$((i + 1))]}
                i=$((i + 1))
                _log_write file_opt=${fileOpt}
                ;;
            -l | --log-file)
                _log_write logfile_opt
                COMPREPLY=($(compgen -f -- "${cur}"))
                ;;
            *)
                _log_write "Unknown command '${COMP_WORDS[i]}'"
                COMPREPLY=($(compgen -W "$builtinOpts $mdcmds" -- ${cur}))
                ;;
            esac

            i=$((i + 1))
        done
    fi
}

# Register the completion function
complete -o default -F _cr -o nospace cr

# for arg in "${COMP_WORDS[@]}"; do
#     _log_write "arg[%d]=$arg" $
# done

# local cur
# COMPREPLY=()
# cur="${COMP_WORDS[COMP_CWORD]}"

# # Define options f#t#mdmmand
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

# if [[ "$subcommand" == "-t" ]]; then
#     COMPREPLY=($(compgen -W "$(cr -1)" -- "${cur}"))
# elif [[ "$subcommand" == "stop" ]]; then
#     COMPREPLY=($(compgen -W "service1 service2" -- "${COMP_WORDS[COMP_CWORD]}"))
# else
#     COMPREPLY=($(compgen -W "start stop restart" -- "${COMP_WORDS[COMP_CWORD]}"))
# fi

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

# for i in "${!COMP_WORDS[@]}"; do
#     _log_write "arg[$i]=${COMP_WORDS[$i]}"
# done
