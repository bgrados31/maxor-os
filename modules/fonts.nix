{ config, pkgs, ... }:

# Tipografía de Maxor OS: Figtree (interfaz), Red Hat Mono (terminal) y
# Krona One (logo y títulos). Las dos últimas no tienen paquete propio en
# nixpkgs: ver packages/fonts.nix.
let
  fonts = pkgs.callPackage ../packages/fonts.nix { };

  maxorFonts = pkgs.runCommand "maxor-fonts" { } ''
    d=$out/share/fonts/truetype/maxor
    mkdir -p $d
    cp ${fonts.kronaOne} $d/KronaOne-Regular.ttf
    cp ${fonts.redHatMono} $d/RedHatMono-VF.ttf
  '';
in
{
  fonts.packages = with pkgs; [
    maxorFonts
    figtree
    nerd-fonts.symbols-only # iconos en kitty y la barra, junto a Red Hat Mono
    noto-fonts
    noto-fonts-cjk-sans
    noto-fonts-color-emoji
    nerd-fonts.jetbrains-mono # respaldo
  ];

  fonts.fontconfig.defaultFonts = {
    sansSerif = [ "Figtree" ];
    monospace = [ "Red Hat Mono" "Symbols Nerd Font Mono" ];
  };

  # Plymouth dibuja el logo con Krona One
  boot.plymouth.font = "${maxorFonts}/share/fonts/truetype/maxor/KronaOne-Regular.ttf";
}
