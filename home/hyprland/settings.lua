-- Maxor OS: apariencia, entrada y animaciones de Hyprland.
-- Se instala en ~/.config/hypr/maxor/settings.lua (solo lectura).
-- Para cambiar algo sin tocar el sistema, usa user.lua.

local machine = require("maxor.machine") -- teclado de la máquina (lo genera Nix)

-- Cada ajuste va protegido: si una opción no existe en tu versión de Hyprland, solo ese ajuste se
-- ignora y el resto de la sesión carga igual.
local function try(f, ...)
  return (pcall(f, ...))
end

local sections = {
  {
    input = {
      kb_layout = machine.kb_layout,
      kb_variant = machine.kb_variant,
      numlock_by_default = true,
      follow_mouse = 1,
      touchpad = { tap_to_click = true, natural_scroll = true },
    },
  },
  { input = { repeat_rate = 40, repeat_delay = 300, touchpad = { disable_while_typing = true, clickfinger_behavior = true } } },
  {
    general = {
      gaps_in = 5,
      gaps_out = 10,
      border_size = 2,
      layout = "dwindle",
    },
  },
  { general = { resize_on_border = true, extend_border_grab_area = 12 } },
  {
    decoration = {
      rounding = 8,
      active_opacity = 1.0,
      inactive_opacity = 0.95,
      blur = { enabled = true, size = 8, passes = 3 },
      shadow = {
        enabled = true,
        range = 30,
        render_power = 4,
        offset = "0 8",
        color = "rgba(00000066)",
      },
    },
  },
  { decoration = { dim_inactive = true, dim_strength = 0.06 } },
  {
    decoration = {
      blur = {
        noise = 0.02,
        contrast = 0.95,
        vibrancy = 0.2,
        vibrancy_darkness = 0.1,
        new_optimizations = true,
        ignore_opacity = true,
        popups = true,
      },
    },
  },
  {
    misc = {
      disable_hyprland_logo = true,
      disable_splash_rendering = true,
      force_default_wallpaper = 0,
    },
  },
  {
    misc = {
      animate_manual_resizes = true,
      focus_on_activate = true,
      mouse_move_enables_dpms = true,
      key_press_enables_dpms = true,
    },
  },
  { cursor = { inactive_timeout = 5, hide_on_key_press = true } }, -- esconde el cursor al escribir o tras 5 s quieto
  { dwindle = { preserve_split = true } },
  { gestures = { workspace_swipe_distance = 300, workspace_swipe_cancel_ratio = 0.2 } },
}
for _, section in ipairs(sections) do
  try(hl.config, section)
end

-- Curvas de movimiento de Maxor (docs/IDENTITY.md: fluido, ~380 ms): "maxor" es la estándar y
-- "maxor-out" frena al final, para lo que aparece. Si la versión no las acepta, se usa "default".
MAXOR_CURVE = "default"
local curve_out = "default"
if try(hl.curve, "maxor", { type = "bezier", points = { { 0.4, 0 }, { 0.2, 1 } } }) then
  MAXOR_CURVE = "maxor"
end
if try(hl.curve, "maxor-out", { type = "bezier", points = { { 0.16, 1 }, { 0.3, 1 } } }) then
  curve_out = "maxor-out"
end

local function anim(leaf, speed, bezier, style)
  local spec = { leaf = leaf, enabled = true, speed = speed, bezier = bezier }
  if style then spec.style = style end
  if not try(hl.animation, spec) and style then
    spec.style = nil
    try(hl.animation, spec)
  end
end

anim("windowsIn", 4, curve_out, "popin 90%")
anim("windowsOut", 3, MAXOR_CURVE, "popin 90%")
anim("windowsMove", 4, MAXOR_CURVE)
anim("workspaces", 5, curve_out, "slide")
anim("fade", 3, MAXOR_CURVE)
anim("border", 4, MAXOR_CURVE)
anim("layers", 4, curve_out, "fade")
