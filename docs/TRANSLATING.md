# Translating Maxor OS

Everything a person reads in Maxor OS can speak their language: the installer, the full-screen app (`maxor-tui`)
and the `maxor` command line. Translations are **standard gettext files (`.po`)**, the format every translation
tool understands, so you can translate in a web browser, in Poedit, or in a text editor. You do not need to know Go
or Nix, and adding a language means adding two files.

| Program | Catalog | Template (the English source) |
|---------|---------|-------------------------------|
| Installer and full-screen app | `tui/internal/i18n/lang/<code>.po` | `tui/internal/i18n/lang/maxor-tui.pot` |
| `maxor` command line | `home/maxor/po/<code>.po` | `home/maxor/po/maxor-cli.pot` |

The full-screen app shows what the command line says (the results of Doctor, for example) and asks it to answer in
its own language, so translate both for a language to feel complete.

## Which language is used

- The installer switches as soon as a language is picked in its first step.
- The app and the command line follow the system language (`LC_ALL`, `LANG`), or `MAXOR_LANG=es` to force one, or
  `maxor --lang es …` for a single command.
- A regional code (`pt_BR`) is loaded on top of its language (`pt`), so it only needs what differs.
- A text that is missing, empty or marked *fuzzy* is shown **in English**, so a partial translation is always usable.

## How to translate

Pick whatever you like. All of them end in a pull request that changes `.po` files.

1. **In the browser (Weblate)**: once the project is online (see *For maintainers*), open it, pick your language,
   translate. No git, no tools. It sends the pull requests for you.
2. **With a PO editor**: Poedit, Lokalize, Gtranslator… open `<code>.po`, translate, save.
3. **With a text editor**: fill the `msgstr` lines.

To add a language, from the repository root:

```
nix shell nixpkgs#gettext nixpkgs#go -c scripts/i18n.sh new <code>     # es, pt, pt_BR, ru, ja…
```

It creates both catalogs with the language's own plural rule, ready to fill in. `scripts/i18n.sh status` shows how
far each language is, and `scripts/i18n.sh check` is what the CI runs.

## Reading a catalog

```po
#. doctor.sys_bad                              ← the id of the text (command line)
msgctxt "doctor.sys_bad"
msgid "%s failed system service: systemctl --failed"
msgid_plural "%s failed system services: systemctl --failed"
msgstr[0] "%s servicio del sistema con fallos: systemctl --failed"
msgstr[1] "%s servicios del sistema con fallos: systemctl --failed"
```

- **Plurals**: a text with a number has one `msgstr[N]` per form your language has. Which number takes which form
  is the `Plural-Forms` rule in the file header (Spanish has 2, Russian 3, Arabic 6, Japanese 1). Write every form.
- **Placeholders** (`%s`, `%d`): keep them. If your language needs another word order, number them:
  `%2$s … %1$s` (the first, the second…). A plain placeholder must keep the English order.
- **Context** (`msgctxt`): in the app it tells two meanings of the same English word apart (a verb and a noun); in the
  command line it is the id of the text. Never change it.
- **Fuzzy** (`#, fuzzy`): "needs a person to look at it". The program ignores fuzzy texts until the flag is removed.
- `%%` is a literal percent sign.

## Rules for good translations

- **Keep it short.** The screens are terminal windows; the tests check every screen at a small size.
- **Do not translate** commands, options, paths, key names you must press (`ctrl+p`, `esc`, `r`), the word the user
  must type to confirm (`INSTALL`, `ERASE`) or brand names (Maxor OS, NVIDIA, Windows, LUKS2). The command words of
  the app's command palette (`go`, `theme`, `search`, `open`) stay English too. The example commands inside the
  help texts must stay exactly as they are: a test checks it.
- **Keep leading and trailing spaces**: they are part of the layout.
- Write like the tools of your language: friendly and direct, with the form you would use with a friend, no jargon
  where a plain word exists.
- The prompt `[y/N]` has its own texts (`ui.yes_no`, `ui.yes_words`): write there the letter and the words that mean
  yes in your language. `y` and `yes` always work.
- A text with two counted things is two short phrases joined by a sentence (`doctor.n_problems` + `doctor.n_warnings`
  + `doctor.sum_bad`), so each plural is right on its own.

## What the tests check

For every language, present and future, with no change to the tests:

