# Licencias de terceros

Maxor OS es un conjunto de configuración (MIT, ver [LICENSE](LICENSE)) que **usa** software y
fuentes de terceros sin incluir su código fuente. Cada componente conserva su licencia original;
las que aparecen aquí son las publicadas por sus autores. Antes de redistribuir una imagen o una
ISO, verifica la licencia vigente en el repositorio de cada proyecto.

## Software

| Componente | Uso | Licencia |
|---|---|---|
| [NixOS / nixpkgs](https://github.com/NixOS/nixpkgs) | Sistema base y paquetes | MIT |
| [home-manager](https://github.com/nix-community/home-manager) | Configuración de usuario | MIT |
| [Hyprland](https://github.com/hyprwm/Hyprland) | Compositor | BSD-3-Clause |
| [hyprlock](https://github.com/hyprwm/hyprlock) / [hypridle](https://github.com/hyprwm/hypridle) | Bloqueo e inactividad | BSD-3-Clause |
| [DankMaterialShell](https://github.com/AvengeMedia/DankMaterialShell) | Shell (barra, launcher, notificaciones) | MIT |
| [Quickshell](https://github.com/quickshell-mirror/quickshell) | Base de DMS | LGPL-3.0 |
| [kitty](https://github.com/kovidgoyal/kitty) | Terminal | GPL-3.0 |
| [fish](https://github.com/fish-shell/fish-shell) | Shell de línea de comandos | GPL-2.0 |
| [starship](https://github.com/starship/starship) | Prompt | ISC |
| [Plymouth](https://www.freedesktop.org/wiki/Software/Plymouth/) | Pantalla de arranque | GPL-2.0 |

## Fuentes

| Fuente | Autor | Licencia |
|---|---|---|
| [Figtree](https://github.com/erikdkennedy/figtree) | Erik D. Kennedy | SIL OFL 1.1 |
| [Red Hat Mono](https://github.com/RedHatOfficial/RedHatFont) | Red Hat | SIL OFL 1.1 |
| [Krona One](https://fonts.google.com/specimen/Krona+One) | Yvonne Schüttler | SIL OFL 1.1 |
| [Nerd Fonts Symbols](https://github.com/ryanoasis/nerd-fonts) | Ryan L. McIntyre y colaboradores | MIT |

Krona One y Red Hat Mono se descargan en tiempo de compilación desde el repositorio
[google/fonts](https://github.com/google/fonts) (fijado por commit y verificado por hash); no se
redistribuyen dentro de este repositorio.

## Iconos y cursor

| Tema | Licencia |
|---|---|
| [Papirus](https://github.com/PapirusDevelopmentTeam/papirus-icon-theme) | GPL-3.0 |
| [Bibata](https://github.com/ful1e5/Bibata_Cursor) | GPL-3.0 |

## Contenido propio

Los temas oficiales (paletas y wallpapers generados) se publican bajo CC0-1.0, como indica su
`theme.toml`.

## Inspiración

Ryoku OS, Omarchy y los dotfiles de end-4 sirven de referencia de experiencia. Maxor OS no copia
su código.
