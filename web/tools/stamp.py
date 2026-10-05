"""Stamps a version on every local CSS/JS/JSON reference of a built copy of the site, so a deploy never
mixes files: GitHub Pages lets browsers cache each file for 10 minutes, and a new main.js importing an
old themes.js would break the page. The sources stay clean; only the copy that is published changes.

usage: python3 web/tools/stamp.py <site dir>   (pages.yml runs it on _site)
"""
import hashlib
import pathlib
import re
import sys

site = pathlib.Path(sys.argv[1])
files = sorted(p for p in site.rglob("*") if p.is_file())
digest = hashlib.sha256()
for p in files:
    digest.update(str(p.relative_to(site)).encode())
    digest.update(p.read_bytes())
v = digest.hexdigest()[:10]

local = r'(?!https?:|data:|//)([^"\'?#]+\.(?:css|js|json|webmanifest))'
patterns = {
    ".html": re.compile(r'((?:href|src)=")' + local + r'(")'),
    ".js": re.compile(r'((?:from |import\(|new URL\()\s*")' + local + r'(")'),
}
count = 0
for p in files:
    rx = patterns.get(p.suffix)
    if not rx or "vendor" in p.parts:
        continue
    text = p.read_text()
    new, n = rx.subn(lambda m: f"{m.group(1)}{m.group(2)}?v={v}{m.group(3)}", text)
    if n:
        p.write_text(new)
        count += n
print(f"stamped {count} references with v={v}")
