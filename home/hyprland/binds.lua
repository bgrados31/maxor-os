-- Maxor OS: atajos de teclado y ratón, mismos que Ryoku OS.
-- Tus propios atajos van en user.lua, no aquí.
local mod = "SUPER"

-- ── Ventanas ──
hl.bind(mod .. " + Q", hl.dsp.window.close())
hl.bind(mod .. " + F", hl.dsp.window.fullscreen())
hl.bind(mod .. " + D", hl.dsp.window.fullscreen({ mode = "maximized" }))
hl.bind(mod .. " + C", hl.dsp.window.center())
hl.bind(mod .. " + SHIFT + P", hl.dsp.window.pin())
hl.bind(mod .. " + T", hl.dsp.group.toggle())
hl.bind("ALT + Tab", hl.dsp.focus({ last = true }))
-- Flotar/tilear: al flotar queda centrada y de tamaño cómodo
hl.bind(mod .. " + A", function()
  hl.dispatch(hl.dsp.window.float({ action = "toggle" }))
  hl.dispatch(hl.dsp.window.resize({ x = 1000, y = 660, exact = true }))
  hl.dispatch(hl.dsp.window.center())
end)

-- Modo redimensionar: flechas o hjkl, Esc/Enter sale
local step = 40
hl.define_submap("resize", function()
  hl.bind("Left",  hl.dsp.window.resize({ x = -step, y = 0, relative = true }), { repeating = true })
  hl.bind("Right", hl.dsp.window.resize({ x = step, y = 0, relative = true }), { repeating = true })
  hl.bind("Up",    hl.dsp.window.resize({ x = 0, y = -step, relative = true }), { repeating = true })
  hl.bind("Down",  hl.dsp.window.resize({ x = 0, y = step, relative = true }), { repeating = true })
  hl.bind("h", hl.dsp.window.resize({ x = -step, y = 0, relative = true }), { repeating = true })
  hl.bind("l", hl.dsp.window.resize({ x = step, y = 0, relative = true }), { repeating = true })
  hl.bind("k", hl.dsp.window.resize({ x = 0, y = -step, relative = true }), { repeating = true })
  hl.bind("j", hl.dsp.window.resize({ x = 0, y = step, relative = true }), { repeating = true })
  hl.bind("Escape", hl.dsp.submap("reset"))
  hl.bind("Return", hl.dsp.submap("reset"))
  hl.bind("SUPER + R", hl.dsp.submap("reset"))
end)
hl.bind(mod .. " + R", function()
  hl.dispatch(hl.dsp.submap("resize"))
  hl.dispatch(hl.dsp.exec_cmd("hyprctl notify -1 2200 0 'Resize mode: arrows or hjkl resize, Esc exits'"))
end)

-- ── Foco y mover ──
hl.bind(mod .. " + Left", hl.dsp.focus({ direction = "left" }))
hl.bind(mod .. " + Right", hl.dsp.focus({ direction = "right" }))
hl.bind(mod .. " + Up", hl.dsp.focus({ direction = "up" }))
hl.bind(mod .. " + Down", hl.dsp.focus({ direction = "down" }))
hl.bind(mod .. " + SHIFT + Left", hl.dsp.window.move({ direction = "left" }))
hl.bind(mod .. " + SHIFT + Right", hl.dsp.window.move({ direction = "right" }))
hl.bind(mod .. " + SHIFT + Up", hl.dsp.window.move({ direction = "up" }))
hl.bind(mod .. " + SHIFT + Down", hl.dsp.window.move({ direction = "down" }))
hl.bind(mod .. " + CTRL + Left", hl.dsp.window.resize({ x = -40, y = 0, relative = true }), { repeating = true })
hl.bind(mod .. " + CTRL + Right", hl.dsp.window.resize({ x = 40, y = 0, relative = true }), { repeating = true })
hl.bind(mod .. " + CTRL + Up", hl.dsp.window.resize({ x = 0, y = -40, relative = true }), { repeating = true })
hl.bind(mod .. " + CTRL + Down", hl.dsp.window.resize({ x = 0, y = 40, relative = true }), { repeating = true })
hl.bind(mod .. " + bracketleft", hl.dsp.window.move({ direction = "left", group_aware = true }))
hl.bind(mod .. " + bracketright", hl.dsp.window.move({ direction = "right", group_aware = true }))

-- ── Pantallas ──
for key, dir in pairs({ Left = "l", Right = "r", Up = "u", Down = "d" }) do
  hl.bind(mod .. " + ALT + " .. key, hl.dsp.focus({ monitor = dir }))
  hl.bind(mod .. " + ALT + SHIFT + " .. key, hl.dsp.window.move({ monitor = dir }))
  hl.bind(mod .. " + CTRL + ALT + " .. key, hl.dsp.workspace.move({ monitor = dir }))
end

-- ── Apps ──
hl.bind(mod .. " + Return", hl.dsp.exec_cmd("kitty"))
hl.bind(mod .. " + E", hl.dsp.exec_cmd("thunar"))
hl.bind(mod .. " + B", hl.dsp.exec_cmd("firefox"))
hl.bind(mod .. " + N", hl.dsp.exec_cmd("kitty -e nvim"))
hl.bind(mod .. " + O", hl.dsp.exec_cmd("dms ipc call notepad toggle"))

