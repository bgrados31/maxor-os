load helper

# Motor de releases (cmd/release.sh): firma, secuencia, estados, avisos y `apply`.
# Todo con archivos locales (file://) y claves ssh creadas aquí: sin red y sin claves reales.

setup() {
  load_lib
  export MAXOR_VERSION=0.1.0-dev
  W="$BATS_TEST_TMPDIR/w"
  mkdir -p "$W/chan" "$W/keys" "$W/bin"
  ssh-keygen -q -t ed25519 -N '' -C t -f "$W/keys/good"
  ssh-keygen -q -t ed25519 -N '' -C t -f "$W/keys/evil"
  printf 'maxor-release namespaces="git,maxor-release" %s\n' "$(cut -d' ' -f1,2 "$W/keys/good.pub")" > "$W/signers"
  MAXOR_RELEASE_KEYS="$W/signers"
  MAXOR_RELEASE_URL="file://$W/chan"
  printf '#!/bin/sh\necho "$*" >> %s/notes\n' "$W" > "$W/bin/notify-send"
  chmod +x "$W/bin/notify-send"
  PATH="$W/bin:$PATH"
}

# publish VERSION [SECUENCIA] [CLAVE] [COMMIT]: escribe y firma el manifiesto del canal
publish() {
  local v="$1" seq="${2:-1000}" k="${3:-good}" c="${4:-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa}"
  jq -n --arg v "$v" --argjson s "$seq" --arg c "$c" \
    '{schema: 1, product: "maxor-os", version: $v, tag: ("v" + $v), commit: $c, sequence: $s,
      published: "2026-10-03T00:00:00Z", summary: "A change", url: "https://example/x"}' > "$W/chan/manifest.json"
  rm -f "$W/chan/manifest.json.sig"
  ssh-keygen -Y sign -f "$W/keys/$k" -n maxor-release "$W/chan/manifest.json" > /dev/null 2>&1
}

field() { jq -r ".$1" "$relfile"; }

# ── versiones ────────────────────────────────────────────────────────

@test "ver_cmp ordena versiones y trata el sufijo como anterior" {
  [ "$(ver_cmp 0.1.0 0.1.0)" = 0 ]
  [ "$(ver_cmp 0.2.0 0.1.9)" = 1 ]
  [ "$(ver_cmp 0.1.9 0.2.0)" = -1 ]
  [ "$(ver_cmp 0.10.0 0.9.0)" = 1 ]
  [ "$(ver_cmp 0.1.0 0.1.0-dev)" = 1 ]
  [ "$(ver_cmp 0.1.0-dev 0.1.0)" = -1 ]
  [ "$(ver_cmp 1.0.0-rc.1 1.0.0-rc.2)" = -1 ]
}

@test "una compilación sin versión cuenta como la más vieja" {
  MAXOR_VERSION=dev
  [ "$(rel_installed)" = 0.0.0-dev ]
}

# ── comprobación ─────────────────────────────────────────────────────

@test "una release más nueva y bien firmada queda como disponible" {
  publish 0.2.0
  rel_refresh 1
  [ "$(field status)" = ok ]
  [ "$(field available)" = true ]
  [ "$(field latest)" = 0.2.0 ]
  [ "$(field tag)" = v0.2.0 ]
}

@test "la misma versión que la instalada no es una actualización" {
  MAXOR_VERSION=0.2.0
  publish 0.2.0
  rel_refresh 1
  [ "$(field status)" = ok ]
  [ "$(field available)" = false ]
}

@test "una versión de desarrollo posterior no se ofrece retroceder" {
  MAXOR_VERSION=0.2.0-dev
  publish 0.1.0
  rel_refresh 1
  [ "$(field available)" = false ]
}

@test "un manifiesto alterado después de firmar se rechaza" {
  publish 0.2.0
  jq '.version = "9.9.9" | .tag = "v9.9.9"' "$W/chan/manifest.json" > "$W/m" && mv "$W/m" "$W/chan/manifest.json"
  rel_refresh 1
  [ "$(field status)" = insecure ]
  [ "$(field reason)" = bad_signature ]
  [ "$(field available)" = false ]
}

