package install

import (
	"sort"
	"strings"
)

// Span is one stretch of a disk as the installer draws it: a partition, or free space.
type Span struct {
	Kind  string // "efi", "windows", "linux", "maxor", "other" or "free": what colour it gets
	Label string // what the legend calls it: "Windows", "EFI", "Maxor OS", "free"…
	Bytes int64
	New   bool // this installation creates it
	start int64
}

// BootBytes is the size of the boot partition the engine creates (DISK_ESP_SECTORS in installer/engine/lib/disk.sh).
const BootBytes = 1 << 30

// Layout is the disk as it is now, from its first sector to its last: its partitions and its free regions, in
// order. With no partitions it is one free region the size of the disk.
func (d Disk) Layout() []Span {
	var out []Span
	known := len(d.Partitions) > 0
	for _, p := range d.Partitions {
		known = known && p.Start > 0
	}
	for i, p := range d.Partitions {
		kind, label := classify(p)
		start := p.Start
		if !known {
			start = int64(i) // the engine did not say where they are: keep their order, before the free space
		}
		out = append(out, Span{Kind: kind, Label: label, Bytes: p.Size, start: start})
	}
	for _, f := range d.Free {
		start := f.Start
		if !known {
			start = int64(len(d.Partitions)) + f.Start
		}
		out = append(out, Span{Kind: "free", Label: tr("free"), Bytes: f.Sectors * 512, start: start})
	}
	if len(out) == 0 {
		return []Span{{Kind: "free", Label: tr("free"), Bytes: d.Size}}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].start < out[j].start })
	return out
}

// Planned is the disk as it will be after installing with this strategy: the whole disk becomes boot and Maxor OS;
// alongside, the chosen free region (the largest one) does, and the rest is left exactly as it is.
func (d Disk) Planned(strategy string) []Span {
	maxor := func(bytes int64) []Span {
		return []Span{
			{Kind: "efi", Label: "boot", Bytes: BootBytes, New: true},
			{Kind: "maxor", Label: tr("Maxor OS"), Bytes: max(bytes-BootBytes, 0), New: true},
		}
	}
	if strategy != "alongside" {
		return maxor(d.Size)
	}
	f := d.LargestFree()
	var out []Span
	for _, r := range d.Layout() {
		if f != nil && r.Kind == "free" && r.Bytes == f.Sectors*512 {
			out = append(out, maxor(r.Bytes)...)
			f = nil // only the first region that size
			continue
		}
		out = append(out, r)
	}
	return out
}

// classify says what a partition is from what is on it, the way a person would name it.
func classify(p Partition) (kind, label string) {
	fs := strings.ToLower(p.FSType)
	switch {
	case p.PartLabel == "maxor-root":
		return "maxor", tr("Maxor OS")
	case p.PartLabel == "MAXOR-ESP":
		return "efi", "boot"
	case fs == "vfat" && p.Size <= 4<<30:
		return "efi", "EFI"
	case fs == "ntfs" || fs == "bitlocker":
		if strings.Contains(strings.ToLower(p.Label), "recovery") || p.Size < 2<<30 {
			return "windows", tr("Windows recovery")
		}
		return "windows", "Windows"
	case fs == "ext4" || fs == "btrfs" || fs == "xfs" || fs == "f2fs":
		return "linux", tr("Linux")
	case fs == "swap":
		return "linux", "swap"
	case fs == "crypto_luks":
		return "linux", "encrypted"
	case fs == "":
		return "other", "unformatted"
	}
	return "other", p.FSType
}

// Cells shares width cells among the regions in proportion to their size, so that every region gets at least one
// cell (a 100 MB partition next to a 2 TB one must still show) and the cells add up to exactly width.
func Cells(regions []Span, width int) []int {
	out := make([]int, len(regions))
	if width <= 0 || len(regions) == 0 {
		return out
	}
	var total int64
	for _, r := range regions {
		total += max(r.Bytes, 0)
	}
	if total == 0 || len(regions) > width {
		for i := range out {
			if i < width {
				out[i] = 1
			}
		}
		return out
	}
	// one cell each, then the rest by the largest remainders
	rest := width - len(regions)
	type rem struct {
		i int
		r float64
	}
	var rems []rem
	used := 0
	for i, r := range regions {
		exact := float64(max(r.Bytes, 0)) / float64(total) * float64(rest)
		out[i] = 1 + int(exact)
		used += int(exact)
		rems = append(rems, rem{i, exact - float64(int(exact))})
	}
	sort.SliceStable(rems, func(a, b int) bool { return rems[a].r > rems[b].r })
	for k := 0; used < rest; k++ {
		out[rems[k%len(rems)].i]++
		used++
	}
	return out
}
