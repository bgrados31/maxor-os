{ pkgs, systems }:

# What building the parts that belong to a machine needs, besides the finished generic systems.
#
# A finished system keeps what it needs to RUN. Installing a different machine means building a few derivations of its own
# (the system's top level, /etc, the units, the activation scripts, the user's environment, the initrd), and building them
# needs what went INTO them: the small files and trees that were copied or merged into the finished ones and left no
# reference behind (the unit files, the initrd's configuration, the generated shell completions), and a couple of build
# tools. All of that is at hand when the generic systems are built, so it is collected here instead of being listed by
# hand: the text below mentions every one of those paths, which makes it (and so the medium) carry them.
#
# `scripts/offline-missing.py` checks the result against a real target: what is left to build must be only the machine's own
# derivations, and nothing that needs the internet.
let
  lib = pkgs.lib;

  # the inputs of the derivations that hold a system together (their attributes mention what they were made from)
  inputsOf = d: builtins.toJSON d.drvAttrs;
  parts = c:
    [ c.system.build.toplevel c.system.build.etc c.system.path c.system.build.initialRamdisk ]
    ++ lib.concatMap (u: [ u.home.activationPackage u.home.path u.home-files ]) (lib.attrValues c.home-manager.users);

  # the pieces that were merged into those: the contents of /etc, the units, the initrd, the user's files
  sources = attrs: map toString (lib.filter (s: s != null) (lib.mapAttrsToList (_: v: v.source or null) attrs));
  files = c:
    sources c.environment.etc
    ++ lib.filter (x: x != "") (lib.mapAttrsToList (_: u: toString (u.unit or "")) c.systemd.units)
    ++ sources c.boot.initrd.systemd.contents
    ++ lib.concatMap (u: sources u.home.file) (lib.attrValues c.home-manager.users);
in
pkgs.writeText "maxor-build-inputs" (
  lib.concatMapStringsSep "\n" inputsOf (lib.concatMap parts systems) + "\n"
  + lib.concatStringsSep "\n" (lib.concatMap files systems) + "\n"
  # build tools that only appear while building (not in any finished system)
  + "${pkgs.babelfish}\n${pkgs.nukeReferences}\n"
)
