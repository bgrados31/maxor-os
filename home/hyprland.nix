{ ... }:

{
  wayland.windowManager.hyprland = {
    enable = true;
    configType = "lua"; # Hyprland 0.55+ usa Lua
    # Los paquetes vienen de programs.hyprland (NixOS)
    package = null;
    portalPackage = null;
    systemd.variables = [ "--all" ];

    extraConfig = ''
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

      hl.window_rule({ match = { class = "^(pavucontrol)$" }, float = true })
      hl.window_rule({ match = { class = "^(blueman-manager)$" }, float = true })
      hl.window_rule({ match = { class = "^(nm-connection-editor)$" }, float = true })
      hl.window_rule({ match = { class = "^(xdg-desktop-portal)$" }, float = true })
      hl.window_rule({ match = { class = "^(firefox)$", title = "^(Picture-in-Picture)$" }, float = true })
      hl.layer_rule({ match = { namespace = "^(quickshell)$" }, no_anim = true })
      hl.layer_rule({ match = { namespace = "^dms:.*" }, no_anim = true })

      -- ── Apps ──
      hl.bind("SUPER + Return", hl.dsp.exec_cmd("kitty"))
      hl.bind("SUPER + T", hl.dsp.exec_cmd("kitty"))
      hl.bind("SUPER + E", hl.dsp.exec_cmd("nautilus"))
      hl.bind("SUPER + B", hl.dsp.exec_cmd("firefox"))

      -- ── DankMaterialShell ──
      hl.bind("SUPER + space", hl.dsp.exec_cmd("dms ipc call spotlight toggle"))
      hl.bind("SUPER + D", hl.dsp.exec_cmd("dms ipc call spotlight toggle"))
      hl.bind("SUPER + V", hl.dsp.exec_cmd("dms ipc call clipboard toggle"))
      hl.bind("SUPER + M", hl.dsp.exec_cmd("dms ipc call processlist focusOrToggle"))
      hl.bind("SUPER + comma", hl.dsp.exec_cmd("dms ipc call settings focusOrToggle"))
      hl.bind("SUPER + N", hl.dsp.exec_cmd("dms ipc call notifications toggle"))
      hl.bind("SUPER + Y", hl.dsp.exec_cmd("dms ipc call dash toggle wallpaper"))
      hl.bind("SUPER + TAB", hl.dsp.exec_cmd("dms ipc call hypr toggleOverview"))
      hl.bind("SUPER + X", hl.dsp.exec_cmd("dms ipc call powermenu toggle"))
      hl.bind("SUPER + SHIFT + Slash", hl.dsp.exec_cmd("dms ipc call keybinds toggle hyprland"))
      hl.bind("SUPER + ALT + L", hl.dsp.exec_cmd("loginctl lock-session"))
      hl.bind("SUPER + SHIFT + E", hl.dsp.exit())

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

      -- ── Capturas ──
      hl.bind("Print", hl.dsp.exec_cmd("dms screenshot"))
      hl.bind("CTRL + Print", hl.dsp.exec_cmd("dms screenshot full"))
      hl.bind("ALT + Print", hl.dsp.exec_cmd("dms screenshot window"))
      hl.bind("SUPER + SHIFT + S", hl.dsp.exec_cmd("dms screenshot"))

      -- ── Ventanas ──
      hl.bind("SUPER + Q", hl.dsp.window.close())
      hl.bind("SUPER + F", hl.dsp.window.fullscreen({ mode = "maximized", action = "toggle" }))
      hl.bind("SUPER + SHIFT + F", hl.dsp.window.fullscreen({ mode = "fullscreen", action = "toggle" }))
      hl.bind("SUPER + SHIFT + Space", hl.dsp.window.float({ action = "toggle" }))
      hl.bind("SUPER + R", hl.dsp.layout("togglesplit"))
      hl.bind("SUPER + W", hl.dsp.group.toggle())

      hl.bind("SUPER + left", hl.dsp.focus({ direction = "l" }))
      hl.bind("SUPER + down", hl.dsp.focus({ direction = "d" }))
      hl.bind("SUPER + up", hl.dsp.focus({ direction = "u" }))
      hl.bind("SUPER + right", hl.dsp.focus({ direction = "r" }))
      hl.bind("SUPER + H", hl.dsp.focus({ direction = "l" }))
      hl.bind("SUPER + J", hl.dsp.focus({ direction = "d" }))
      hl.bind("SUPER + K", hl.dsp.focus({ direction = "u" }))
      hl.bind("SUPER + L", hl.dsp.focus({ direction = "r" }))

      hl.bind("SUPER + SHIFT + left", hl.dsp.window.move({ direction = "l" }))
      hl.bind("SUPER + SHIFT + down", hl.dsp.window.move({ direction = "d" }))
      hl.bind("SUPER + SHIFT + up", hl.dsp.window.move({ direction = "u" }))
      hl.bind("SUPER + SHIFT + right", hl.dsp.window.move({ direction = "r" }))
      hl.bind("SUPER + SHIFT + H", hl.dsp.window.move({ direction = "l" }))
      hl.bind("SUPER + SHIFT + J", hl.dsp.window.move({ direction = "d" }))
      hl.bind("SUPER + SHIFT + K", hl.dsp.window.move({ direction = "u" }))
      hl.bind("SUPER + SHIFT + L", hl.dsp.window.move({ direction = "r" }))

      hl.bind("SUPER + minus", hl.dsp.window.resize({ x = -100, y = 0, relative = true }), { repeating = true })
      hl.bind("SUPER + equal", hl.dsp.window.resize({ x = 100, y = 0, relative = true }), { repeating = true })
      hl.bind("SUPER + SHIFT + minus", hl.dsp.window.resize({ x = 0, y = -100, relative = true }), { repeating = true })
      hl.bind("SUPER + SHIFT + equal", hl.dsp.window.resize({ x = 0, y = 100, relative = true }), { repeating = true })
      hl.bind("SUPER + mouse:272", hl.dsp.window.drag(), { mouse = true })
      hl.bind("SUPER + mouse:273", hl.dsp.window.resize(), { mouse = true })

      -- ── Workspaces ──
      for i = 1, 9 do
        hl.bind("SUPER + " .. i, hl.dsp.focus({ workspace = tostring(i) }))
        hl.bind("SUPER + SHIFT + " .. i, hl.dsp.window.move({ workspace = tostring(i) }))
      end
      hl.bind("SUPER + mouse_down", hl.dsp.focus({ workspace = "e+1" }))
      hl.bind("SUPER + mouse_up", hl.dsp.focus({ workspace = "e-1" }))
      hl.bind("SUPER + S", hl.dsp.workspace.toggle_special())
      hl.bind("SUPER + CTRL + S", hl.dsp.window.move({ workspace = "special", follow = false }))
      hl.gesture({ fingers = 3, direction = "horizontal", action = "workspace" })

      -- ── Archivos gestionados por DMS (colores del tema, monitores, etc.) ──
      require("dms.colors")
      require("dms.outputs")
      require("dms.layout")
      require("dms.cursor")
      require("dms.binds-user")
      require("dms.windowrules")
    '';
  };
}
