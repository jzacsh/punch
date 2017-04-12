#!/usr/bin/env bash

__punchClientCompletion() {
  {
    local firstArg; firstArg="${COMP_WORDS[COMP_CWORD-2]}"
    local secondArg; secondArg="${COMP_WORDS[COMP_CWORD-1]}"
    [[ "$firstArg" = punch ]] || [[ "$firstArg" = p ]] ||
    [[ "$firstArg" = bill ]] || {
      { [[ "$firstArg" = q ]] || [[ "$firstArg" = query ]]; } && {
        [[ "$secondArg" = report ]] ||
        [[ "$secondArg" = bills ]] ||
        [[ "$secondArg" = bill ]];
      };
    }
  } || return 0 # only autocomplete clients

  local clients; clients="$(punch query list)"
  COMPREPLY=( $(compgen -W "${clients[@]}" -- "${COMP_WORDS[$COMP_CWORD]}") )
}

complete -F __punchClientCompletion punch
