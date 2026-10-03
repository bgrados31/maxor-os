-- Maxor OS: forma del tema activo (esquinas, espacios, desenfoque, animaciones).
--
-- Se carga DESPUÉS de los módulos de DMS, porque DMS genera su propio layout
-- (esquinas y espacios) y, si este módulo fuera antes, lo pisaría.
--
-- `maxor theme apply` genera ~/.config/maxor/current/hyprland.lua a partir del
-- style.json del tema (solo números validados). Si el tema no define forma, o
-- aún no has aplicado ninguno, se usa la forma por defecto de Maxor (la de
-- Sakura nocturna). Mantén estos valores iguales a themes/sakura/style.json.
local base = os.getenv("XDG_CONFIG_HOME") or ((os.getenv("HOME") or "") .. "/.config")
local path = base .. "/maxor/current/hyprland.lua"

local loaded = false
local f = io.open(path, "r")
if f then
  f:close()
  loaded = pcall(dofile, path)
end

if not loaded then
  hl.config({
    general = { gaps_in = 5, gaps_out = 10, border_size = 2 },
    decoration = { rounding = 8 },
  })
end
