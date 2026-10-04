#!/usr/bin/env python3
"""What would installing this system need that a medium does not carry?

    offline-missing.py TARGET.drv ROOT [ROOT...]

TARGET.drv is the derivation of the system to install; each ROOT is a store path the medium carries (its whole closure
counts). The script walks the derivation graph of the target and reports the derivations whose outputs are not in that
closure, because Nix would have to build them, and, among those, the ones that would have to fetch something from the
internet (fixed-output derivations). Run it with the toplevels and tools the medium carries as ROOTs to learn what is
missing from the medium before building a 5 GB image or running a VM test to find out.
"""
import json
import subprocess
import sys
from collections import Counter

STORE = "/nix/store/"


def nix(*args):
    return subprocess.run(["nix", *args], check=True, capture_output=True, text=True).stdout


def main():
    target, roots = sys.argv[1], sys.argv[2:]
    have = set(nix("path-info", "--recursive", *roots).split())
    drvs = json.loads(nix("derivation", "show", "--recursive", target))["derivations"]
    key = lambda d: d[len(STORE):] if d.startswith(STORE) else d
    name = lambda d: d.split("-", 1)[1]

    def out_path(d, o):
        return drvs[d]["env"].get(o) or drvs[d]["outputs"].get(o, {}).get("path")

    seen, build = set(), []

    def visit(d, wanted):
        paths = [out_path(d, o) for o in wanted]
        if paths and all(p and p in have for p in paths):
            return
        if d in seen:
            return
        seen.add(d)
        build.append(d)
        for dep, spec in drvs[d]["inputs"]["drvs"].items():
            visit(dep, spec["outputs"])

    t = key(target)
    visit(t, list(drvs[t]["outputs"]))
    fixed = lambda d: any("hash" in o for o in drvs[d]["outputs"].values())
    fetch = [d for d in build if fixed(d)]
    print(f"derivations to build: {len(build)}   (of which fixed-output, i.e. need the internet: {len(fetch)})")
    top = Counter(name(d).removesuffix(".drv") for d in build if not fixed(d))
    print("\nmost common among those to build:")
    for n, c in top.most_common(25):
        print(f"  {c:4d}  {n}")
    print("\nfixed-output (fetches) — first 25:")
    for d in fetch[:25]:
        print("  ", name(d))


main()
