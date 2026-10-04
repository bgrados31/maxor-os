# ── Releases: aviso y aplicación de versiones firmadas ───────────────
# Cada release publica un manifiesto (manifest.json) firmado con la clave de
# release del proyecto (ssh-keygen -Y, espacio de nombres «maxor-release»).
# La CLI trae la clave pública (keys/allowed_signers) dentro del paquete, así
# que lo que no esté firmado por esa clave no se acepta, venga de donde venga.
#
#   maxor release check    mira la red, verifica y guarda el resultado
#   maxor release status   enseña lo guardado, sin red
#   maxor release apply    verifica la etiqueta, avanza el repositorio y actualiza
#
# El estado vive en ~/.local/state/maxor/release.json y lo leen la pantalla
# completa y el temporizador. Ver docs/UPDATES.md.
maxor_cmd release system "check status apply"

MAXOR_RELEASE_URL="${MAXOR_RELEASE_URL:-https://github.com/bgrados31/maxor-os/releases/latest/download}"
MAXOR_RELEASE_KEYS="${MAXOR_RELEASE_KEYS:-}"
REL_MIN_AGE=600 # segundos: dentro de este margen un «check» sin --force no vuelve a la red
reldir="$state/release"
relfile="$state/release.json"

# Versión válida: X.Y.Z con sufijo opcional (0.2.0-dev, 0.2.0-rc.1).
ver_valid() { [[ "$1" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]]; }

