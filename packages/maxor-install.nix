{ lib, writeShellApplication, jq, coreutils, util-linux, parted, dosfstools, e2fsprogs, btrfs-progs, cryptsetup
, git, curl, openssl, gnugrep, gnused, gawk, findutils, nixos-install-tools, nix, systemd, iproute2, maxor }:

# El motor del instalador: `maxor-install`. Los scripts reales viven en installer/engine/ (lib/
# una etapa por archivo) y se ensamblan en este orden; shellcheck revisa el resultado al compilar.
# Ver docs/INSTALLER.md.
let
  src = ../installer/engine;
  files = [
    "lib/common.sh"
    "lib/answers.sh"
    "lib/preflight.sh"
    "lib/disk.sh"
    "lib/luks.sh"
    "lib/filesystem.sh"
    "lib/host.sh"
    "lib/nixinstall.sh"
    "lib/bootloader.sh"
    "lib/finish.sh"
    "main.sh"
  ];
in
writeShellApplication {
  name = "maxor-install";
  runtimeInputs = [ jq coreutils util-linux parted dosfstools e2fsprogs btrfs-progs cryptsetup git curl openssl gnugrep gnused gawk findutils nixos-install-tools nix systemd iproute2 ];
  runtimeEnv = {
    MAXOR_INSTALL_SCHEMA = "${../installer/schema/answers.v1.json}"; # interpolado: Nix lo copia al store y lo mantiene vivo
    MAXOR_PROFILES = "${../modules/profiles-catalog.json}";
    MAXOR_BIN = "${maxor}/bin/maxor";
  };
  excludeShellChecks = [ "SC2034" "SC2001" "SC2155" "SC2086" "SC2012" "SC2015" "SC2016" "SC2312" ];
  text = lib.concatMapStringsSep "\n" (f: builtins.readFile (src + "/${f}")) files;
}
