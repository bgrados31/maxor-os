"""Copies the official themes (themes/*/colors.json, name, gradient.json) into the website's data.

usage: python3 web/tools/sync-themes.py   (from the repository root)
The site never invents a colour: the live desktop on the landing page is painted with these values.
"""
import json
import pathlib
import re

root = pathlib.Path(__file__).resolve().parents[2]
order = ["maxor-dark", "maxor-light", "sakura", "glacier", "obsidian", "ember", "ultraviolet",
         "dawn", "frost", "paper", "breeze", "amber"]
themes = []
for tid in sorted((p.name for p in (root / "themes").iterdir() if p.is_dir()),
                  key=lambda t: (order.index(t) if t in order else len(order), t)):
    d = root / "themes" / tid
    colors = json.loads((d / "colors.json").read_text())
    name = re.search(r'^name\s*=\s*"([^"]+)"', (d / "theme.toml").read_text(), re.M).group(1)
    grad = d / "gradient.json"
    themes.append({"id": tid, "name": name, **colors,
                   "gradient": json.loads(grad.read_text()) if grad.exists() else None})
out = root / "web/assets/data/themes.json"
out.write_text(json.dumps(themes, indent=1) + "\n")
print(f"{len(themes)} themes → {out.relative_to(root)}")