# ver_cmp A B → imprime -1, 0 o 1. Sin sufijo gana al que lo tiene (0.1.0 > 0.1.0-dev).
ver_cmp() {
  local a="$1" b="$2" ca cb pa="" pb="" i x y
  local -a xa xb
  ca="${a%%-*}"
  cb="${b%%-*}"
  [ "$ca" = "$a" ] || pa="${a#*-}"
  [ "$cb" = "$b" ] || pb="${b#*-}"
  IFS=. read -r -a xa <<< "$ca"
  IFS=. read -r -a xb <<< "$cb"
  for i in 0 1 2; do
    x=$((10#${xa[i]:-0}))
    y=$((10#${xb[i]:-0}))
    if [ "$x" -lt "$y" ]; then echo -1; return 0; fi
    if [ "$x" -gt "$y" ]; then echo 1; return 0; fi
  done
  if [ -z "$pa" ] && [ -n "$pb" ]; then echo 1
  elif [ -n "$pa" ] && [ -z "$pb" ]; then echo -1
  elif [[ "$pa" < "$pb" ]]; then echo -1
  elif [[ "$pa" > "$pb" ]]; then echo 1
  else echo 0; fi
}

# La versión instalada; una compilación sin VERSION («dev») cuenta como la más vieja.
rel_installed() { if ver_valid "$MAXOR_VERSION"; then printf '%s' "$MAXOR_VERSION"; else printf '0.0.0-dev'; fi; }

# ¿Hay al menos una clave de confianza? (las líneas con # no cuentan)
rel_has_key() { [ -n "$MAXOR_RELEASE_KEYS" ] && [ -r "$MAXOR_RELEASE_KEYS" ] && grep -qvE '^[[:space:]]*(#|$)' "$MAXOR_RELEASE_KEYS"; }

# rel_verify manifiesto firma → 0 si la firma es de la clave de release
rel_verify() {
  ssh-keygen -Y verify -f "$MAXOR_RELEASE_KEYS" -I maxor-release -n maxor-release -s "$2" < "$1" > /dev/null 2>&1
}

# Forma del manifiesto: producto, versión, etiqueta coherente, commit completo y secuencia.
rel_manifest_ok() { # rel_manifest_ok manifiesto
  jq -e '
    .schema == 1 and .product == "maxor-os"
    and (.version | type == "string" and test("^[0-9]+\\.[0-9]+\\.[0-9]+(-[0-9A-Za-z.]+)?$"))
    and .tag == ("v" + .version)
    and (.commit | type == "string" and test("^[0-9a-f]{40}$"))
    and (.sequence | type == "number" and . == floor and . > 0)
  ' "$1" > /dev/null 2>&1
}

# rel_write estado motivo [disponible] [manifiesto]
# Escribe release.json conservando lo que no cambia (último bueno, avisos ya dados).
rel_write() {
  local status="$1" reason="$2" avail="${3:-keep}" m="${4:-}" prev="{}" now tmp
  now="$(printf '%(%s)T' -1)"
  [ -s "$relfile" ] && prev="$(cat "$relfile" 2> /dev/null || echo '{}')"
  jq -e . <<< "$prev" > /dev/null 2>&1 || prev="{}"
  mkdir -p "$state" 2> /dev/null || return 0
  tmp="$(mktemp "$state/.release.XXXXXX")"
  jq -nc --arg status "$status" --arg reason "$reason" --arg installed "$(rel_installed)" \
    --arg avail "$avail" --argjson now "$now" --argjson prev "$prev" --slurpfile m "${m:-/dev/null}" '
    ($m[0] // {}) as $mm
    | {installed: $installed,
       status: $status,
       reason: $reason,
       checked_at: $now,
       ok_at: (if $status == "ok" then $now else ($prev.ok_at // 0) end),
       available: (if $avail == "true" then true elif $avail == "false" then false else ($prev.available // false) end),
       latest: ($mm.version // $prev.latest // ""),
       tag: ($mm.tag // $prev.tag // ""),
       commit: ($mm.commit // $prev.commit // ""),
       sequence: ($mm.sequence // $prev.sequence // 0),
       published: ($mm.published // $prev.published // ""),
       summary: ($mm.summary // $prev.summary // ""),
       url: ($mm.url // $prev.url // ""),
       notified: ($prev.notified // ""),
       alerted: ($prev.alerted // "")}
    | if $status == "insecure" then .available = false else . end' > "$tmp" \
    && mv -f "$tmp" "$relfile" || rm -f "$tmp"
  return 0
}

# Un intento de consulta dentro de la carpeta temporal $1. Siempre devuelve 0: el
# resultado queda en release.json (ok, unavailable o insecure).
rel_try() {
  local t="$1" code rc=0 m s highest seq ver cmp avail et=()
  if [ -s "$reldir/manifest.json" ] && [ -s "$reldir/manifest.json.sig" ] && [ -s "$reldir/etag" ]; then
    et=(--etag-compare "$reldir/etag")
  fi
  code="$(curl -sS -L --max-time 20 --connect-timeout 8 --retry 1 -A "maxor/$MAXOR_VERSION" "${et[@]}" \
    --etag-save "$t/etag" -o "$t/manifest.json" -w '%{http_code}' "$MAXOR_RELEASE_URL/manifest.json" 2> "$t/err")" || rc=$?
  if [ "$rc" != 0 ]; then
    log WARN "release: sin red ($(head -c 200 "$t/err" 2> /dev/null))"
    rel_write unavailable network
    return 0
  fi
  # Un archivo local (file://) no da código HTTP: si trajo cuerpo, cuenta como 200.
  if [ "$code" = 000 ] && [ -s "$t/manifest.json" ]; then code=200; fi
  case "$code" in
    304)
      m="$reldir/manifest.json"
      s="$reldir/manifest.json.sig"
      ;;
    200)
      m="$t/manifest.json"
      s="$t/manifest.json.sig"
      # -f: un 404 del servidor no se guarda como si fuera la firma (su página de error no está vacía)
      rc=0
      curl -sS -f -L --max-time 20 --connect-timeout 8 -A "maxor/$MAXOR_VERSION" -o "$s" "$MAXOR_RELEASE_URL/manifest.json.sig" 2> /dev/null || rc=$?
      case "$rc" in
        0 | 22 | 37) ;; # respondió (con la firma, con un error HTTP o sin el archivo): se juzga abajo
        *)
          # se cayó la red entre el manifiesto y su firma: no se pudo comprobar, no es una firma ausente
          rel_write unavailable network
          return 0
          ;;
      esac
      if [ "$rc" != 0 ] || [ ! -s "$s" ]; then
        log ERROR "release: el manifiesto no trae firma"
        rel_write insecure unsigned
        return 0
      fi
      ;;
    *)
      log WARN "release: HTTP $code"
      rel_write unavailable "http_$code"
      return 0
      ;;
  esac
  if ! rel_verify "$m" "$s"; then
    log ERROR "release: la firma del manifiesto no es válida"
    rel_write insecure bad_signature
    return 0
  fi
  if ! rel_manifest_ok "$m"; then
    log ERROR "release: manifiesto con forma inválida"
    rel_write insecure invalid_manifest
    return 0
  fi
  # Nunca se acepta una versión más vieja que la última vista (ataque de retroceso).
  seq="$(jq -r '.sequence' "$m")"
  highest=0
  [ -s "$reldir/sequence" ] && highest="$(cat "$reldir/sequence" 2> /dev/null || echo 0)"
  [[ "$highest" =~ ^[0-9]+$ ]] || highest=0
  if [ "$seq" -lt "$highest" ]; then
    log ERROR "release: secuencia $seq menor que la ya vista ($highest)"
    rel_write insecure rollback
    return 0
  fi
  ver="$(jq -r '.version' "$m")"
  if [ "$code" = 200 ]; then
    cp -f "$m" "$reldir/manifest.json"
    cp -f "$s" "$reldir/manifest.json.sig"
    [ -s "$t/etag" ] && cp -f "$t/etag" "$reldir/etag" || rm -f "$reldir/etag"
  fi
  printf '%s\n' "$seq" > "$reldir/sequence"
  cmp="$(ver_cmp "$ver" "$(rel_installed)")"
  avail=false
  [ "$cmp" = 1 ] && avail=true
  rel_write ok "" "$avail" "$reldir/manifest.json"
  return 0
}

# rel_refresh [force] → consulta la red (como mucho una vez cada REL_MIN_AGE s sin force)
rel_refresh() {
  local force="${1:-0}" now at st t
  now="$(printf '%(%s)T' -1)"
  mkdir -p "$reldir" 2> /dev/null || return 0
  exec 9> "$reldir/lock"
  flock -w 60 9 || true
  if [ "$force" != 1 ] && [ -s "$relfile" ]; then
    at="$(jq -r '.checked_at // 0' "$relfile" 2> /dev/null || echo 0)"
    st="$(jq -r '.status // ""' "$relfile" 2> /dev/null || true)"
    if [ "$st" = ok ] && [ "$((now - at))" -lt "$REL_MIN_AGE" ]; then return 0; fi
  fi
  if ! rel_has_key; then
    rel_write unavailable no_key
    return 0
  fi
  t="$(mktemp -d)"
  rel_try "$t"
  rm -rf "$t"
  return 0
}

# Aviso de escritorio, una sola vez por versión (y una vez por motivo si algo es inseguro).
rel_notify() {
  local st ver notified alerted reason title body tmp
  [ -s "$relfile" ] || return 0
  command -v notify-send > /dev/null 2>&1 || return 0
  st="$(jq -r '.status' "$relfile")"
  ver="$(jq -r '.latest' "$relfile")"
  notified="$(jq -r '.notified' "$relfile")"
  alerted="$(jq -r '.alerted' "$relfile")"
  reason="$(jq -r '.reason' "$relfile")"
  if [ "$st" = ok ] && [ "$(jq -r '.available' "$relfile")" = true ] && [ "$ver" != "$notified" ]; then
    msg title @release.notify_title
    msg body @release.notify_body "$ver" "$(rel_installed)"
    notify-send -a Maxor -i software-update-available "$title" "$body" || true
    tmp="$(mktemp "$state/.release.XXXXXX")"
    jq -c --arg v "$ver" '.notified = $v' "$relfile" > "$tmp" && mv -f "$tmp" "$relfile" || rm -f "$tmp"
  elif [ "$st" = insecure ] && [ "$reason" != "$alerted" ]; then
    msg title @release.alert_title
    msg body @release.alert_body
    notify-send -a Maxor -u critical -i dialog-warning "$title" "$body" || true
    tmp="$(mktemp "$state/.release.XXXXXX")"
    jq -c --arg r "$reason" '.alerted = $r' "$relfile" > "$tmp" && mv -f "$tmp" "$relfile" || rm -f "$tmp"
  fi
  return 0
}

# Estado guardado más «stale» (si el último chequeo bueno tiene más de 3 días).
rel_status_json() {
  local now
  now="$(printf '%(%s)T' -1)"
  if [ -s "$relfile" ]; then
    jq -c --argjson now "$now" --arg installed "$(rel_installed)" \
      '. + {installed: $installed, stale: (($now - (.ok_at // 0)) > 259200)}
       | .available = (.available and .status != "insecure")' "$relfile"
  else
    jq -cn --arg installed "$(rel_installed)" \
      '{installed: $installed, status: "never", reason: "", checked_at: 0, ok_at: 0, available: false, latest: "", tag: "", commit: "", sequence: 0, published: "", summary: "", url: "", stale: true}'
  fi
}

# Resultado en pantalla, a partir del estado.
rel_show() {
  local st reason latest installed
  st="$(jq -r '.status' <<< "$1")"
  reason="$(jq -r '.reason' <<< "$1")"
  latest="$(jq -r '.latest' <<< "$1")"
  installed="$(jq -r '.installed' <<< "$1")"
  case "$st" in
    ok)
      if [ "$(jq -r '.available' <<< "$1")" = true ]; then
        ui_row ok @release.available "$latest" "$installed"
        local sum
        sum="$(jq -r '.summary' <<< "$1")"
        [ -z "$sum" ] || ui_row info "$sum"
        ui_row info @release.hint_apply
      else
        ui_row ok @release.up_to_date "$installed"
      fi
      ;;
    insecure) ui_row bad "$(rel_msgkey "$reason")" ;;
    unavailable) ui_row warn "$(rel_msgkey "$reason")" "${reason#http_}" ;;
    never) ui_row info @release.never ;;
  esac
}

# Mensaje del catálogo para el motivo de un estado que no es «ok» (http_404 → el de HTTP).
rel_msgkey() {
  case "$1" in
    unsigned) echo "@release.insecure_unsigned" ;;
    bad_signature) echo "@release.insecure_bad_signature" ;;
    invalid_manifest) echo "@release.insecure_invalid_manifest" ;;
    rollback) echo "@release.insecure_rollback" ;;
    http_*) echo "@release.unavailable_http" ;;
    no_key) echo "@release.unavailable_no_key" ;;
    *) echo "@release.unavailable_network" ;;
  esac
}

rel_help_exit() { usage_error release; }

cmd_release() {
  local sub="${1:-}" json=0 force=0 notify=0 yes=0 a
  [ -n "$sub" ] || rel_help_exit
  shift
  for a in "$@"; do
    case "$a" in
      --json) json=1 ;;
      --force) force=1 ;;
      --notify) notify=1 ;;
      -y | --yes) yes=1 ;;
      *) die_code "$EX_USAGE" @err.unknown_option "$a" ;;
    esac
  done
  case "$sub" in
    check)
      local st
      if [ "$json" = 1 ] || [ "$MAXOR_QUIET" = 1 ]; then
        rel_refresh "$force"
      else
        echo
        ui_intro @release.title
        ui_run @release.step_check rel_refresh "$force" || true
      fi
      [ "$notify" = 0 ] || rel_notify
      if [ "$json" = 1 ]; then
        rel_status_json
      elif [ "$MAXOR_QUIET" != 1 ]; then
        rel_show "$(rel_status_json)"
        ui_outro
      fi
      st="$(rel_status_json | jq -r '.status')"
      case "$st" in
        unavailable) return "$EX_NET" ;;
        insecure) return "$EX_FAIL" ;;
      esac
      return 0
      ;;
    status)
      if [ "$json" = 1 ]; then
        rel_status_json
      else
        echo
        ui_intro @release.title
        rel_show "$(rel_status_json)"
        ui_outro
      fi
      ;;
    apply) rel_apply "$yes" ;;
    *) rel_help_exit ;;
  esac
}

