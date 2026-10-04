package install

import "testing"

func TestDiskLabelAndContents(t *testing.T) {
	d := Disk{Path: "/dev/nvme0n1", Model: "Samsung SSD", Size: 512_110_190_592, Windows: true,
		Partitions: []Partition{{Path: "/dev/nvme0n1p1"}, {Path: "/dev/nvme0n1p2"}}}
	if got := d.Label(); got != "/dev/nvme0n1 · 512 GB · Samsung SSD" {
		t.Fatalf("label: %q", got)
	}
	if got := d.Contents(); got != "2 partitions (Windows)" {
		t.Fatalf("contents: %q", got)
	}
	if (Disk{}).Contents() != "empty" {
		t.Fatal("no partitions is empty")
	}
	usb := Disk{Path: "/dev/sdb", Size: 16_000_000_000, Removable: true}
	if usb.Label() != "/dev/sdb · 16 GB · removable" {
		t.Fatalf("label: %q", usb.Label())
	}
}

func TestDiskProblemsMirrorThePreflight(t *testing.T) {
	ok := Disk{Size: 64 * gib}
	if ok.Problem("whole") != "" {
		t.Fatal("a 64 GiB free disk is fine")
	}
	for name, d := range map[string]Disk{
		"small":    {Size: 16 * gib},
		"mounted":  {Size: 64 * gib, Mounted: true},
		"readonly": {Size: 64 * gib, ReadOnly: true},
	} {
		if d.Problem("whole") == "" {
			t.Errorf("%s should be refused", name)
		}
	}
	alongside := Disk{Size: 512 * gib, Free: []Free{{Start: 2048, End: 100, Sectors: 10 * 2097152}, {Start: 5, End: 6, Sectors: 60 * 2097152}}}
	if alongside.Problem("alongside") != "" {
		t.Fatal("a 60 GiB region is enough")
	}
	if (Disk{Size: 512 * gib, Free: []Free{{Sectors: 10 * 2097152}}}).Problem("alongside") == "" {
		t.Fatal("10 GiB is not enough to install alongside")
	}
	if (Disk{Size: 512 * gib}).Problem("alongside") == "" {
		t.Fatal("no free region at all")
	}
}

func TestLargestFreePicksTheBiggest(t *testing.T) {
	d := Disk{Free: []Free{{Start: 1, Sectors: 10}, {Start: 2, Sectors: 99}, {Start: 3, Sectors: 50}}}
	if f := d.LargestFree(); f == nil || f.Start != 2 {
		t.Fatalf("%+v", f)
	}
	if (Disk{}).LargestFree() != nil {
		t.Fatal("none")
	}
}

func TestHumanSizeUsesPowersOfTen(t *testing.T) {
	for in, want := range map[int64]string{500_107_862_016: "500 GB", 2_000_398_934_016: "2.0 TB", 128_035_676_160: "128 GB", 250_000_000: "250 MB"} {
		if got := HumanSize(in); got != want {
			t.Errorf("%d → %q, want %q", in, got, want)
		}
	}
}
