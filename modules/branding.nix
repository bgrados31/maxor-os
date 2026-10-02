{ config, pkgs, lib, ... }:

# Identidad de Maxor OS: nombre del sistema, arranque silencioso y splash.
# Paleta "Sakura nocturna": fondo #120b12, texto #fbe9f2, acento #ff86b8.
let
  # Tema Plymouth "barra de carga": MAXOR OS en Krona One y una barra fina
  # que avanza con el progreso del arranque.
  maxorPlymouth = pkgs.runCommand "maxor-plymouth" { nativeBuildInputs = [ pkgs.imagemagick ]; } ''
    d=$out/share/plymouth/themes/maxor
    mkdir -p $d
    magick -size 320x6 xc:none -fill 'rgba(251,233,242,0.14)' -draw 'roundrectangle 0,0 319,5 3,3' $d/track.png
    magick -size 320x6 xc:none -fill '#ff86b8' -draw 'roundrectangle 0,0 319,5 3,3' $d/fill.png
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
    Window.SetBackgroundTopColor(0.071, 0.043, 0.071);
    Window.SetBackgroundBottomColor(0.071, 0.043, 0.071);

    sw = Window.GetWidth();
    sh = Window.GetHeight();

    title = Image.Text("MAXOR OS", 0.984, 0.914, 0.949, 1, "Krona One 40");
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
  boot.consoleLogLevel = 3;
  boot.kernelParams = [ "quiet" "splash" "udev.log_level=3" ];

  # Menú de arranque: muestra el menú un par de segundos y renombra la entrada
  boot.loader.timeout = 3;

  # Mensaje de la TTY
  environment.etc."issue".text = ''

    \e[1;36mMaxor OS\e[0m  \n  (\l)

  '';
}
