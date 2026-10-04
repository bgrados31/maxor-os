load helper

setup() {
  isolate
  mkdir -p "$BATS_TEST_TMPDIR/r/scripts"
  cp "$ROOT/scripts/release-notes.sh" "$BATS_TEST_TMPDIR/r/scripts/"
  cat > "$BATS_TEST_TMPDIR/r/CHANGELOG.md" <<'MD'
# Cambios

## [Sin publicar]

### Añadido

- algo futuro

## [0.2.0] - 2026-10-09

### Añadido

- **Cosa nueva**: descripción.

### Corregido

- Un arreglo.

## [0.1.0] - 2026-10-03

### Añadido

- Lo primero.

## Historial previo

- viejo
MD
}

@test "release-notes extrae solo la sección de la versión" {
  run "$BATS_TEST_TMPDIR/r/scripts/release-notes.sh" 0.2.0
  [ "$status" = 0 ]
  [[ "$output" == *"Cosa nueva"* ]]
  [[ "$output" == *"Un arreglo"* ]]
  [[ "$output" != *"Lo primero"* ]]
  [[ "$output" != *"algo futuro"* ]]
}

@test "release-notes no incluye líneas en blanco sobrantes" {
  out="$("$BATS_TEST_TMPDIR/r/scripts/release-notes.sh" 0.1.0)"
  [ "${out:0:3}" = "###" ]
  [ "${out: -1}" != $'\n' ]
}

@test "release-notes falla si la versión no existe" {
  run "$BATS_TEST_TMPDIR/r/scripts/release-notes.sh" 9.9.9
  [ "$status" = 1 ]
}

@test "release-notes se detiene en el historial previo" {
  out="$("$BATS_TEST_TMPDIR/r/scripts/release-notes.sh" 0.1.0)"
  [[ "$out" != *viejo* ]]
}

@test "release.sh rechaza una versión mal escrita" {
  run "$ROOT/scripts/release.sh" 1.0
  [ "$status" = 1 ]
  [[ "$output" == *"X.Y.Z"* ]]
}