# Aplica la release avisada: comprueba que la etiqueta del repositorio esté firmada con la
# clave de release y que apunte al commit del manifiesto, avanza el repositorio solo en línea
# recta (nunca pisa commits tuyos) y actualiza con el flake.lock de esa versión.
rel_apply() {
  local yes="$1" js ver tag commit tagc before
  need_flake
  rel_has_key || die_code "$EX_NEEDS" @release.unavailable_no_key
  echo
  ui_intro @release.title
  git -C "$flake_dir" rev-parse --git-dir > /dev/null 2>&1 || die_code "$EX_NEEDS" @release.no_repo "$flake_dir"
  ui_run @release.step_check rel_refresh 1 || true
  js="$(rel_status_json)"
  case "$(jq -r '.status' <<< "$js")" in
    ok) ;;
    insecure) die_code "$EX_FAIL" "$(rel_msgkey "$(jq -r '.reason' <<< "$js")")" ;;
    *) die_code "$EX_NET" "$(rel_msgkey "$(jq -r '.reason' <<< "$js")")" "$(jq -r '.reason' <<< "$js" | sed 's/^http_//')" ;;
  esac
  if [ "$(jq -r '.available' <<< "$js")" != true ]; then
    ui_row ok @release.up_to_date "$(rel_installed)"
    ui_outro
    return 0
  fi
  ver="$(jq -r '.latest' <<< "$js")"
  tag="$(jq -r '.tag' <<< "$js")"
  commit="$(jq -r '.commit' <<< "$js")"
  [ -z "$(git -C "$flake_dir" status --porcelain 2> /dev/null)" ] || die_code "$EX_NEEDS" @release.dirty
  ui_run @release.step_fetch git -C "$flake_dir" fetch --quiet --tags origin || return $?
  if ! GIT_CONFIG_COUNT=2 GIT_CONFIG_KEY_0=gpg.format GIT_CONFIG_VALUE_0=ssh \
    GIT_CONFIG_KEY_1=gpg.ssh.allowedSignersFile GIT_CONFIG_VALUE_1="$MAXOR_RELEASE_KEYS" \
    git -C "$flake_dir" verify-tag "$tag" > /dev/null 2>&1; then
    die_code "$EX_FAIL" @release.tag_unsigned "$tag"
  fi
  tagc="$(git -C "$flake_dir" rev-parse "$tag^{commit}" 2> /dev/null || true)"
  [ "$tagc" = "$commit" ] || die_code "$EX_FAIL" @release.tag_mismatch "$tag"
  git -C "$flake_dir" merge-base --is-ancestor HEAD "$tag" 2> /dev/null || die_code "$EX_NEEDS" @release.diverged "$tag"
  ui_row ok @release.verified "$tag"
  if [ "$yes" = 0 ] && ! ui_confirm @release.confirm "$ver"; then
    ui_outro @update.cancelled
    return 0
  fi
  before="$(git -C "$flake_dir" rev-parse HEAD)"
  ui_run @release.step_merge git -C "$flake_dir" merge --ff-only --quiet "$tag" || return $?
  # Con el flake.lock que trae la release: lo que se probó es lo que se instala.
  if ! cmd_update --yes --no-lock; then
    ui_row warn @release.back_hint "${before:0:12}"
    return "$EX_FAIL"
  fi
  rel_write ok "" false "$reldir/manifest.json"
  ui_outro @release.applied "$ver"
}