- The placeholders are the ones the English text has; positional ones may come in any order, plain ones keep theirs.
  In a counted text a form may leave the number out ("one service"), but may not add a placeholder.
- Each counted text has one form for every form the language has, and the plural rule is valid (the command line
  evaluates it as arithmetic, so only `n`, numbers and operators are accepted).
- The catalog has nothing the code no longer uses; the templates match the code (`scripts/i18n.sh update` fixes it).
- In the command line, the example commands of every help text are exactly as in English.
- The installer, every tab and the help screen fit a small window in each language, and also in a pseudo-language that
  makes every text 40% longer, so a language longer than any we have is covered before it exists.
- Reviewed languages (`reviewed` in `tui/internal/i18n/tools_test.go`, `REVIEWED` in `tests/i18n.bats`) must be
  complete. Others may have gaps; the tests report how many.

## Status of each language

| Code | Language | App | Command line |
|------|----------|-----|--------------|
| `es` | Español | reviewed | reviewed |
| `pt`, `fr`, `de`, `it` | Português, Français, Deutsch, Italiano | **machine translation** | **machine translation** |

Native speakers are very welcome to review a language: say so in your pull request, and we mark it reviewed.

## Not translated on purpose

- The log of the installer's engine and of `nixos-rebuild`, which are for bug reports and stay as the tools print them.
- Keyboard layout and language names, which come from the system's own data (xkeyboard-config, iso-codes).

## Limits

- **Right-to-left languages** (Arabic, Hebrew, Persian): terminals draw text left to right, so the words come out in the
  right order but aligned the wrong way. Translating is possible; a good result depends on the terminal.
- **Fonts**: the system carries Noto for every script and Noto CJK, so Cyrillic, Greek, Arabic, Indic and CJK text
  draws. The brand mark (MAXOR OS) is Latin on purpose.
- **Width**: Chinese, Japanese and Korean characters are two cells wide; the screens measure them that way.

## For maintainers

**Adding or changing a text in the code.** Write it in English, then run `scripts/i18n.sh update` (with gettext and
Go). That refreshes both templates and merges them into every language: new texts appear empty (in the command line,
a text whose English changed comes back as a *fuzzy* suggestion). The CI fails if you forget.

**Weblate** (free hosting for free software: <https://hosted.weblate.org>) is the tool that makes more languages
cheap to add. Once, someone with an account creates the project and two components from this repository:

| Component | File mask | Template (new language base) |
|-----------|-----------|------------------------------|
| Maxor OS · app | `tui/internal/i18n/lang/*.po` | `tui/internal/i18n/lang/maxor-tui.pot` |
| Maxor OS · command line | `home/maxor/po/*.po` | `home/maxor/po/maxor-cli.pot` |

File format *gettext PO file*; add-ons *Update PO files to match POT (msgmerge)*, and *Squash Git commits*; send changes
as pull requests (it opens them against `development`). Turn on its automatic suggestions (machine translation) for
new languages: they arrive as suggestions a person accepts, never as final text. A glossary with the brand and
technical words (generation, flake, profile, theme) keeps them consistent across languages.

**Reviewing a pull request** that touches a `.po`: the CI already checked placeholders, plural forms and layout; read
the texts for tone and for anything that sounds off, and merge.

## For developers

- **Go** (`tui/`): `tr("English text")` in `screens` and `install`, `i18n.T("…")` elsewhere. With arguments it formats
  like `fmt.Sprintf`: `tr("Joining %s", name)`. A counted text: `trn("%d app", "%d apps", n)`. An ambiguous word:
  `trc("verb", "Update")`. Never build a text with `+`: the test refuses it, because a translator needs the whole
  sentence. A text in a package-level table is written `i18n.Mark("…")` and translated where it is shown with
  `tr(text)`. Tests can turn on `i18n.SetPseudo(true)`.
- **Bash** (`home/maxor/`): write `@section.key` where a text goes (`ui_say ok @apps.removing "$name"`) and add the
  English text to `lib/lang/en.sh`. A counted text has a second form, `MSG[key#1]`, and its **first argument is the
  number**: `msg out @doctor.sys_bad "$n"`. Match what a program prints by an **id**, never by its text: the text
  changes with the language (the Doctor checks, for example, come with an `id`).
- Programs the screen reads by their English output (`sudo`, `nixos-rebuild`) run with `LC_MESSAGES=C`.
