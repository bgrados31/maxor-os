# ── Base: rutas y utilidades de color ────────────────────────────────
# C.UTF-8 existe siempre en glibc: así wc y ${#var} cuentan caracteres, no bytes.
export LC_ALL=C.UTF-8

data="${XDG_DATA_HOME:-$HOME/.local/share}/maxor"
cfg="${XDG_CONFIG_HOME:-$HOME/.config}/maxor"
state="${XDG_STATE_HOME:-$HOME/.local/state}/maxor"
themes="$data/themes"
tpl="$data/templates"
dms_settings="${XDG_CONFIG_HOME:-$HOME/.config}/DankMaterialShell/settings.json"

hex() { printf '%s' "${1#\#}"; }
rgb() { local h; h="$(hex "$1")"; printf '%d, %d, %d' "0x${h:0:2}" "0x${h:2:2}" "0x${h:4:2}"; }
# mix A B P  →  A con P % de B
mix() {
  local a b p out="#" i x y
  a="$(hex "$1")"; b="$(hex "$2")"; p="$3"
  for i in 0 2 4; do
    x=$((0x${a:i:2})); y=$((0x${b:i:2}))
    out+="$(printf '%02x' $(((x * (100 - p) + y * p) / 100)))"
  done
  printf '%s' "$out"
}
