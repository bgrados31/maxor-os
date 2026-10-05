# ── Idiomas: catálogo de mensajes ────────────────────────────────────
# Todo texto que ve el usuario vive en lib/lang/<idioma>.sh como una entrada
# MSG[clave]='formato printf'. El código usa «@clave» donde iría el texto:
#
#   ui_row ok @doctor.kernel_ok            # sin argumentos
#   ui_say warn @apps.not_installed "$id"  # con argumentos para el formato
#   msg texto @update.step_build "$host" # a una variable, para componer líneas
#
# Un texto que no empieza por «@» se imprime tal cual. Si falta una clave se
# muestra la propia clave (y tests/ avisa en CI).
#
# Idiomas: lib/lang/<código>.sh define msgs_<código>() con las claves que traduce; se
# registra en packages/maxor.nix. Lo que no traduce cae al inglés, así que una traducción
# parcial ya sirve. Un código regional (pt_BR) se carga encima del de su idioma (pt), de
# modo que solo necesita lo que cambia. Cómo añadir uno: docs/TRANSLATING.md.
declare -gA MSG=()
maxor_lang="${MAXOR_LANG:-${LANG:-en}}"
# i18n_init lo normaliza: es_PE.UTF-8 → es_PE, pt-BR → pt_BR, C.UTF-8 → C (sin catálogo: inglés)

msg() { # msg variable texto|@clave [argumentos…]
  local _v="$1" _k="$2" _f
  shift 2
  if [ "${_k:0:1}" = "@" ]; then
    _f="${MSG[${_k:1}]-}"
    [ -n "$_f" ] || _f="${_k:1}"
    # shellcheck disable=SC2059
    printf -v "$_v" "$_f" "$@"
  else
    printf -v "$_v" '%s' "$_k"
  fi
}
t() { local _o; msg _o "@$1" "${@:2}"; printf '%s' "$_o"; } # t clave [argumentos…] → imprime

i18n_init() {
  local full="${maxor_lang%%.*}" base
  full="${full%%@*}"
  full="${full//-/_}"
  base="${full%%_*}"
  maxor_lang="$base"
  msgs_en
  if [ "$base" != en ] && declare -F "msgs_$base" > /dev/null; then "msgs_$base"; fi
  if [ "$full" != "$base" ] && declare -F "msgs_$full" > /dev/null; then "msgs_$full"; maxor_lang="$full"; fi
}
