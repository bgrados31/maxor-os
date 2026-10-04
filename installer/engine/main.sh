# ── Orquestador: etapas, reanudación y subcomandos ───────────────────
#   maxor-install run      --answers FILE [--resume|--reset] [--events FILE] [--secret-fd N]
#   maxor-install plan     --answers FILE        lo que haría, sin tocar nada (simulacro)
#   maxor-install validate --answers FILE        solo comprueba las respuestas
#   maxor-install hash     --answers FILE        la huella del plan (para instalar junto a otro sistema)
#   maxor-install probe                          los discos del equipo y su espacio libre, en JSON
IN_STAGES=(preflight disk luks filesystem host install bootloader finish)

in_usage() {
  cat << '__END_USAGE__'
Usage: maxor-install <run|plan|validate|hash|probe> [--answers FILE] [options]

  run        install Maxor OS with the answers in FILE
  plan       print what `run` would do, without touching anything (dry run)
  validate   check the answers only
  hash       print the plan hash (an install alongside another system must carry it)
  probe      list the disks of this machine and their free space, as JSON

Options:
  --answers FILE    the answers (JSON, see installer/schema/answers.v1.json)
  --resume          continue a previous run: stages that finished are skipped
  --reset           forget a previous run's progress (it does not undo the disk)
  --events FILE     write progress events (JSON lines) to FILE
  --secret-fd N     read the encryption passphrase from file descriptor N
  --only A,B        run only these stages, in their normal order (for tests and repairs)
__END_USAGE__
}

# stage_run NOMBRE → corre una etapa en un subproceso con errexit, para que el primer fallo la pare.
stage_run() {
  local s="$1" fn="stage_$1" sentinel="$IN_STATE/stages/$1" rc=0 i total=${#IN_STAGES[@]} idx=0
  for i in "${!IN_STAGES[@]}"; do [ "${IN_STAGES[$i]}" = "$s" ] && idx=$i; done
  if [ "$IN_DRY" != 1 ] && [ -e "$sentinel" ]; then
    in_emit "$s" skip "already done" "$(awk -v i="$((idx + 1))" -v n="$total" 'BEGIN{printf "%.2f", i/n}')"
    return 0
  fi
  IN_STAGE="$s"
  in_emit "$s" start "" "$(awk -v i="$idx" -v n="$total" 'BEGIN{printf "%.2f", i/n}')"
  set +e
  (
    set -e
    "$fn"
  )
  rc=$?
  set -e
  if [ "$rc" != 0 ]; then
    in_emit "$s" fail "stage failed (exit $rc); the log is $IN_LOG"
    case "$rc" in
      2 | 3 | 4 | 130) exit "$rc" ;;
      *) exit "$IN_EX_FAIL" ;;
    esac
  fi
  if [ "$IN_DRY" != 1 ]; then
    mkdir -p "$IN_STATE/stages"
    : > "$sentinel"
  fi
  in_emit "$s" ok "" "$(awk -v i="$((idx + 1))" -v n="$total" 'BEGIN{printf "%.2f", i/n}')"
}

