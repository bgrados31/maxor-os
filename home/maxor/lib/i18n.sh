# ── Idiomas: catálogo de mensajes ────────────────────────────────────
# Todo texto que ve el usuario vive en lib/lang/<idioma>.sh como una entrada
# MSG[clave]='formato printf'. El código usa «@clave» donde iría el texto:
#
#   ui_row ok @doctor.kernel_ok            # sin argumentos
#   ui_say warn @apps.not_installed "$id"  # con argumentos para el formato
#   msg texto @update.step_build "$host" # a una variable, para componer líneas
#
# Un texto que no empieza por «@» se imprime tal cual. Si falta una clave se
# muestra la propia clave (y tests/ avisa en CI). Hoy solo existe `en`; para
# añadir un idioma se crea lang/<código>.sh con msgs_<código>() y se registra en
# maxor.nix: lo que no traduzca cae al inglés.
declare -gA MSG=()
maxor_lang="${MAXOR_LANG:-${LANG:-en}}"
maxor_lang="${maxor_lang%%_*}" # es_PE.UTF-8 → es
maxor_lang="${maxor_lang%%.*}" # C.UTF-8 → C (sin catálogo: cae al inglés)

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
  msgs_en
  if [ "$maxor_lang" != en ] && declare -F "msgs_$maxor_lang" > /dev/null; then "msgs_$maxor_lang"; fi
}
