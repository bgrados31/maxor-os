# Política de seguridad

## Versiones con soporte

Maxor OS está en desarrollo activo y no tiene versiones estables. Solo la rama `main` recibe
correcciones.

## Cómo informar de una vulnerabilidad

Usa el [reporte privado de GitHub](https://github.com/bgrados31/maxor-os/security/advisories/new).
No abras un issue público ni publiques detalles antes de que haya una corrección.

Incluye, si puedes:

- qué componente afecta (CLI `maxor`, motor de temas, configuración del sistema);
- cómo reproducirlo;
- el impacto que crees que tiene.

Se acusará recibo en un plazo razonable y se te mantendrá al tanto de la corrección.

## Qué se considera en alcance

- El CLI `maxor` y el motor de temas: por ejemplo, que un tema pueda ejecutar código, escribir
  fuera de las rutas permitidas o inyectar contenido en archivos de configuración.
- Configuraciones del sistema que debiliten la seguridad de forma no evidente.

Fuera de alcance: vulnerabilidades de componentes de terceros (Hyprland, DMS, nixpkgs), que deben
informarse a sus proyectos.

## Modelo de seguridad de los temas

Un tema solo aporta datos. El motor valida que cada color sea `#rrggbb` antes de usarlo, no
ejecuta ningún archivo del tema y solo escribe en `~/.config/maxor/current/`,
`~/.local/state/maxor/` y dos claves de la configuración de DMS. Detalles en
[docs/THEMING.md](docs/THEMING.md#seguridad).