@test "firmado con otra clave no se acepta" {
  publish 0.2.0 1000 evil
  rel_refresh 1
  [ "$(field status)" = insecure ]
  [ "$(field reason)" = bad_signature ]
}

@test "un manifiesto sin firma se rechaza" {
  publish 0.2.0
  rm "$W/chan/manifest.json.sig"
  rel_refresh 1
  [ "$(field status)" = insecure ]
  [ "$(field reason)" = unsigned ]
}

@test "una firma de otro espacio de nombres no vale" {
  publish 0.2.0
  rm "$W/chan/manifest.json.sig"
  ssh-keygen -Y sign -f "$W/keys/good" -n file "$W/chan/manifest.json" > /dev/null 2>&1
  rel_refresh 1
  [ "$(field status)" = insecure ]
}

@test "un manifiesto firmado pero mal formado se rechaza" {
  publish 0.2.0 1000 good notahash
  rel_refresh 1
  [ "$(field status)" = insecure ]
  [ "$(field reason)" = invalid_manifest ]
}

@test "un manifiesto de otro producto se rechaza" {
  publish 0.2.0
  jq '.product = "otra-cosa"' "$W/chan/manifest.json" > "$W/m" && mv "$W/m" "$W/chan/manifest.json"
  rm "$W/chan/manifest.json.sig"
  ssh-keygen -Y sign -f "$W/keys/good" -n maxor-release "$W/chan/manifest.json" > /dev/null 2>&1
  rel_refresh 1
  [ "$(field reason)" = invalid_manifest ]
}

@test "una secuencia menor que la ya vista es un retroceso y se rechaza" {
  publish 0.3.0 2000
  rel_refresh 1
  [ "$(field status)" = ok ]
  publish 0.2.0 1500 # firmado y válido, pero más viejo que lo ya visto
  rel_refresh 1
  [ "$(field status)" = insecure ]
  [ "$(field reason)" = rollback ]
  [ "$(cat "$reldir/sequence")" = 2000 ]
}

@test "lo rechazado no reemplaza lo bueno guardado" {
  publish 0.2.0
  rel_refresh 1
  publish 0.9.0 3000 evil
  rel_refresh 1
  [ "$(jq -r .version "$reldir/manifest.json")" = 0.2.0 ]
}

@test "sin red no se dice que está al día" {
  publish 0.2.0
  rel_refresh 1
  MAXOR_RELEASE_URL="file://$W/no-existe"
  rel_refresh 1
  [ "$(field status)" = unavailable ]
  [ "$(field reason)" = network ]
  [ "$(field available)" = true ] # se conserva lo último verificado
  [ "$(field ok_at)" -gt 0 ]
}

@test "sin ninguna clave de confianza no se confía en nada" {
  printf '# solo comentarios\n' > "$W/signers"
  publish 0.2.0
  rel_refresh 1
  [ "$(field status)" = unavailable ]
  [ "$(field reason)" = no_key ]
  [ "$(field available)" = false ]
}

@test "check sin --force no vuelve a la red dentro del margen" {
  publish 0.2.0
  rel_refresh 1
  MAXOR_RELEASE_URL="file://$W/no-existe"
  rel_refresh 0
  [ "$(field status)" = ok ]
}

@test "check con --force sí vuelve a la red" {
  publish 0.2.0
  rel_refresh 1
  MAXOR_RELEASE_URL="file://$W/no-existe"
  rel_refresh 1
  [ "$(field status)" = unavailable ]
}

# ── interfaz de la orden ─────────────────────────────────────────────

@test "check --json imprime el estado y sale con 0" {
  publish 0.2.0
  run cmd_release check --json --force
  [ "$status" = 0 ]
  [ "$(jq -r .latest <<< "$output")" = 0.2.0 ]
  [ "$(jq -r .available <<< "$output")" = true ]
}

@test "check sale con el código de red cuando no hay canal" {
  MAXOR_RELEASE_URL="file://$W/no-existe"
  run cmd_release check --json --force
  [ "$status" = "$EX_NET" ]
  [ "$(jq -r .status <<< "$output")" = unavailable ]
}

