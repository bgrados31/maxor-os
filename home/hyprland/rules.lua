-- Maxor OS: reglas de ventana y de capa.

hl.window_rule({ match = { class = "^(pavucontrol)$" }, float = true })
hl.window_rule({ match = { class = "^(blueman-manager)$" }, float = true })
hl.window_rule({ match = { class = "^(nm-connection-editor)$" }, float = true })
hl.window_rule({ match = { class = "^(xdg-desktop-portal)$" }, float = true })
hl.window_rule({ match = { class = "^(firefox)$", title = "^(Picture-in-Picture)$" }, float = true })
hl.layer_rule({ match = { namespace = "^(quickshell)$" }, no_anim = true })
hl.layer_rule({ match = { namespace = "^dms:.*" }, no_anim = true })

-- Cristal en las capas del shell (barra, launcher, notificaciones) y sin parpadeo al aparecer.
-- Protegido: si tu versión de Hyprland no conoce estas reglas, el resto sigue cargando.
pcall(hl.layer_rule, { match = { namespace = "^dms:.*" }, blur = true, ignore_alpha = 0.3 })
-- Pantalla completa (vídeo, presentaciones) no deja que la pantalla se bloquee.
pcall(hl.window_rule, { match = { class = ".*" }, idle_inhibit = "fullscreen" })