engine_run() {
  local answers="$1" resume="$2" reset="$3" secret_fd="$4" only="${5:-}" s
  ans_load "$answers" || in_die "$IN_EX_ANSWERS" "the answers are not valid (see above); nothing was changed"

  if [ -n "$secret_fd" ]; then IN_SECRET="$(cat <&"$secret_fd")"; fi
  if ans_true '.disk.encrypt.enabled' && [ -z "$IN_SECRET" ] && [ "$IN_DRY" != 1 ]; then
    in_die "$IN_EX_ANSWERS" "encryption is on but no passphrase was given (--secret-fd)"
  fi

  # Instalar junto a otro sistema exige haber visto ESTE plan: la huella que la pantalla enseñó.
  if [ "$(ans .disk.strategy)" = alongside ] && [ "$(ans .disk.plan_hash)" != "$(ans_plan_hash)" ]; then
    in_die "$IN_EX_ANSWERS" "disk.plan_hash does not match these answers: the plan changed after it was shown; nothing was changed"
  fi

  if [ "$IN_DRY" != 1 ]; then
    if [ "$reset" = 1 ]; then rm -rf "$IN_STATE/stages" "$IN_STATE/devices.json"; fi
    if [ -d "$IN_STATE/stages" ] && [ "$resume" != 1 ] && [ -n "$(ls -A "$IN_STATE/stages" 2> /dev/null)" ]; then
      in_die "$IN_EX_USAGE" "a previous run left progress in $IN_STATE: use --resume to continue it or --reset to forget it"
    fi
    mkdir -p "$IN_STATE"
    [ -n "$IN_EVENTS" ] || IN_EVENTS="$IN_STATE/events.jsonl"
  else
    printf 'DRYRUN: plan %s\n' "$(ans_plan_hash)"
    printf 'DRYRUN: install Maxor OS on %s (%s, %s%s) for user %s\n' "$(ans .disk.device)" "$(ans .disk.strategy)" "$(ans .disk.filesystem)" \
      "$(ans_true '.disk.encrypt.enabled' && printf ', encrypted' || true)" "$(ans .user.name)"
  fi

  if [ -n "$only" ]; then
    local keep=() want w
    IFS=, read -ra want <<< "$only"
    for s in "${IN_STAGES[@]}"; do
      for w in "${want[@]}"; do [ "$s" = "$w" ] && keep+=("$s"); done
    done
    [ "${#keep[@]}" = "${#want[@]}" ] || in_die "$IN_EX_USAGE" "--only names a stage that does not exist (stages: ${IN_STAGES[*]})"
    IN_STAGES=("${keep[@]}")
  fi
  for s in "${IN_STAGES[@]}"; do stage_run "$s"; done
  in_emit "done" ok "Maxor OS is installed" 1.00
}

main() {
  local cmd="${1:-}" answers="" resume=0 reset=0 secret_fd="" only=""
  [ "$#" -gt 0 ] && shift
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --answers) answers="${2:-}"; shift ;;
      --answers=*) answers="${1#--answers=}" ;;
      --resume) resume=1 ;;
      --reset) reset=1 ;;
      --events) IN_EVENTS="${2:-}"; shift ;;
      --events=*) IN_EVENTS="${1#--events=}" ;;
      --secret-fd) secret_fd="${2:-}"; shift ;;
      --only) only="${2:-}"; shift ;;
      --only=*) only="${1#--only=}" ;;
      --dry-run) IN_DRY=1 ;;
      -h | --help) in_usage; return 0 ;;
      *) in_usage >&2; in_die "$IN_EX_USAGE" "unknown option: $1" ;;
    esac
    shift
  done
  case "$cmd" in
    probe) disk_probe_json ;;
    validate)
      [ -n "$answers" ] || { in_usage >&2; in_die "$IN_EX_USAGE" "validate needs --answers FILE"; }
      ans_load "$answers" || in_die "$IN_EX_ANSWERS" "the answers are not valid"
      printf 'The answers are valid.\n'
      ;;
    hash)
      [ -n "$answers" ] || { in_usage >&2; in_die "$IN_EX_USAGE" "hash needs --answers FILE"; }
      ans_load "$answers" structure || in_die "$IN_EX_ANSWERS" "the answers are not valid"
      ans_plan_hash
      ;;
    plan | run)
      [ -n "$answers" ] || { in_usage >&2; in_die "$IN_EX_USAGE" "$cmd needs --answers FILE"; }
      [ "$cmd" = plan ] && IN_DRY=1
      engine_run "$answers" "$resume" "$reset" "$secret_fd" "$only"
      ;;
    -h | --help | help | "") in_usage ;;
    *) in_usage >&2; in_die "$IN_EX_USAGE" "unknown command: $cmd" ;;
  esac
}

if [ "${MAXOR_INSTALL_NO_MAIN:-0}" != 1 ]; then main "$@"; fi
