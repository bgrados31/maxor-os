package install

import (
	"fmt"
	"sort"
	"strings"
)

// Free is a free region of a disk, in 512-byte sectors, as reported by `maxor-install probe`.
type Free struct {
	Start   int64 `json:"start"`
	End     int64 `json:"end"`
	Sectors int64 `json:"sectors"`
}

// Partition is one partition of a disk.
type Partition struct {
	Path        string   `json:"path"`
	Size        int64    `json:"size"`
	Start       int64    `json:"start"` // first sector (512 bytes); 0 when the engine did not say
	FSType      string   `json:"fstype"`
	Label       string   `json:"label"`
	PartLabel   string   `json:"partlabel"`
	Mountpoints []string `json:"mountpoints"`
}

// Disk is a disk of this machine and what is on it (the output of `maxor-install probe`).
type Disk struct {
	Path       string      `json:"path"`
	Model      string      `json:"model"`
	Size       int64       `json:"size"` // bytes
	Transport  string      `json:"transport"`
	Removable  bool        `json:"removable"`
	ReadOnly   bool        `json:"readonly"`
	Mounted    bool        `json:"mounted"`
	Windows    bool        `json:"windows"`
	Maxor      bool        `json:"maxor"`
	Partitions []Partition `json:"partitions"`
	Free       []Free      `json:"free"`
}

const gib = int64(1) << 30

// MinWholeGiB and MinRegionGiB are the sizes the engine requires.
const (
	MinWholeGiB  = 32
	MinRegionGiB = 40
)

// Label is a one-line description for a list: «/dev/nvme0n1 · 512 GB · Samsung SSD».
func (d Disk) Label() string {
	parts := []string{d.Path, HumanSize(d.Size)}
	if d.Model != "" {
		parts = append(parts, d.Model)
	}
	if d.Removable || d.Transport == "usb" {
		parts = append(parts, "removable")
	}
	return strings.Join(parts, " · ")
}

// Contents is what is on the disk, in a few words, so the user is never asked to pick blind.
func (d Disk) Contents() string {
	if len(d.Partitions) == 0 {
		return tr("empty")
	}
	var kinds []string
	if d.Windows {
		kinds = append(kinds, "Windows")
	}
	if d.Maxor {
		kinds = append(kinds, tr("a previous Maxor OS"))
	}
	n := len(d.Partitions)
	desc := trn("%d partition", "%d partitions", n)
	if n == 1 {
		desc = "1 partition"
	}
	if len(kinds) > 0 {
		return desc + " (" + strings.Join(kinds, ", ") + ")"
	}
	return desc
}

// LargestFree is the biggest free region, or nil.
func (d Disk) LargestFree() *Free {
	if len(d.Free) == 0 {
		return nil
	}
	f := append([]Free(nil), d.Free...)
	sort.Slice(f, func(i, j int) bool { return f[i].Sectors > f[j].Sectors })
	return &f[0]
}

// Problem says why the disk cannot be used with this strategy, or "" if it can. It mirrors the engine's
// preflight so the wizard explains early what the engine would refuse late.
func (d Disk) Problem(strategy string) string {
	switch {
	case d.ReadOnly:
		return "read-only"
	case d.Mounted:
		return tr("in use (something on it is mounted)")
	}
	if strategy == "alongside" {
		f := d.LargestFree()
		if f == nil || f.Sectors*512 < MinRegionGiB*gib {
			return fmt.Sprintf(tr("no free region of %d GiB or more"), MinRegionGiB)
		}
		return ""
	}
	if d.Size < MinWholeGiB*gib {
		return fmt.Sprintf(tr("smaller than %d GiB"), MinWholeGiB)
	}
	return ""
}

// HumanSize prints a size in bytes as 512 GB, 1.8 TB…, as disk makers do (powers of ten).
func HumanSize(b int64) string {
	const k = 1000.0
	f := float64(b)
	switch {
	case f >= k*k*k*k:
		return fmt.Sprintf(tr("%.1f TB"), f/(k*k*k*k))
	case f >= k*k*k:
		return fmt.Sprintf(tr("%.0f GB"), f/(k*k*k))
	case f >= k*k:
		return fmt.Sprintf(tr("%.0f MB"), f/(k*k))
	}
	return fmt.Sprintf("%d B", b)
}

// RegionOf turns the largest free region of a disk into the Region the engine wants, leaving the region
// as it is (the engine aligns and checks it again before writing).
func RegionOf(f Free) Region { return Region{Start: f.Start, End: f.End} }
