# Ramas y releases

## Ramas

| Rama | Para qué |
|---|---|
| `development` | El trabajo diario. Es la rama predeterminada: aquí llegan los commits y las fusiones. |
| `main` | Solo versiones publicadas. Nadie hace commits aquí; se mueve únicamente con un release. |
| `feat/…`, `fix/…` | Cambios que tardan o son arriesgados. Salen de `development` y vuelven con una fusión o un Pull Request. |

Un cambio pequeño puede ir directo a `development`. Lo que toque el arranque, el login o los
gráficos va en una rama aparte y se prueba antes de fusionar.

## Versiones

Numeración `0.N.0` hasta la 1.0, que llegará con el instalador. Mientras sea `0.x`, un cambio
incompatible sube el segundo número; un arreglo o una mejora pequeña sube el tercero.

## Commits y changelog

Los commits siguen [Conventional Commits](https://www.conventionalcommits.org/es/v1.0.0/)
(`feat:`, `fix:`, `perf:`, `docs:`…), lo que permite generar un borrador del changelog.
`CHANGELOG.md` sigue el formato de «Keep a Changelog»: lo que aún no salió va en
`## [Sin publicar]`, agrupado en Añadido, Cambiado y Corregido, escrito para quien usa Maxor y no
para quien lee el código.

## Sacar una versión

```
scripts/release.sh 0.1.0          muestra el plan y no cambia nada
scripts/release.sh 0.1.0 --yes    lo ejecuta
```

Hay que estar en `development`, sin cambios pendientes, al día con `origin` y con algo en
«Sin publicar». El script entonces:

1. comprueba el flake (`nix flake check --no-build` y la evaluación de `nitro`);
2. mueve «Sin publicar» a la sección `## [0.1.0] - fecha`, escribe `VERSION` (que `maxor --version` lee al compilar) y lo confirma;
3. fusiona `development` en `main` (sin avance rápido, para que la versión quede marcada);
4. crea la etiqueta anotada `v0.1.0` y sube `main` y la etiqueta;
5. avanza `development` hasta `main` y abre la siguiente versión menor (`VERSION` pasa a `0.2.0-dev`) en un commit propio.

Al llegar la etiqueta, la CI (`.github/workflows/release.yml`) crea la **GitHub Release** con el
texto de esa sección. Así los cambios de cada versión se leen en la pestaña de Releases.
`scripts/release-notes.sh 0.1.0` imprime esas notas en local para revisarlas antes.

## Si algo sale mal

- **El plan se niega a seguir:** el mensaje dice qué falta (rama, cambios sin commit, changelog vacío).
- **Se subió una versión equivocada:** se borra la etiqueta y la Release en GitHub, se corrige
  en `development` y se publica la siguiente versión. Las versiones no se reutilizan.
- **El repositorio es privado.** Las Releases también lo son; publicarlo es una decisión aparte.
