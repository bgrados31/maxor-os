{ config, pkgs, ... }:

# Tipografía de Maxor OS: Figtree (interfaz), Red Hat Mono (terminal) y Cinzel (el logo «MAXOR OS»).
# Las dos últimas no tienen paquete propio en nixpkgs: ver packages/fonts.nix. Krona One, el logo
# anterior, se queda instalada para los temas y ajustes que aún la nombren.
let
  fonts = pkgs.callPackage ../packages/fonts.nix { };

  maxorFonts = pkgs.runCommand "maxor-fonts" { } ''
    d=$out/share/fonts/truetype/maxor
    mkdir -p $d
    cp ${fonts.cinzel} $d/Cinzel-VF.ttf
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

  # El texto de Plymouth (mensajes, la frase del cifrado) en Cinzel; el logo es una imagen ya dibujada
  # (modules/branding.nix).
  boot.plymouth.font = "${maxorFonts}/share/fonts/truetype/maxor/Cinzel-VF.ttf";
}