@test "check sale con error cuando algo no es de fiar" {
  publish 0.2.0 1000 evil
  run cmd_release check --json --force
  [ "$status" = "$EX_FAIL" ]
  [ "$(jq -r .available <<< "$output")" = false ]
}

@test "status --json no toca la red y marca lo viejo como stale" {
  run cmd_release status --json
  [ "$status" = 0 ]
  [ "$(jq -r .status <<< "$output")" = never ]
  [ "$(jq -r .stale <<< "$output")" = true ]
}

@test "status --json de un estado reciente no es stale" {
  publish 0.2.0
  rel_refresh 1
  [ "$(rel_status_json | jq -r .stale)" = false ]
}

@test "una opción desconocida es un error de uso" {
  run cmd_release check --nope
  [ "$status" = "$EX_USAGE" ]
}

@test "release sin subcomando muestra la ayuda y sale con uso" {
  run cmd_release
  [ "$status" = "$EX_USAGE" ]
}

# ── avisos ───────────────────────────────────────────────────────────

@test "avisa una sola vez por versión" {
  publish 0.2.0
  rel_refresh 1
  rel_notify
  rel_notify
  [ "$(wc -l < "$W/notes")" = 1 ]
  publish 0.3.0 2000
  rel_refresh 1
  rel_notify
  [ "$(wc -l < "$W/notes")" = 2 ]
  [[ "$(tail -n1 "$W/notes")" == *0.3.0* ]]
}

@test "no avisa si no hay nada nuevo" {
  MAXOR_VERSION=0.2.0
  publish 0.2.0
  rel_refresh 1
  rel_notify
  [ ! -e "$W/notes" ]
}

@test "avisa de una firma mala, crítico y una sola vez" {
  publish 0.2.0 1000 evil
  rel_refresh 1
  rel_notify
  rel_notify
  [ "$(wc -l < "$W/notes")" = 1 ]
  [[ "$(cat "$W/notes")" == *critical* ]]
}

@test "lo inseguro nunca se anuncia como actualización" {
  publish 0.2.0 1000 evil
  rel_refresh 1
  rel_notify
  [[ "$(cat "$W/notes")" != *"is out"* ]]
}

# ── apply ────────────────────────────────────────────────────────────

# mkrepo: un remoto con la release v0.2.0 firmada y una copia en la versión anterior
mkrepo() {
  git init -q -b main "$W/up"
  git -C "$W/up" config user.email t@t
  git -C "$W/up" config user.name t
  echo 1 > "$W/up/flake.nix"
  git -C "$W/up" add .
  git -C "$W/up" commit -qm uno
  git clone -q "$W/up" "$W/clone"
  git -C "$W/clone" config user.email t@t
  git -C "$W/clone" config user.name t
  echo 2 > "$W/up/cambio"
  git -C "$W/up" add .
  git -C "$W/up" commit -qm dos
  RELEASE_COMMIT="$(git -C "$W/up" rev-parse HEAD)"
  MAXOR_FLAKE="$W/clone"
  flake_dir="$W/clone"
  cmd_update() { echo "UPDATED $*"; }
}

signtag() { git -C "$W/up" -c gpg.format=ssh -c "user.signingkey=$W/keys/${1:-good}" tag -s -m rel v0.2.0; }

@test "apply avanza el repositorio y actualiza con el lock de la release" {
  mkrepo
  signtag
  publish 0.2.0 1000 good "$RELEASE_COMMIT"
  run cmd_release apply --yes
  [ "$status" = 0 ]
  [[ "$output" == *"UPDATED --yes --no-lock"* ]]
  [ "$(git -C "$W/clone" rev-parse HEAD)" = "$RELEASE_COMMIT" ]
}

@test "apply se niega si la etiqueta no está firmada" {
  mkrepo
  git -C "$W/up" tag -a -m rel v0.2.0
  publish 0.2.0 1000 good "$RELEASE_COMMIT"
  before="$(git -C "$W/clone" rev-parse HEAD)"
  run cmd_release apply --yes
  [ "$status" = "$EX_FAIL" ]
  [[ "$output" != *UPDATED* ]]
  [ "$(git -C "$W/clone" rev-parse HEAD)" = "$before" ]
}

