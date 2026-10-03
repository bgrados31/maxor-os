# Cómo contribuir

Gracias por querer mejorar Maxor OS. Este documento explica cómo proponer cambios.

## Antes de empezar

- Lee la [arquitectura](docs/ARCHITECTURE.md) y la [hoja de ruta](docs/ROADMAP.md) para ver
  hacia dónde va el proyecto.
- Para algo grande, abre primero un issue y describe la idea; evita trabajo perdido.
- Sigue el [código de conducta](CODE_OF_CONDUCT.md).

## Entorno

Necesitas NixOS (o Nix con flakes) y Git.

```sh
git clone https://github.com/bgrados31/maxor-os
cd maxor-os
nix flake check --no-build          # evalúa la configuración
nix build .#nixosConfigurations.nitro.config.system.build.toplevel   # compila el sistema
```

`nixos-rebuild switch` solo debe hacerse en una máquina de pruebas o una VM.

## Flujo de trabajo

1. Haz un fork y crea una rama desde `main`: `feat/mi-cambio` o `fix/mi-arreglo`.
2. Haz cambios pequeños y enfocados. Un cambio, un propósito.
3. Comprueba que `nix flake check --no-build` pasa y que el sistema compila.
4. Abre un pull request y completa la plantilla.

## Mensajes de commit

Usamos [Conventional Commits](https://www.conventionalcommits.org/es/v1.0.0/), en español o
inglés, en modo imperativo y sin punto final.

```
feat(theming): añadir tema Obsidiana
fix(lockscreen): usar la plantilla del tema activo al bloquear
docs: explicar cómo adaptar el host a otra GPU
```

Tipos habituales: `feat`, `fix`, `docs`, `refactor`, `chore`, `ci`.

## Estilo

- **Nix:** indentación de dos espacios, un comentario breve sobre el *porqué* de cada bloque
  no obvio y ningún valor específico de una máquina fuera de `hosts/`.
- **Shell (CLI `maxor`):** debe pasar `shellcheck` (lo ejecuta `writeShellApplication`).
- **Lua (Hyprland):** una responsabilidad por archivo cuando se modularice.
- **Documentación:** en español, directa, con ejemplos que se puedan copiar y pegar.

## Contribuir un tema

Un tema es una carpeta con `colors.json`, `theme.toml` y, opcionalmente, `style.json`; el formato está en
[docs/THEMING.md](docs/THEMING.md). Los oficiales son las carpetas de `themes/`: añade una carpeta
y el build se encarga del resto.

Requisitos:

- Los ocho colores de la paleta, en `#rrggbb`.
- Contraste WCAG AA (4.5:1 o más) entre `fg` y `bg`, `mu` y `bg`, `ac` y `s`, y `on` y `ac`.
- Una licencia que permita redistribuirlo (se recomienda CC0-1.0).
- Un tema nunca incluye scripts ni ejecutables.

## Seguridad

No abras un issue público por una vulnerabilidad. Sigue [SECURITY.md](SECURITY.md).
