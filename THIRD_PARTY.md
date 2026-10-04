# Third-party licenses

Maxor OS is a set of configuration (MIT, see [LICENSE](LICENSE)) that **uses** third-party
software and fonts without including their source code. Each component keeps its original
license; the ones listed here are those published by their authors. Before redistributing an
image or an ISO, check the current license in each project's repository.

## Software

| Component | Use | License |
|---|---|---|
| [NixOS / nixpkgs](https://github.com/NixOS/nixpkgs) | Base system and packages | MIT |
| [home-manager](https://github.com/nix-community/home-manager) | User configuration | MIT |
| [Hyprland](https://github.com/hyprwm/Hyprland) | Compositor | BSD-3-Clause |
| [hyprlock](https://github.com/hyprwm/hyprlock) / [hypridle](https://github.com/hyprwm/hypridle) | Lock screen and idle handling | BSD-3-Clause |
| [DankMaterialShell](https://github.com/AvengeMedia/DankMaterialShell) | Base of Maxor Shell (bar, launcher, notifications, login). Used without copying its code: Maxor Shell extends it with branding patches | MIT |
| [greetd](https://git.sr.ht/~kennylevinsen/greetd) | Login manager | GPL-3.0 |
| [Quickshell](https://github.com/quickshell-mirror/quickshell) | Base of DMS | LGPL-3.0 |
| [kitty](https://github.com/kovidgoyal/kitty) | Terminal | GPL-3.0 |
| [fish](https://github.com/fish-shell/fish-shell) | Command-line shell | GPL-2.0 |
| [starship](https://github.com/starship/starship) | Prompt | ISC |
| [Plymouth](https://www.freedesktop.org/wiki/Software/Plymouth/) | Boot screen | GPL-2.0 |
| [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Lip Gloss](https://github.com/charmbracelet/lipgloss) | The full-screen app (`maxor-tui`) | MIT |

## Fonts

| Font | Author | License |
|---|---|---|
| [Figtree](https://github.com/erikdkennedy/figtree) | Erik D. Kennedy | SIL OFL 1.1 |
| [Red Hat Mono](https://github.com/RedHatOfficial/RedHatFont) | Red Hat | SIL OFL 1.1 |
| [Krona One](https://fonts.google.com/specimen/Krona+One) | Yvonne Schüttler | SIL OFL 1.1 |
| [Nerd Fonts Symbols](https://github.com/ryanoasis/nerd-fonts) | Ryan L. McIntyre and contributors | MIT |

Krona One and Red Hat Mono are downloaded at build time from the
[google/fonts](https://github.com/google/fonts) repository (pinned by commit and verified by
hash); they are not redistributed inside this repository.

## Icons and cursor

| Theme | License |
|---|---|
| [Papirus](https://github.com/PapirusDevelopmentTeam/papirus-icon-theme) | GPL-3.0 |
| [Bibata](https://github.com/ful1e5/Bibata_Cursor) | GPL-3.0 |

## Our own content

The official themes (palettes and generated wallpapers) are published under CC0-1.0, as their
`theme.toml` states.

## Inspiration

Ryoku OS, Omarchy and end-4's dotfiles serve as an experience reference. Maxor OS does not copy
their code.
