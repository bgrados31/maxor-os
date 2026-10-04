{ lib, runCommand, python3, glibcLocales, isocodes, xkeyboard-config }:

# The lists the installer offers, taken from the system's own data instead of written by hand:
#   locales   glibc's list of supported locales, named with iso-codes (language and country)
#   layouts   every keyboard layout and variant of xkeyboard-config, with its description
# Timezones are read at run time (timedatectl). The result is one JSON file the installer loads; without it the
# installer falls back to its short built-in lists. Checked by checks.catalog in flake.nix.
runCommand "maxor-catalog.json" { nativeBuildInputs = [ python3 ]; } ''
  python3 - <<'PY'
  import json, os, re, xml.etree.ElementTree as ET

  codes = "${isocodes}/share/iso-codes/json"
  langs = {}
  for l in json.load(open(codes + "/iso_639-3.json"))["639-3"]:
      name = l.get("common_name") or l["name"]
      langs[l["alpha_3"]] = name
      if "alpha_2" in l:
          langs[l["alpha_2"]] = name
  for l in json.load(open(codes + "/iso_639-2.json"))["639-2"]:
      langs.setdefault(l["alpha_3"], (l.get("common_name") or l["name"]).split(";")[0])
  countries = {c["alpha_2"]: (c.get("common_name") or c["name"]) for c in json.load(open(codes + "/iso_3166-1.json"))["3166-1"]}

  locales, seen = [], set()
  for line in open("${glibcLocales}/share/i18n/SUPPORTED"):
      m = re.match(r"^([A-Za-z_]+)(?:\.UTF-8)?(@\w+)?/UTF-8", line.strip())
      if not m:
          continue
      base, mod = m.group(1), m.group(2) or ""
      code = base + ".UTF-8" + mod
      if code in seen or "_" not in base:
          continue
      seen.add(code)
      lang, terr = base.split("_", 1)
      name = langs.get(lang, lang)
      name += " (" + countries.get(terr, terr) + (", " + mod[1:] if mod else "") + ")"
      locales.append({"code": code, "name": name})
  locales.sort(key=lambda x: x["name"])

  layouts = []
  root = ET.parse("${xkeyboard-config}/share/X11/xkb/rules/evdev.xml").getroot()
  for lay in root.iter("layout"):
      ci = lay.find("configItem")
      xkb, desc = ci.find("name").text, ci.find("description").text
      layouts.append({"xkb": xkb, "variant": "", "name": desc})
      for v in lay.iter("variant"):
          vi = v.find("configItem")
          layouts.append({"xkb": xkb, "variant": vi.find("name").text, "name": vi.find("description").text})
  layouts.sort(key=lambda x: (x["name"], x["variant"]))

  json.dump({"version": 1, "locales": locales, "layouts": layouts}, open(os.environ["out"], "w"), ensure_ascii=False)
  PY
''
