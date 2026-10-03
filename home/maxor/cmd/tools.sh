# ── Herramientas: logs, debug, completions, version ──────────────────
maxor_cmd ui tools "home store themes update doctor"
maxor_cmd setup tools ""
maxor_cmd logs tools "--last --path"
maxor_cmd debug tools ""
maxor_cmd completions tools "fish bash zsh"
maxor_cmd version tools "--json"

# Versión del contrato --json: sube solo con cambios incompatibles.
MAXOR_JSON_SCHEMA=1

# La pantalla completa es un programa aparte (maxor-tui). `maxor ui [pantalla]`
# la abre y `maxor setup` abre el asistente; sin terminal o sin el programa,
# se explica por qué y se sale con el código de «falta algo».
tui_run() { # tui_run pantalla
  [ -t 0 ] && [ -t 1 ] || die_code "$EX_NEEDS" @ui.no_tty
  command -v maxor-tui > /dev/null || die_code "$EX_NEEDS" @ui.no_tui
  exec maxor-tui --screen "$1"
}
cmd_ui() {
  case "${1:-home}" in
    home | store | themes | update | doctor | setup) tui_run "${1:-home}" ;;
    *) usage_error ui ;;
  esac
}
cmd_setup() { tui_run setup; }

cmd_version() {
  local pretty
  pretty="$(os_pretty)"
  case "${1:-}" in
    --json) jq -cn --arg v "$MAXOR_VERSION" --arg os "$pretty" --argjson s "$MAXOR_JSON_SCHEMA" '{version: $v, schema: $s, os: $os}' ;;
    "") t version.line "$MAXOR_VERSION" "$pretty"; echo ;;
    *) usage_error version ;;
  esac
}

cmd_logs() {
  case "${1:-}" in
    --path) printf '%s\n' "$logfile" ;;
    --last)
      if [ -s "$logdir/last-error.log" ]; then cat "$logdir/last-error.log"; else ui_say info @logs.no_last; fi
      ;;
    "")
      if [ -s "$logfile" ]; then tail -n 40 "$logfile"; else ui_say info @logs.none; fi
      ;;
    *) usage_error logs ;;
  esac
}

# Informe para adjuntar a un error: versión, equipo, diagnóstico y registro.
cmd_debug() {
  local out="$PWD/maxor-debug-$(date +%Y%m%d-%H%M%S).txt"
  {
    echo "== maxor $MAXOR_VERSION · $(date -Is)"
    grep -m1 '^PRETTY_NAME=' /etc/os-release || true
    uname -a
    nix --version 2> /dev/null || true
    if git -C "$flake_dir" rev-parse HEAD > /dev/null 2>&1; then
      echo "flake: $(git -C "$flake_dir" rev-parse --abbrev-ref HEAD) $(git -C "$flake_dir" rev-parse --short HEAD)"
    fi
    echo
    echo "== hardware"
    hw_detect
    echo
    echo "== doctor"
    NO_COLOR=1 "$0" doctor 2>&1 || true
    echo
    echo "== last error"
    cat "$logdir/last-error.log" 2> /dev/null || echo "(none)"
    echo
    echo "== log (last 100 lines)"
    tail -n 100 "$logfile" 2> /dev/null || echo "(empty)"
  } > "$out" 2>&1
  ui_say ok @debug.written "${out/#$HOME/~}"
  ui_say info @debug.review
}

# Autocompletado generado desde el registro: siempre al día con los comandos.
cmd_completions() {
  local sh="${1:-}" c sum
  case "$sh" in
    fish)
      echo "complete -c maxor -f"
      for c in "${CMD_ORDER[@]}"; do
        msg sum "@cmd.$c"
        printf 'complete -c maxor -n __fish_use_subcommand -a %s -d "%s"\n' "$c" "${sum//\"/}"
        if [ -n "${CMD_SUBS[$c]}" ]; then
          printf 'complete -c maxor -n "__fish_seen_subcommand_from %s" -a "%s"\n' "$c" "${CMD_SUBS[$c]}"
        fi
      done
      echo 'complete -c maxor -l no-color -d "plain output"'
      echo 'complete -c maxor -s q -l quiet -d "only errors"'
      echo 'complete -c maxor -s v -l verbose -d "log to stderr"'
      ;;
    bash)
      printf '_maxor() {\n  local cur="${COMP_WORDS[COMP_CWORD]}" cmd="${COMP_WORDS[1]}"\n  if [ "$COMP_CWORD" -eq 1 ]; then\n    COMPREPLY=($(compgen -W "%s" -- "$cur"))\n    return\n  fi\n  case "$cmd" in\n' "${CMD_ORDER[*]}"
      for c in "${CMD_ORDER[@]}"; do
        [ -n "${CMD_SUBS[$c]}" ] && printf '    %s) COMPREPLY=($(compgen -W "%s" -- "$cur")) ;;\n' "$c" "${CMD_SUBS[$c]}"
      done
      printf '  esac\n}\ncomplete -F _maxor maxor\n'
      ;;
    zsh)
      printf '#compdef maxor\n_maxor() {\n  local -a cmds\n  cmds=(\n'
      for c in "${CMD_ORDER[@]}"; do
        msg sum "@cmd.$c"
        printf "    '%s:%s'\n" "$c" "${sum//\'/}"
      done
      printf '  )\n  if (( CURRENT == 2 )); then\n    _describe command cmds\n    return\n  fi\n  case $words[2] in\n'
      for c in "${CMD_ORDER[@]}"; do
        [ -n "${CMD_SUBS[$c]}" ] && printf '    %s) compadd %s ;;\n' "$c" "${CMD_SUBS[$c]}"
      done
      printf '  esac\n}\n_maxor "$@"\n'
      ;;
    *) usage_error completions ;;
  esac
}
