# Translating Maxor OS

The installer (and, soon, the rest of `maxor-tui`) can speak your language. You do not need to know Go:
a translation is one JSON file.

## How it works

The English text is the key. A catalog maps each English text to its translation:

```json
{
  "Keyboard": "Teclado",
  "Connected to the internet (%s)": "Conectado a internet (%s)"
}
```

- Catalogs live in `tui/internal/i18n/lang/<code>.json` (`es`, `pt`, `fr`…). A regional one (`pt_BR`) wins over
  the language one (`pt`) when both exist.
- A text a catalog leaves empty or does not have is shown **in English**, so a partial translation is fine.
- When someone picks a language in the first step of the installer, the rest of the installer switches to it.

## Add or improve a language

1. Fork the repository and create `tui/internal/i18n/lang/<code>.json` with `{}` inside (or open the existing one).
2. Fill it with the texts: from `tui/`, run

   ```
   CGO_ENABLED=0 nix shell nixpkgs#go -c go test ./internal/i18n -update
   ```

   It adds every text of the program to every catalog with an empty value (`""` = not translated yet) and removes
   the ones the code no longer uses. Fill in the empty values of your file.
3. Check it:

   ```
   CGO_ENABLED=0 nix shell nixpkgs#go -c go test ./internal/i18n ./internal/app
   ```

   The tests fail if a placeholder (`%s`, `%d`, `%.1f`…) is not exactly as in the English text and in the same
   order, or if the catalog has a text the code does not have. They also report how many texts are missing.
4. Open a pull request. One language per pull request.

## Rules for good translations

- **Keep the placeholders** (`%s`, `%d`…) and move them where your grammar needs them only if the order of
  *different* ones stays the same (the test checks it).
- **Keep it short.** The installer must fit in a small terminal; the tests run every step in a small window in
  every language and fail if one overflows.
- **Do not translate** commands, paths, key names you must press as typed (`ctrl+p`, `esc`, `r`) when the word is
  literal, or the word the user must type to confirm (`INSTALL`, `ERASE`): those are the same in every language.
  Brand names (Maxor OS, NVIDIA, Windows, LUKS2) stay as they are.
- Leading and trailing spaces in a text are part of the layout: keep them.
- Use the register of the language's own tools: friendly, direct, no jargon where a plain word exists.

## Status of each language

| Code | Language | Status |
|------|----------|--------|
| `es` | Español | reviewed |
| `pt`, `fr`, `de`, `it` | Português, Français, Deutsch, Italiano | **machine translation**: native speakers, please review |

Only `es` is required to be complete by the tests. If you review a language, say so in your pull request and we
add it to the list of reviewed ones in `internal/i18n/i18n_test.go`.

## For developers

In Go, write `tr("English text")` (package `screens` and `install`) or `i18n.T("…")` elsewhere. With arguments it
formats like `fmt.Sprintf`: `tr("Joining %s", name)`. Never build a text with `+`: the test refuses it, because a
translator needs the whole sentence. A text defined in a package-level table is written `i18n.Mark("…")` and
translated where it is shown with `tr(text)`. After adding or changing texts, run the `-update` command above.
