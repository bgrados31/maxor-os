{ runCommand, imagemagick, grub2, callPackage }:

# The boot menu of the installation medium (GRUB, graphical): the dark Sakura background with the mark in Krona One
# and a gradient, the entries in Red Hat Mono, the selected one on the accent color. GRUB wants its own font format
# (.pf2): the theme loads every one it finds in this folder, and refers to them by the name grub-mkfont gives them
# ("<family> <style> <size>").
let
  fonts = callPackage ../../packages/fonts.nix { };
in
runCommand "maxor-grub-theme" { nativeBuildInputs = [ imagemagick grub2 ]; } ''
  mkdir -p $out
  grub-mkfont -s 22 -o $out/redhat-22.pf2 ${fonts.redHatMono} 2> /dev/null
  grub-mkfont -s 16 -o $out/redhat-16.pf2 ${fonts.redHatMono} 2> /dev/null

  # background: the mark, filled with the accent gradient, above the menu
  magick -size 1920x1080 xc:'#120b12' $out/base.png
  magick -background none -fill white -font ${fonts.kronaOne} -pointsize 112 -kerning 18 label:"MAXOR OS" $TMPDIR/mask.png
  magick $TMPDIR/mask.png -alpha extract $TMPDIR/alpha.png
  magick -size "$(magick identify -format '%wx%h' $TMPDIR/mask.png)" gradient:'#ff86b8-#c58bff' -rotate -90 -resize "$(magick identify -format '%wx%h!' $TMPDIR/mask.png)" $TMPDIR/grad.png
  magick $TMPDIR/grad.png $TMPDIR/alpha.png -alpha off -compose CopyOpacity -composite $TMPDIR/mark.png
  magick $out/base.png $TMPDIR/mark.png -gravity north -geometry +0+250 -composite $out/background.png
  rm $out/base.png

  # No box behind the selected entry (a pixmap style did not draw in testing, and left dark text on a dark
  # background): the selected entry is simply the one in the accent color.
  cat > $out/theme.txt <<'THEME'
  desktop-image: "background.png"
  desktop-color: "#120b12"
  title-text: ""
  message-font: "Red Hat Mono Regular 16"
  message-color: "#fbe9f2"
  terminal-font: "Red Hat Mono Regular 16"

  + boot_menu {
    left = 32%
    width = 36%
    top = 52%
    height = 34%
    item_font = "Red Hat Mono Regular 22"
    item_color = "#a58a99"
    selected_item_font = "Red Hat Mono Regular 22"
    selected_item_color = "#ff86b8"
    item_height = 40
    item_padding = 14
    item_spacing = 8
    scrollbar = false
  }

  + label {
    left = 0
    width = 100%
    top = 93%
    align = "center"
    font = "Red Hat Mono Regular 16"
    color = "#6f5a69"
    text = "enter  start      e  edit      c  command line"
  }
  THEME
''
