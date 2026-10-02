{ config, pkgs, ... }:

# Tipografía de Maxor OS: Figtree (interfaz), Red Hat Mono (terminal) y
# Krona One (logo y títulos). Las dos últimas no tienen paquete propio en
# nixpkgs: se bajan solo esos archivos del repo google/fonts (fijado por commit).
let
  gf = "https://raw.githubusercontent.com/google/fonts/5174b3333331c966c38f4355d50b03ca1c1df2f9/ofl";

  kronaOne = pkgs.fetchurl {
    name = "KronaOne-Regular.ttf";
    url = "${gf}/kronaone/KronaOne-Regular.ttf";
    hash = "sha256-JzRjkW+WpHB+aX4wE9lVzegVS05tY1pVEy56EvJ2U0w=";
  };
  redHatMono = pkgs.fetchurl {
    name = "RedHatMono-VF.ttf";
    url = "${gf}/redhatmono/RedHatMono%5Bwght%5D.ttf";
    hash = "sha256-JTN3rCnMzonLG1+yl8aYEv/pk7DENjIrNlYyP/MP0U8=";
  };

  maxorFonts = pkgs.runCommand "maxor-fonts" { } ''
    d=$out/share/fonts/truetype/maxor
    mkdir -p $d
    cp ${kronaOne} $d/KronaOne-Regular.ttf
    cp ${redHatMono} $d/RedHatMono-VF.ttf
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
