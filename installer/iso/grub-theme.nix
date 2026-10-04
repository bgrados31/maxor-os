{ runCommand, imagemagick, grub2, callPackage }:

# The boot menu of the installation medium (GRUB, graphical): the brand gradient of Maxor Dark (violet → indigo →
# navy) with the mark in Cinzel, the entries in Red Hat Mono, the selected one in the accent. GRUB wants its own
# font format (.pf2): the theme loads every one it finds in this folder, and refers to them by the name grub-mkfont
# gives them ("<family> <style> <size>").
let
  fonts = callPackage ../../packages/fonts.nix { };
in
runCommand "maxor-grub-theme" { nativeBuildInputs = [ imagemagick grub2 ]; } ''
  mkdir -p $out
  grub-mkfont -s 22 -o $out/redhat-22.pf2 ${fonts.redHatMono} 2> /dev/null
  grub-mkfont -s 16 -o $out/redhat-16.pf2 ${fonts.redHatMono} 2> /dev/null

  # background: the gradient of themes/maxor-dark/gradient.json, and the mark above the menu with its letters open
  magick -size 1920x1080 xc: -sparse-color Shepards '0,0 #6700a3  960,540 #1b2062  1920,1080 #050c38' -colorspace sRGB $out/base.png
  magick -background none -fill '#eef0ff' -font ${fonts.cinzel} -pointsize 104 -kerning 34 label:"MAXOR OS" $TMPDIR/mark.png
  magick $out/base.png $TMPDIR/mark.png -gravity north -geometry +0+250 -composite $out/background.png
  rm $out/base.png

  # No box behind the selected entry (a pixmap style did not draw in testing, and left dark text on a dark
  # background): the selected entry is simply the one in the accent color.
  cat > $out/theme.txt <<'THEME'
  desktop-image: "background.png"
  desktop-color: "#050c38"
  title-text: ""
  message-font: "Red Hat Mono Regular 16"
  message-color: "#eef0ff"
  terminal-font: "Red Hat Mono Regular 16"

  + boot_menu {
    left = 32%
    width = 36%
    top = 52%
    height = 34%
    item_font = "Red Hat Mono Regular 22"
    item_color = "#b8bce0"
    selected_item_font = "Red Hat Mono Regular 22"
    selected_item_color = "#c084ff"
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
    color = "#9ea3d1"
    text = "enter  start      e  edit      c  command line"
  }
  THEME
''
