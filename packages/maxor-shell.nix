{ lib, runCommand, python3, callPackage, dmsShell }:

# Maxor Shell: DankMaterialShell con la identidad de Maxor OS.
#
# No es una copia del código de DMS. Es una capa de parches sobre el paquete de
# upstream, de modo que seguimos recibiendo sus mejoras con `nix flake update`.
# Cambia solo lo que el usuario ve: el logo, el nombre y unos pocos textos.
# `substituteInPlace --replace-fail` hace que el build falle si upstream
# reescribe una de esas líneas, en vez de dejar una marca a medias sin avisar.
#
# DMS se distribuye bajo MIT; la licencia y el aviso de copyright originales
# se conservan en el paquete.
let
  fonts = callPackage ./fonts.nix { };

  mark = runCommand "maxor-mark"
    { nativeBuildInputs = [ (python3.withPackages (p: [ p.fonttools ])) ]; }
    ''
      mkdir -p $out
      python3 ${./make-mark.py} ${fonts.kronaOne} $out
    '';
in
dmsShell.overrideAttrs (old: {
  pname = "maxor-shell";

  postInstall = (old.postInstall or "") + ''
    shell=$out/share/quickshell/dms

    # Logo: el botón del launcher, «Acerca de», bienvenida y novedades lo usan.
    for logo in danklogo.svg danklogo2.svg danklogonormal.svg; do
      cp ${mark}/mark.svg "$shell/assets/$logo"
    done
    cp ${mark}/icon.svg $out/share/icons/hicolor/scalable/apps/com.danklinux.dms.svg
    # El botón del launcher dibuja ese logo por defecto (DMS pone el icono de aplicaciones): así sale
    # desde el primer arranque, sin tocar los ajustes ni reiniciar la barra.
    substituteInPlace $shell/Common/settings/BarWidgetDefaults.js \
      --replace-fail 'launcherLogoMode: "apps",' 'launcherLogoMode: "dank",'

    # Nombre en la interfaz
    substituteInPlace $shell/Modules/Settings/AboutTab.qml \
      --replace-fail 'text: "DANK LINUX"' 'text: "MAXOR OS"'
    substituteInPlace $shell/Modals/Greeter/GreeterWelcomePage.qml \
      --replace-fail 'Welcome to DankMaterialShell' 'Welcome to Maxor OS'
    substituteInPlace $shell/Services/MprisController.qml \
      --replace-fail '"identity": "DankMaterialShell"' '"identity": "Maxor Shell"'
    substituteInPlace $shell/Modules/Settings/SoftwareUpdatesTab.qml \
      --replace-fail 'title: "DankMaterialShell"' 'title: "Maxor Shell"'
    substituteInPlace $shell/Modules/Settings/PluginsHubHeader.qml \
      --replace-fail 'Extend DankMaterialShell with' 'Extend Maxor Shell with'
  '';

  meta = old.meta // {
    description = "Maxor Shell: DankMaterialShell con la identidad de Maxor OS";
    homepage = "https://github.com/bgrados31/maxor-os";
  };
})
