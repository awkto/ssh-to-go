# bash completion for stogo — subcommands, flags, and live session names.
# Installed by the sshtogo .deb; or load manually:
#   source <(stogo completion bash)

_stogo() {
    local cur prev
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    if [[ $COMP_CWORD -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "auth list ls connect c offload kill status completion version help" -- "$cur") )
        return
    fi

    case "${COMP_WORDS[1]}" in
        auth)
            [[ $COMP_CWORD -eq 2 ]] && COMPREPLY=( $(compgen -W "login logout" -- "$cur") )
            ;;
        list|ls)
            [[ "$prev" == "-o" ]] && COMPREPLY=( $(compgen -W "json" -- "$cur") ) \
                || COMPREPLY=( $(compgen -W "-t -a -o" -- "$cur") )
            ;;
        connect|c|offload|kill)
            if [[ $COMP_CWORD -eq 2 ]]; then
                # Session names come from the server; keep a tight timeout so a
                # dead server never hangs the shell mid-tab.
                local sessions
                if command -v timeout >/dev/null 2>&1; then
                    sessions=$(timeout 2 stogo __sessions 2>/dev/null)
                else
                    sessions=$(stogo __sessions 2>/dev/null)
                fi
                COMPREPLY=( $(compgen -W "$sessions" -- "$cur") )
            fi
            ;;
        completion)
            [[ $COMP_CWORD -eq 2 ]] && COMPREPLY=( $(compgen -W "bash" -- "$cur") )
            ;;
    esac
}

complete -F _stogo stogo
