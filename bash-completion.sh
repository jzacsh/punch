#!/usr/bin/env bash

__punchClientCompletion() {
  [ "${COMP_WORDS[COMP_CWORD-1]}" = -c ] || return 0 # only autocomplete clients

  local clients; clients="$(punch query list)"
  COMPREPLY=( $(compgen -W "${clients[@]}" -- "${COMP_WORDS[$COMP_CWORD]}") )
}

complete -F __punchClientCompletion punch
