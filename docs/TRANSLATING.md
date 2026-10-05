# Translating Maxor OS

Everything a person reads in Maxor OS can speak their language: the installer, the full-screen app (`maxor-tui`)
and the `maxor` command line. A translation is a plain text file, and you do not need to know Go or Nix to write one.

There are two catalogs, because the two programs are written in different languages. The rules are the same.

| What | Catalog | How a text is found |
|------|---------|---------------------|
| Installer and full-screen app | `tui/internal/i18n/lang/<code>.json` | the English text is the key |
| `maxor` command line | `home/maxor/lib/lang/<code>.sh` | a stable id (`doctor.sys_ok`) |

The full-screen app shows what the command line says (the results of Doctor, for example), and it asks the command
line to answer in its own language, so translate both for a language to feel complete.

## Which language is used

- The installer switches as soon as a language is picked in its first step.
- The app and the command line follow the system language (`LC_ALL`, `LANG`), or `MAXOR_LANG=es` to force one, or
  `maxor --lang es …` for a single command.
- A regional code (`pt_BR`) is loaded on top of its language (`pt`), so it only needs what differs.
- A text that is missing is shown **in English**, so a partial translation is always usable.

## Add or improve a language

Fork the repository and send one language per pull request.

### The installer and the app

1. Create `tui/internal/i18n/lang/<code>.json` with `{}` inside (or open the existing one).
2. From `tui/`, add every text of the program to every catalog with an empty value (`""` = not translated yet):

   ```
   CGO_ENABLED=0 nix shell nixpkgs#go -c go test ./internal/i18n -update
   ```

3. Fill in the empty values of your file:

   ```json
   {
     "Keyboard": "Teclado",
     "Connected to the internet (%s)": "Conectado a internet (%s)"
   }
   ```

4. Check it: `CGO_ENABLED=0 nix shell nixpkgs#go -c go test ./internal/i18n ./internal/app`.

### The command line

1. Create the skeleton, with every English message commented out:

   ```
   scripts/i18n.sh new <code>          # es, pt, pt_BR, fr…
   ```

2. Uncomment each message and translate it:

   ```bash
   msgs_es() {
     MSG[doctor.sys_ok]='ningún servicio del sistema ha fallado'
     MSG[doctor.sys_bad]='%s servicio(s) del sistema con fallos: systemctl --failed'
   }
   ```

   Single quotes. If the text has an apostrophe or a line break, use `$'…'` and write `\'` and `\n`
   (the help texts do this). Do not use typographic apostrophes (`’`): the build rejects them.

3. See what is left, and check it:

   ```
   scripts/i18n.sh missing <code>
   scripts/i18n.sh status
   nix shell nixpkgs#bats nixpkgs#jq nixpkgs#openssl nixpkgs#openssh nixpkgs#util-linux nixpkgs#ncurses -c bats tests/i18n.bats
   ```

   Nothing else has to be registered: the package picks up every file in `lib/lang/`.

## What the tests check

For every language, present and future, with no change to the tests:

- The placeholders (`%s`, `%d`, `%.1f`) are the same as in the English text, **in the same order**. A wrong one would
  garble the text or crash it. (`printf` cannot reorder arguments; write the sentence so the order works.)
- The catalog has no text the code does not have (`-update` removes stale ones in the app catalogs).
- In the command line, every example command of a help text (`    maxor theme apply …`) is exactly as in English.
- Every message formats without error.
- The installer, every tab and the help screen of the app fit a small window in each language, because translations
  are longer than English.
- Reviewed languages (`reviewed` in `tui/internal/i18n/i18n_test.go` and `REVIEWED` in `tests/i18n.bats`) must be
  complete. Others may have gaps; the tests report how many.

## Rules for good translations

- **Keep it short.** The screens are terminal windows.
- **Do not translate** commands, options, paths, key names you must press (`ctrl+p`, `esc`, `r`), the word the user
  must type to confirm (`INSTALL`, `ERASE`) or brand names (Maxor OS, NVIDIA, Windows, LUKS2). The command words of
  the app's command palette (`go`, `theme`, `search`, `open`) stay English too.
- **Keep the placeholders** and the leading and trailing spaces: they are part of the layout.
- A plural is written for both cases (`servicio(s)`, `%s problema(s)`) in the command line; the app passes the
  singular and the plural words separately (`warning` / `warnings`), translate both.
- Write like the tools of your language: friendly and direct, with the form you would use with a friend, no jargon
  where a plain word exists.
- Prompts that read an answer (`[y/N]`) have their own keys (`ui.yes_no`, `ui.yes_words`): put there the letter and
  the words that mean yes in your language. `y` and `yes` always work.

## Status of each language

| Code | Language | App | Command line |
|------|----------|-----|--------------|
| `es` | Español | reviewed | reviewed |
| `pt`, `fr`, `de`, `it` | Português, Français, Deutsch, Italiano | **machine translation** | **machine translation** |

Native speakers are very welcome to review a language: say so in your pull request, and we mark it reviewed.

## Not translated on purpose

- The log of the installer's engine and of `nixos-rebuild`, which are for bug reports and stay as the tools print them.
- Keyboard layout and language names, which come from the system's own data (xkeyboard-config, iso-codes).

## For developers

- **Go** (`tui/`): write `tr("English text")` in `screens` and `install`, `i18n.T("…")` elsewhere. With arguments it
  formats like `fmt.Sprintf`: `tr("Joining %s", name)`. Never build a text with `+`: the test refuses it, because a
  translator needs the whole sentence. A text in a package-level table is written `i18n.Mark("…")` and translated
  where it is shown with `tr(text)`. After adding or changing texts, run the `-update` command above.
- **Bash** (`home/maxor/`): write `@section.key` where a text goes (`ui_say ok @apps.removing "$name"`), and add the
  English text to `lib/lang/en.sh`. Match what the program prints by an **id**, never by its text: the text changes
  with the language (the Doctor checks, for example, come with an `id`).
- Programs the screen reads by their English output (`sudo`, `nixos-rebuild`) run with `LC_MESSAGES=C`.
