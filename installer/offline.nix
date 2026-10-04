{ config, pkgs, ... }:

# What a machine needs in its store to evaluate and BUILD a NixOS system without a network. nixos-install takes the big
# finished paths (the desktop, the kernel...) from the store it runs in, but the parts that belong to the machine being
# installed (its fstab, its initrd, its units, the system's top level) are built on the spot, and building needs
# these tools. The list is what nixpkgs' own installer tests carry for the same reason. Used by the installation
# medium and by the install test, so the test proves what the medium ships.
{
  system.extraDependencies = with pkgs; [
    stdenv
    stdenvNoCC
    bintools
    brotli
    brotli.dev
    brotli.lib
    desktop-file-utils
    docbook5
    docbook_xsl_ns
    kbd.dev
    kmod.dev
    libarchive.dev
    libxml2.bin
    libxslt.bin
    perlPackages.ConfigIniFiles
    perlPackages.FileSlurp
    perlPackages.JSON
    perlPackages.ListCompare
    perlPackages.XMLLibXML
    (python3.withPackages (p: [ p.mistune ]))
    shared-mime-info
    sudo
    switch-to-configuration-ng
    texinfo
    unionfs-fuse
    lndir
    shellcheck-minimal
    systemdMinimal.out
    curl
    zstd.bin
    mypy
    config.boot.bootspec.package
  ];
}