@test "apply se niega si la etiqueta la firmó otra clave" {
  mkrepo
  signtag evil
  publish 0.2.0 1000 good "$RELEASE_COMMIT"
  run cmd_release apply --yes
  [ "$status" = "$EX_FAIL" ]
  [[ "$output" != *UPDATED* ]]
}

@test "apply se niega si el manifiesto nombra otro commit que la etiqueta" {
  mkrepo
  signtag
  publish 0.2.0 1000 good bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
  run cmd_release apply --yes
  [ "$status" = "$EX_FAIL" ]
  [[ "$output" != *UPDATED* ]]
}

@test "apply se niega con cambios sin commit" {
  mkrepo
  signtag
  publish 0.2.0 1000 good "$RELEASE_COMMIT"
  echo sucio >> "$W/clone/flake.nix"
  run cmd_release apply --yes
  [ "$status" = "$EX_NEEDS" ]
  [[ "$output" != *UPDATED* ]]
}

@test "apply no pisa commits propios: se niega si el repositorio divergió" {
  mkrepo
  echo mio > "$W/clone/mio"
  git -C "$W/clone" add .
  git -C "$W/clone" commit -qm mio
  signtag
  publish 0.2.0 1000 good "$RELEASE_COMMIT"
  before="$(git -C "$W/clone" rev-parse HEAD)"
  run cmd_release apply --yes
  [ "$status" = "$EX_NEEDS" ]
  [ "$(git -C "$W/clone" rev-parse HEAD)" = "$before" ]
}

@test "apply no hace nada si ya está al día" {
  mkrepo
  MAXOR_VERSION=0.2.0
  signtag
  publish 0.2.0 1000 good "$RELEASE_COMMIT"
  run cmd_release apply --yes
  [ "$status" = 0 ]
  [[ "$output" != *UPDATED* ]]
}

@test "apply se niega con un manifiesto de firma mala" {
  mkrepo
  signtag
  publish 0.2.0 1000 evil "$RELEASE_COMMIT"
  run cmd_release apply --yes
  [ "$status" = "$EX_FAIL" ]
  [[ "$output" != *UPDATED* ]]
}

# ── lado de quien publica ────────────────────────────────────────────

@test "release-manifest genera un manifiesto con la forma que la CLI exige" {
  mkdir -p "$W/r/scripts"
  cp "$ROOT/scripts/release-manifest.sh" "$ROOT/scripts/release-notes.sh" "$W/r/scripts/"
  printf '## [Sin publicar]\n\n## [0.2.0] - 2026-10-03\n\n- **Cosa nueva**: detalle.\n' > "$W/r/CHANGELOG.md"
  "$W/r/scripts/release-manifest.sh" 0.2.0 cccccccccccccccccccccccccccccccccccccccc > "$W/chan/manifest.json"
  [ "$(jq -r .summary "$W/chan/manifest.json")" = "Cosa nueva" ]
  ssh-keygen -Y sign -f "$W/keys/good" -n maxor-release "$W/chan/manifest.json" > /dev/null 2>&1
  rel_refresh 1
  [ "$(field status)" = ok ]
  [ "$(field commit)" = cccccccccccccccccccccccccccccccccccccccc ]
}

@test "release-manifest rechaza un commit abreviado" {
  run "$ROOT/scripts/release-manifest.sh" 0.2.0 abc123
  [ "$status" != 0 ]
}

@test "release-manifest sin negrita corta el resumen en una palabra entera" {
  mkdir -p "$W/r/scripts"
  cp "$ROOT/scripts/release-manifest.sh" "$ROOT/scripts/release-notes.sh" "$W/r/scripts/"
  printf '## [Sin publicar]\n\n## [0.2.0] - 2026-10-03\n\n- Un cambio largo sin ningún título en negrita que sigue y sigue explicándose durante mucho rato más\n' > "$W/r/CHANGELOG.md"
  s="$("$W/r/scripts/release-manifest.sh" 0.2.0 cccccccccccccccccccccccccccccccccccccccc | jq -r .summary)"
  [ "${#s}" -le 90 ]
  [[ "$s" == "Un cambio largo"* ]]
  [[ "$s" != *" " ]]
  [[ "${s##* }" =~ ^[a-zA-Zñáéíóú]+$ ]]
}