-- ── Shell (DankMaterialShell) ──
hl.bind(mod .. " + Space", hl.dsp.exec_cmd("dms ipc call spotlight toggle"))
hl.bind(mod .. " + K", hl.dsp.exec_cmd("dms ipc call keybinds toggle hyprland"))
hl.bind(mod .. " + L", hl.dsp.exec_cmd("loginctl lock-session"))
hl.bind(mod .. " + Escape", hl.dsp.exec_cmd("dms ipc call powermenu toggle"))
hl.bind(mod .. " + W", hl.dsp.exec_cmd("dms ipc call dash toggle wallpaper"))
hl.bind(mod .. " + SHIFT + W", hl.dsp.exec_cmd("dms ipc call wallpaper next"))
hl.bind(mod .. " + V", hl.dsp.exec_cmd("dms ipc call clipboard toggle"))
hl.bind(mod .. " + Tab", hl.dsp.exec_cmd("dms ipc call hypr toggleOverview"))
hl.bind(mod .. " + M", hl.dsp.exec_cmd("dms ipc call processlist focusOrToggle"))
hl.bind(mod .. " + comma", hl.dsp.exec_cmd("dms ipc call settings focusOrToggle"))
hl.bind(mod .. " + SHIFT + N", hl.dsp.exec_cmd("dms ipc call notifications toggle"))
hl.bind(mod .. " + SHIFT + C", hl.dsp.exec_cmd("dms ipc call color-picker toggle"))
hl.bind(mod .. " + SHIFT + A", hl.dsp.exec_cmd("systemctl --user restart pipewire pipewire-pulse wireplumber"))
hl.bind(mod .. " + SHIFT + E", hl.dsp.exit())

-- ── Capturas ──
hl.bind(mod .. " + SHIFT + S", hl.dsp.exec_cmd("dms screenshot"))
hl.bind("Print", hl.dsp.exec_cmd("dms screenshot"))
hl.bind("SHIFT + Print", hl.dsp.exec_cmd("dms screenshot full"))

-- ── Audio / brillo / media ──
hl.bind("XF86AudioRaiseVolume", hl.dsp.exec_cmd("dms ipc call audio increment 3"), { locked = true, repeating = true })
hl.bind("XF86AudioLowerVolume", hl.dsp.exec_cmd("dms ipc call audio decrement 3"), { locked = true, repeating = true })
hl.bind("XF86AudioMute", hl.dsp.exec_cmd("dms ipc call audio mute"), { locked = true })
hl.bind("XF86AudioMicMute", hl.dsp.exec_cmd("dms ipc call audio micmute"), { locked = true })
hl.bind("XF86AudioPlay", hl.dsp.exec_cmd("dms ipc call mpris playPause"), { locked = true })
hl.bind("XF86AudioPause", hl.dsp.exec_cmd("dms ipc call mpris playPause"), { locked = true })
hl.bind("XF86AudioPrev", hl.dsp.exec_cmd("dms ipc call mpris previous"), { locked = true })
hl.bind("XF86AudioNext", hl.dsp.exec_cmd("dms ipc call mpris next"), { locked = true })
hl.bind("XF86MonBrightnessUp", hl.dsp.exec_cmd([[dms ipc call brightness increment 5 ""]]), { locked = true, repeating = true })
hl.bind("XF86MonBrightnessDown", hl.dsp.exec_cmd([[dms ipc call brightness decrement 5 ""]]), { locked = true, repeating = true })

-- ── Workspaces 1-10 (la tecla 0 es el 10) ──
for i = 1, 10 do
  local key = tostring(i % 10)
  hl.bind(mod .. " + " .. key, hl.dsp.focus({ workspace = tostring(i) }))
  hl.bind(mod .. " + ALT + " .. key, hl.dsp.window.move({ workspace = tostring(i) }))
  hl.bind(mod .. " + SHIFT + " .. key, hl.dsp.window.move({ workspace = tostring(i), follow = false }))
end
hl.bind(mod .. " + Prior", hl.dsp.focus({ workspace = "r-1" }))
hl.bind(mod .. " + Next", hl.dsp.focus({ workspace = "r+1" }))
hl.bind(mod .. " + SHIFT + Prior", hl.dsp.window.move({ workspace = "r-1" }))
hl.bind(mod .. " + SHIFT + Next", hl.dsp.window.move({ workspace = "r+1" }))
hl.bind(mod .. " + mouse_up", hl.dsp.focus({ workspace = "r-1" }))
hl.bind(mod .. " + mouse_down", hl.dsp.focus({ workspace = "r+1" }))
hl.bind(mod .. " + H", hl.dsp.window.move({ workspace = "special", follow = false })) -- esconder ventana
hl.bind(mod .. " + ALT + H", hl.dsp.workspace.toggle_special())                       -- mostrar/ocultar escondidas
hl.gesture({ fingers = 3, direction = "horizontal", action = "workspace" })

-- ── Ratón ──
hl.bind(mod .. " + mouse:272", hl.dsp.window.drag(), { mouse = true })
hl.bind(mod .. " + mouse:273", hl.dsp.window.resize(), { mouse = true })
