"""Static checks for the website. Fast, no dependencies; CI runs it.

usage: python3 web/tools/check.py
  - every data-i18n key has its English string, and the dictionary has no leftovers
  - assets/data/themes.json matches themes/ (run tools/sync-themes.py when it does not)
  - the published pages carry no inline script, inline style or event handler (the CSP forbids them)
  - the scripts never write HTML (innerHTML, document.write, eval…): Trusted Types would throw
  - every local file the pages, the CSS and the manifest point at exists
"""
import json
import pathlib
import re
import subprocess
import sys

web = pathlib.Path(__file__).resolve().parents[1]
errors = []
pages = [web / "index.html", web / "404.html"]

# i18n
html = (web / "index.html").read_text()
en = json.loads((web / "assets/i18n/en.json").read_text())
keys = set(re.findall(r'data-i18n="([^"]+)"', html))
for spec in re.findall(r'data-i18n-attr="([^"]+)"', html):
    keys |= {p.split(":")[1].strip() for p in spec.split(";") if ":" in p}
for k in sorted(keys - en.keys()):
    errors.append(f"en.json: missing {k}")
for k in sorted(en.keys() - keys - {"meta.description"}):
    errors.append(f"en.json: unused {k}")

# themes
data = (web / "assets/data/themes.json").read_text()
fresh = subprocess.run([sys.executable, str(web / "tools/sync-themes.py")], capture_output=True, text=True)
if fresh.returncode != 0:
    errors.append(f"sync-themes.py failed: {fresh.stderr.strip()}")
elif (web / "assets/data/themes.json").read_text() != data:
    errors.append("assets/data/themes.json was out of date with themes/ (now regenerated: commit it)")

# CSP hygiene
for page in pages:
    text = page.read_text()
    rel = page.relative_to(web)
    if re.search(r"<script(?![^>]*\bsrc=)[^>]*>", text):
        errors.append(f"{rel}: inline <script>")
    if re.search(r"<style\b|\sstyle=\"", text):
        errors.append(f"{rel}: inline style")
    if re.search(r"\son[a-z]+=\"", text):
        errors.append(f"{rel}: inline event handler")
    if "Content-Security-Policy" not in text:
        errors.append(f"{rel}: no Content-Security-Policy")
for js in sorted((web / "assets/js").glob("*.js")):
    for n, line in enumerate(js.read_text().splitlines(), 1):
        if re.search(r"\.(innerHTML|outerHTML)\s*=|insertAdjacentHTML|document\.write|\beval\(|new Function\(", line):
            errors.append(f"{js.relative_to(web)}:{n}: writes HTML or evaluates code")

# local references
def check_ref(src, ref, base):
    ref = ref.split("#")[0].split("?")[0]
    if not ref or re.match(r"^(https?:|data:|mailto:)", ref):
        return
    target = (web / ref.removeprefix("/maxor-os/")) if ref.startswith("/maxor-os/") else (base / ref)
    if not target.resolve().exists():
        errors.append(f"{src}: missing {ref}")

for page in pages:
    for ref in re.findall(r'(?:href|src)="([^"]+)"', page.read_text()):
        check_ref(page.relative_to(web), ref, page.parent)
css = web / "assets/css/site.css"
for ref in re.findall(r'url\("?([^")]+)"?\)', css.read_text()):
    check_ref(css.relative_to(web), ref, css.parent)
for icon in json.loads((web / "site.webmanifest").read_text())["icons"]:
    check_ref("site.webmanifest", icon["src"], web)

for e in errors:
    print("✗", e)
print("web: ok" if not errors else f"web: {len(errors)} problem(s)")
sys.exit(1 if errors else 0)
