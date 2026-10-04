{ config, pkgs, lib, ... }:

# Identidad de Maxor OS: nombre del sistema, arranque silencioso y splash.
# Paleta de Maxor Dark (themes/maxor-dark): azul marino #050c38, índigo #1b2062, texto #eef0ff,
# acento violeta #c084ff y coral #ff5a57.
let
  fonts = pkgs.callPackage ../packages/fonts.nix { };

  # Tema Plymouth "barra de carga": MAXOR OS en Cinzel sobre un degradado índigo → azul marino, y una barra
  # fina, del violeta al coral, que avanza con el progreso del arranque. El nombre se dibuja aquí como imagen:
  # Plymouth no sabe espaciar letras, y la marca va con las letras abiertas.
  maxorPlymouth = pkgs.runCommand "maxor-plymouth" { nativeBuildInputs = [ pkgs.imagemagick ]; } ''
    d=$out/share/plymouth/themes/maxor
    mkdir -p $d
    magick -background none -fill '#eef0ff' -font ${fonts.cinzel} -pointsize 64 -kerning 22 label:"MAXOR OS" -depth 8 $d/title.png
    magick -size 320x6 xc:none -fill 'rgba(238,240,255,0.14)' -draw 'roundrectangle 0,0 319,5 3,3' $d/track.png
    magick -size 6x320 gradient:'#c084ff-#ff5a57' -rotate -90 \
      \( -size 320x6 xc:none -fill white -draw 'roundrectangle 0,0 319,5 3,3' \) -compose CopyOpacity -composite -depth 8 $d/fill.png
    cat > $d/maxor.plymouth <<EOT
    [Plymouth Theme]
    Name=Maxor OS
    Description=Maxor OS boot splash
    ModuleName=script

    [script]
    ImageDir=$d
    ScriptFile=$d/maxor.script
    EOT
    cat > $d/maxor.script <<'EOT'
    Window.SetBackgroundTopColor(0.106, 0.125, 0.384);
    Window.SetBackgroundBottomColor(0.020, 0.047, 0.220);

    sw = Window.GetWidth();
    sh = Window.GetHeight();

    title = Image("title.png");
    title_sprite = Sprite(title);
    title_sprite.SetX(sw / 2 - title.GetWidth() / 2);
    title_sprite.SetY(sh / 2 - title.GetHeight() - 16);

    track = Image("track.png");
    track_sprite = Sprite(track);
    track_x = sw / 2 - track.GetWidth() / 2;
    track_y = sh / 2 + 24;
    track_sprite.SetPosition(track_x, track_y, 1);

    fill = Image("fill.png");
    fill_sprite = Sprite();
    fill_sprite.SetPosition(track_x, track_y, 2);

    fun progress_cb (duration, progress) {
      w = Math.Int(fill.GetWidth() * progress);
      if (w < 6) w = 6;
      fill_sprite.SetImage(fill.Scale(w, fill.GetHeight()));
    }
    Plymouth.SetBootProgressFunction(progress_cb);
    EOT
  '';
in
{
  # ── Nombre del sistema (ID se queda "nixos": hay herramientas que lo usan)
  system.nixos.distroName = "Maxor OS";

  # ── Arranque silencioso + splash ───────────────────────────────────
  boot.plymouth = {
    enable = true;
    theme = "maxor";
    themePackages = [ maxorPlymouth ];
  };
  boot.consoleLogLevel = 0; # ni los avisos del kernel (p. ej. watchdog) al apagar
  boot.kernelParams = [
    "quiet"
    "splash"
    "udev.log_level=3"
    "rd.udev.log_level=3"
    "systemd.show_status=false"
    "systemd.log_level=warning" # systemd-shutdown no imprime "Syncing filesystems..." etc.
    "rd.systemd.show_status=false"
    "vt.global_cursor_default=0" # sin cursor parpadeando sobre el splash
  ];
  boot.initrd.verbose = false;
  # initrd con systemd: el splash aparece antes y sin saltos de texto
  boot.initrd.systemd.enable = true;

  # Menú de arranque: con un solo sistema arranca directo, sin esperar; el menú (generaciones anteriores,
  # firmware) aparece manteniendo pulsada una tecla, p. ej. Espacio, al encender. Un equipo con otro
  # sistema (Windows) lo muestra: el instalador escribe boot.loader.timeout en su host/boot.nix.
  boot.loader.timeout = lib.mkDefault 0;

  # Mensaje de la TTY
  environment.etc."issue".text = ''

    \e[1;36mMaxor OS\e[0m  \n  (\l)

  '';
}
