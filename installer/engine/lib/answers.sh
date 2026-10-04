# ── Respuestas: validar contra el esquema y completar los valores por defecto ─
# El esquema (installer/schema/answers.v1.json) es la única lista de campos: este archivo no
# repite nombres, solo lo interpreta. Las reglas que mezclan varios campos van aparte.

# Validador de esquema en jq: type, enum, pattern, minimum, maximum, required, properties
# (las claves que no están en `properties` son un error) e items.
ANS_JQ_VALIDATE='
def typeok($t):
  if $t == "object" then type == "object" elif $t == "string" then type == "string"
  elif $t == "integer" then (type == "number" and . == floor) elif $t == "boolean" then type == "boolean"
  elif $t == "array" then type == "array" else true end;
def v($s; $p):
  if ($s | has("type")) and (typeok($s.type) | not) then "\($p): must be of type \($s.type)"
  else
    (if ($s | has("enum")) and ((. as $x | $s.enum | index($x)) == null) then "\($p): must be one of \($s.enum | map(tostring) | join(", "))" else empty end),
    (if ($s | has("pattern")) and (type == "string") and (test($s.pattern) | not) then "\($p): has an invalid format" else empty end),
    (if ($s | has("minimum")) and (type == "number") and (. < $s.minimum) then "\($p): must be at least \($s.minimum)" else empty end),
    (if ($s | has("maximum")) and (type == "number") and (. > $s.maximum) then "\($p): must be at most \($s.maximum)" else empty end),
    (if type == "object" and ($s | has("properties")) then
       (($s.required // [])[] as $r | select(has($r) | not) | "\($p).\($r): is required"),
       (keys[] as $k | select(($s.properties | has($k)) | not) | "\($p).\($k): unknown field"),
       (. as $d | $s.properties | keys[] as $k | select($d | has($k)) | $d[$k] | v($s.properties[$k]; "\($p).\($k)"))
     else empty end),
    (if type == "array" and ($s | has("items")) then to_entries[] | . as $e | .value | v($s.items; "\($p)[\($e.key)]") else empty end)
  end;
v($s[0]; "answers")
'

# Completa con los valores por defecto del esquema lo que falte (también dentro de objetos).
ANS_JQ_DEFAULTS='
def fill($s):
  if type == "object" and ($s | has("properties")) then
    . as $o
    | reduce ($s.properties | to_entries[]) as $e ($o;
        if has($e.key) then .[$e.key] |= fill($e.value)
        elif ($e.value | has("default")) then .[$e.key] = ($e.value.default | fill($e.value))
        else . end)
  else . end;
fill($s[0])
'

# Reglas que mezclan campos. Cada salida es un error legible; el hash de la contraseña no aparece.
ANS_JQ_CROSS='
def reserved: ["root","daemon","bin","sys","nobody","nixbld","maxor","admin","guest","shutdown","halt"];
(if .disk.strategy == "whole" and (.disk.confirmed // "") != "ERASE" then
   "disk.confirmed: type ERASE to confirm that the whole disk will be erased" else empty end),
(if .disk.strategy == "alongside" then
   (if (.disk.region // null) == null then "disk.region: is required to install alongside another system" else empty end),
   (if (.disk.region // null) != null and .disk.region.end <= .disk.region.start then "disk.region: end must be after start" else empty end),
   (if (.disk.plan_hash // "") == "" then "disk.plan_hash: is required: it proves you saw this exact plan" else empty end),
   (if (.disk.confirmed // "") != "INSTALL" then "disk.confirmed: type INSTALL to confirm the plan" else empty end)
 else empty end),
(if (.disk.swap.kind == "file") and (.disk.swap.gib < 1) then "disk.swap.gib: a swap file needs at least 1 GiB" else empty end),
(if (.user.name | IN(reserved[])) then "user.name: is a reserved name" else empty end),
(if .machine.hostname == "localhost" then "machine.hostname: cannot be localhost" else empty end),
(.look.profiles | (. - ($cat | map(tostring))) | map("look.profiles: unknown profile \(.)")[]),
(if (.look.profiles | length) != (.look.profiles | unique | length) then "look.profiles: has repeated profiles" else empty end)
'

# ans_load ARCHIVO → deja en $IN_ANSWERS las respuestas completas; imprime los errores y devuelve 1.
# ans_load ARCHIVO [structure] → con «structure» solo se comprueba la forma: sirve para calcular la huella
# del plan antes de tener la confirmación (que se pide después de enseñar el resumen).
ans_load() {
  local file="$1" mode="${2:-full}" errs raw cat
  [ -r "$file" ] || in_die "$IN_EX_USAGE" "cannot read the answers file: $file"
  [ -r "$IN_SCHEMA" ] || in_die "$IN_EX_USAGE" "the answers schema is missing: $IN_SCHEMA"
  jq -e . "$file" > /dev/null 2>&1 || in_die "$IN_EX_ANSWERS" "the answers file is not valid JSON"
  errs="$(jq -r --slurpfile s "$IN_SCHEMA" "$ANS_JQ_VALIDATE" "$file")"
  if [ -n "$errs" ]; then
    printf '%s\n' "$errs" >&2
    return 1
  fi
  raw="$(jq -c --slurpfile s "$IN_SCHEMA" "$ANS_JQ_DEFAULTS" "$file")"
  if [ "$mode" = structure ]; then
    IN_ANSWERS="$(mktemp)"
    chmod 600 "$IN_ANSWERS"
    printf '%s\n' "$raw" > "$IN_ANSWERS"
    return 0
  fi
  cat='[]'
  if [ -n "${MAXOR_PROFILES:-}" ] && [ -r "$MAXOR_PROFILES" ]; then cat="$(jq -c 'keys' "$MAXOR_PROFILES")"; fi
  errs="$(jq -r --argjson cat "$cat" "$ANS_JQ_CROSS" <<< "$raw")"
  if [ -n "$errs" ]; then
    printf '%s\n' "$errs" >&2
    return 1
  fi
  IN_ANSWERS="$(mktemp)"
  chmod 600 "$IN_ANSWERS"
  printf '%s\n' "$raw" > "$IN_ANSWERS"
}

# Huella del plan: lo que decide qué se escribe en el disco, sin secretos. La pantalla la enseña
# junto al resumen y, para instalar junto a otro sistema, el motor exige que coincida.
ans_plan_hash() {
  jq -S -c 'del(.user.password_hash) | del(.disk.plan_hash) | del(.disk.confirmed)' "$IN_ANSWERS" | sha256sum | cut -d' ' -f1
}
