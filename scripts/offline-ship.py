#!/usr/bin/env python3
"""Which build-time tools must a medium carry so that a machine's own parts can be built offline?

    offline-ship.py TARGET.drv GENERIC.drv [GENERIC.drv ...]

The medium carries finished generic systems. A real machine's system shares almost all derivations with them, and
differs in a few (its fstab, units, top level...). Those few have to be BUILT during the installation, and building
needs the outputs of their input derivations: the build-time tools and libraries, which a finished system does not
keep (it only keeps what it needs to run). This prints the inputs of the machine-specific derivations that come from
the shared part, i.e. what has to be on the medium besides the generic systems.
"""
import json
import subprocess
import sys

STORE = "/nix/store/"


def graph(drv):
    out = subprocess.run(["nix", "derivation", "show", "--recursive", drv], check=True, capture_output=True, text=True).stdout
    return json.loads(out)["derivations"]


def main():
    target, generics = sys.argv[1], sys.argv[2:]
    t = graph(target)
    shared = {}
    for g in generics:
        shared.update(graph(g))
    specific = {k for k in t if k not in shared}
    ship = {}
    for k in specific:
        for dep, spec in t[k]["inputs"]["drvs"].items():
            if dep not in specific:
                ship.setdefault(dep, set()).update(spec["outputs"])
    print(f"machine-specific derivations: {len(specific)}")
    print(f"inputs they take from the shared part (to ship): {len(ship)}\n")
    for dep in sorted(ship, key=lambda d: d.split("-", 1)[1]):
        env = t[dep]["env"]
        paths = [env.get(o) for o in sorted(ship[dep])]
        print(dep.split("-", 1)[1].removesuffix(".drv"), *[p for p in paths if p])


main()
