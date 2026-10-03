{ fetchurl }:

# Fuentes de Maxor que no tienen paquete propio en nixpkgs: se bajan solo esos
# archivos del repo google/fonts (fijado por commit y verificado por hash).
let
  gf = "https://raw.githubusercontent.com/google/fonts/5174b3333331c966c38f4355d50b03ca1c1df2f9/ofl";
in
{
  kronaOne = fetchurl {
    name = "KronaOne-Regular.ttf";
    url = "${gf}/kronaone/KronaOne-Regular.ttf";
    hash = "sha256-JzRjkW+WpHB+aX4wE9lVzegVS05tY1pVEy56EvJ2U0w=";
  };
  redHatMono = fetchurl {
    name = "RedHatMono-VF.ttf";
    url = "${gf}/redhatmono/RedHatMono%5Bwght%5D.ttf";
    hash = "sha256-JTN3rCnMzonLG1+yl8aYEv/pk7DENjIrNlYyP/MP0U8=";
  };
}
