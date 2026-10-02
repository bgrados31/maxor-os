# Solución de problemas

## No puedo desbloquear la pantalla

hyprlock autentica con PAM. La configuración declara el servicio en
`security.pam.services.hyprlock`; si lo quitaste, no podrás desbloquear.

1. Cambia a una TTY con `Ctrl+Alt+F3` e inicia sesión.
2. Ejecuta `pkill hyprlock`.
3. Vuelve a la sesión con `Ctrl+Alt+F1` o `F2`.

Para evitarlo en el futuro, no elimines esa línea de `hosts/<equipo>/configuration.nix`.

## `maxor theme apply` no cambia nada

El comando actualiza dos claves de `~/.config/DankMaterialShell/settings.json`. Comprueba que se
escribieron:

```sh
jq '{currentThemeName, customThemeFile}' ~/.config/DankMaterialShell/settings.json
```

Debe mostrar `"custom"` y una ruta dentro de `~/.config/maxor/current/`. Si es así y la barra no
cambia, reinicia DMS con `systemctl --user restart dms`. Si `jq` da error, el archivo no existe
todavía: abre antes los ajustes de DMS (`SUPER + ,`).

## Un tema da error de «colores inválidos» o «incompleto»

`colors.json` necesita las ocho claves (`bg`, `s`, `s2`, `fg`, `mu`, `ac`, `ac2`, `on`) y cada
valor debe ser `#rrggbb` (seis dígitos hexadecimales). Ver [THEMING.md](THEMING.md).

## El sistema no arranca tras un `switch`

Elige una generación anterior en el menú de arranque; NixOS conserva las últimas diez
(`configurationLimit = 10`). Una vez dentro:

```sh
sudo nixos-rebuild switch --rollback
```

## El menú de arranque no muestra Windows

Windows aparece porque systemd-boot detecta su gestor de arranque en la ESP. Comprueba que la ESP
sigue montada en `/efi`:

```sh
findmnt /efi
sudo bootctl list
```

## NVIDIA: una app no usa la GPU dedicada

PRIME está configurado en modo *offload*: la gráfica integrada dibuja el escritorio y la NVIDIA
solo se usa cuando se pide.

```sh
nvidia-offload <programa>        # en Steam: nvidia-offload %command%
```

Para comprobar qué GPU usa un programa: `nvidia-offload glxinfo | grep "OpenGL renderer"`.

## El splash de arranque no aparece

Plymouth solo se muestra si el kernel arranca con `quiet splash` (ya incluido) y la fuente del
tema está en el initrd. Tras cambiar el tema o la fuente hace falta un `switch` y reiniciar;
Plymouth no se puede previsualizar en una sesión ya iniciada.

## Errores en la configuración de Hyprland

Hyprland 0.55 usa configuración en Lua. Los errores salen en pantalla al iniciar o al recargar
(`hyprctl reload`). Para ver el detalle:

```sh
journalctl --user -b | grep -i hyprland
```

## Las apps Qt no usan los colores del tema

Abre `qt6ct` una vez y elige el esquema de colores de DMS.

## Informar de un fallo

Abre un [issue](https://github.com/bgrados31/maxor-os/issues/new/choose) con la salida de:

```sh
nixos-version
hyprctl version | head -n 3
journalctl --user -b -p err --no-pager | tail -n 30
```
