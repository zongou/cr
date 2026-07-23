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
        printf "%s\n" "$*" >>"${TMPDIR-/tmp}/a.log"
    }

    _log_write "---------------------------------------------"

    COMPREPLY=()                             # Reset the COMPREPLY array
    local cur=${COMP_WORDS[COMP_CWORD]}      # Current word
    local prev=${COMP_WORDS[COMP_CWORD - 1]} # Previous word
    local builtinOpts="-h --help -f --file -c --code -1 -t --tree -l --log-file"
    local mdCmds=$(cr -1)
    local mdHeadings=$(cr -t | grep -Eo '(├──|└──).+  ' | cut -d ' ' -f2-)

    _log_write cur=$cur
    _log_write prev=$prev

    if test ${COMP_CWORD} -eq 1; then
        _log_write 0 args
        COMPREPLY=($(compgen -W "${builtinOpts} ${mdCmds}" -- ${cur}))
    else
        local i=1
        local fileOpt
        local logFileOpt
        while test $i -le ${COMP_CWORD}; do
            _log_write arg[$i]=${COMP_WORDS[i]}

            case "${COMP_WORDS[i]}" in
            -h | --help)
                _log_write is_help
                ;;
            -c | --code)
                _log_write is_code
                COMPREPLY=($(compgen -W "${mdCmds}" -- "${cur}"))
                ;;
            -t | --tree)
                _log_write is_tree
                COMPREPLY=($(compgen -W "${mdHeadings}" -- "${cur}"))
                ;;
            -1)
                _log_write is_one
                COMPREPLY=($(compgen -W "${mdCmds}" -- "${cur}"))
                ;;
            -f | --file)
                COMPREPLY=($(compgen -f -- "${cur}"))
                fileOpt=$(eval echo "${COMP_WORDS[$((i + 1))]}")
                _log_write fileOpt=${fileOpt}

                if test -f "${fileOpt}"; then
                    mdCmds=$(cr -1 -f "${fileOpt}")
                    mdHeadings=$(cr -t -f "${fileOpt}" | grep -Eo '(├──|└──).+  ' | cut -d ' ' -f2-)
                fi

                i=$((i + 1))
                ;;
            -l | --log-file)
                COMPREPLY=($(compgen -f -- "${cur}"))
                logFileOpt=$(eval echo "${COMP_WORDS[$((i + 1))]}")
                _log_write logFileOpt=${logFileOpt}

                i=$((i + 1))
                ;;
            :)
                local lastArg=$(echo ${COMP_LINE} | grep -o '[^ ]*$')
                _log_write :lastArg=$lastArg
                COMPREPLY=($(compgen -W "${mdCmds}" -- ${lastArg} | sed "s/${lastArg}//g"))
                _log_write :COMPREPLY=$COMPREPLY
                ;;
            '')
                _log_write response previous reply
                ;;

            *)
                local lastArg=$(echo ${COMP_LINE} | grep -o '[^ ]*$')
                case "$lastArg" in
                *:*)
                    _log_write \*lastArg=$lastArg
                    COMPREPLY=($(compgen -W "${mdCmds}" -- ${lastArg} | sed "s/${lastArg}/${cur}/g"))
                    _log_write \*COMPREPLY=$COMPREPLY
                    ;;
                *)
                    _log_write \*response previous reply
                    ;;
                esac
                ;;
            esac

            i=$((i + 1))
        done
    fi
}

# Register the completion function
complete -o default -F _cr -o nospace cr
