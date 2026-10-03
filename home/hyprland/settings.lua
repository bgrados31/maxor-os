-- Maxor OS: apariencia, entrada y animaciones de Hyprland.
-- Se instala en ~/.config/hypr/maxor/settings.lua (solo lectura).
-- Para cambiar algo sin tocar el sistema, usa user.lua.

hl.config({
  input = {
    kb_layout = "latam",
    numlock_by_default = true,
    follow_mouse = 1,
    touchpad = {
      tap_to_click = true,
      natural_scroll = true,
    },
  },
  general = {
    gaps_in = 5,
    gaps_out = 10,
    border_size = 2,
    layout = "dwindle",
  },
  decoration = {
    rounding = 8,
    active_opacity = 1.0,
    inactive_opacity = 0.95,
    blur = {
      enabled = true,
      size = 8,
      passes = 3,
    },
    shadow = {
      enabled = true,
      range = 30,
      render_power = 5,
      offset = "0 5",
      color = "rgba(00000070)",
    },
  },
  misc = {
    disable_hyprland_logo = true,
    disable_splash_rendering = true,
  },
  dwindle = {
    preserve_split = true,
  },
})

hl.animation({ leaf = "windowsIn", enabled = true, speed = 3, bezier = "default" })
hl.animation({ leaf = "windowsOut", enabled = true, speed = 3, bezier = "default" })
hl.animation({ leaf = "workspaces", enabled = true, speed = 5, bezier = "default" })
hl.animation({ leaf = "windowsMove", enabled = true, speed = 4, bezier = "default" })
hl.animation({ leaf = "fade", enabled = true, speed = 3, bezier = "default" })
hl.animation({ leaf = "border", enabled = true, speed = 3, bezier = "default" })
