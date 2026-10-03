# ── Forma de los temas: style.json → Lua de Hyprland ─────────────────
# style.json solo contiene números. Nunca se ejecuta: maxor valida cada valor y
# genera el Lua que Hyprland carga, de modo que un tema no puede aportar código.

# Devuelve 0 si style.json es válido; si no, imprime el motivo y devuelve 1.
theme_check_style() {
  local f="$1"
  jq -e 'type == "object"' "$f" > /dev/null 2>&1 || { echo "style.json no es un objeto JSON"; return 1; }
  jq -e '
    def ok(k; lo; hi): (.[k] == null) or ((.[k] | type == "number") and (.[k] == (.[k] | floor)) and .[k] >= lo and .[k] <= hi);
    ok("rounding"; 0; 30) and ok("gaps_in"; 0; 20) and ok("gaps_out"; 0; 40) and ok("border_size"; 0; 6)
    and ok("blur_size"; 1; 20) and ok("blur_passes"; 1; 5) and ok("inactive"; 50; 100) and ok("anim"; 0; 300)
  ' "$f" > /dev/null \
    || { echo "style.json tiene valores fuera de rango (enteros: rounding 0-30, gaps_in 0-20, gaps_out 0-40, border_size 0-6, blur_size 1-20, blur_passes 1-5, inactive 50-100, anim 0-300)"; return 1; }
}

# Genera el Lua de Hyprland a partir de un style.json ya validado.
theme_style_lua() {
  jq -r '
    def kv(k; v): if v == null then null else "\(k) = \(v)," end;
    def block(name; lines): (lines | map(select(. != null))) as $l
      | if ($l | length) > 0 then ["  \(name) = {"] + ($l | map("    " + .)) + ["  },"] else [] end;
    ( block("general"; [kv("gaps_in"; .gaps_in), kv("gaps_out"; .gaps_out), kv("border_size"; .border_size)]) ) as $general
    | ( [kv("size"; .blur_size), kv("passes"; .blur_passes)] | map(select(. != null)) ) as $blur
    | ( [kv("rounding"; .rounding), kv("inactive_opacity"; (if .inactive == null then null else .inactive / 100 end))] | map(select(. != null)) ) as $deco
    | ( if ($blur | length) > 0 then $deco + ["blur = {"] + ($blur | map("  " + .)) + ["},"] else $deco end ) as $decoall
    | ( if ($decoall | length) > 0 then ["  decoration = {"] + ($decoall | map("    " + .)) + ["  },"] else [] end ) as $decoration
    | .anim as $an
    | ( if $an == null then [] else
        [["windowsIn", 3], ["windowsOut", 3], ["workspaces", 5], ["windowsMove", 4], ["fade", 3], ["border", 3]]
        | map("hl.animation({ leaf = \"\(.[0])\", enabled = \(if $an == 0 then "false" else "true" end), speed = \(.[1] * (if $an == 0 then 100 else $an end) / 100), bezier = \"default\" })")
      end ) as $anims
    | ["-- Generado por `maxor theme apply`: no lo edites, se reescribe con cada tema.", "hl.config({"]
      + $general + $decoration + ["})"] + $anims
    | .[]
  ' "$1"
}
