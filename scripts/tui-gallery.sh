#!/usr/bin/env bash
# Draw every step of the installer, in colour and at the sizes people use, as pictures: a design review without a
# virtual machine.
#
#   scripts/tui-gallery.sh [OUT]      OUT defaults to ./tui-gallery; pictures land in OUT/<cols>x<rows>/*.png
#
# The steps are drawn by the Go test TestInstallerGallery (tui/internal/app/gallery_test.go), with sample disks and
# hardware, and photographed in a real terminal with vhs. Transitions are drawn finished; motion is checked in the VM.
set -euo pipefail

die() { printf '✗ %s\n' "$*" >&2; exit 1; }
say() { printf '· %s\n' "$*"; }

root="$(cd "$(dirname "$0")/.." && pwd)"
out="$(realpath -m "${1:-tui-gallery}")"
rm -rf "$out"
mkdir -p "$out"

say "drawing the steps"
(cd "$root/tui" && MAXOR_GALLERY="$out" CGO_ENABLED=0 nix shell nixpkgs#go -c go test ./internal/app -run TestInstallerGallery -count=1 > /dev/null) \
  || die "the gallery test failed"

for dir in "$out"/*x*/; do
  size="$(basename "$dir")"
  geom="${size##*-}" # «stack-132x40» → «132x40»
  cols="${geom%x*}" rows="${geom#*x}"
  say "photographing $size"
  tape="$dir/gallery.tape"
  {
    printf 'Output "%s/run.gif"\n' "$dir"
    printf 'Set Shell bash\nSet FontSize 16\nSet Padding 0\nSet Margin 0\n'
    # one cell of the font at size 16 is about 10 × 20 pixels; a margin so no row wraps, and room for the prompt
    # (even sizes: the video encoder behind vhs refuses odd ones, and then no picture is taken)
    printf 'Set Width %d\nSet Height %d\nSet TypingSpeed 1ms\n' "$(((cols * 10 + 40) / 2 * 2))" "$(((rows * 20 + 60) / 2 * 2))"
    for f in "$dir"/*.ansi; do
      printf 'Type "clear; tput civis; cat %s; sleep 60"\nEnter\nSleep 400ms\nScreenshot "%s"\nCtrl+C\n' \
        "$f" "${f%.ansi}.png"
    done
  } > "$tape"
  nix shell nixpkgs#vhs -c vhs "$tape" > /dev/null 2>&1 || die "vhs could not photograph $size"
  rm -f "$tape" "$dir/run.gif" "$dir"/*.ansi
done
say "done: $out"
