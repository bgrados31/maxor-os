-- Maxor OS: forma del tema activo (esquinas, espacios, desenfoque, animaciones).
--
-- `maxor theme apply` genera ~/.config/maxor/current/hyprland.lua a partir del
-- style.json del tema (solo números validados). Si el tema no define forma, o
-- aún no has aplicado ninguno, este módulo no hace nada y valen settings.lua.
local base = os.getenv("XDG_CONFIG_HOME") or ((os.getenv("HOME") or "") .. "/.config")
local path = base .. "/maxor/current/hyprland.lua"
local f = io.open(path, "r")
if f then
  f:close()
  pcall(dofile, path)
end
