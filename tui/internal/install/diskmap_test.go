package install

import (
	"reflect"
	"testing"
)

func dualBoot() Disk {
	return Disk{Path: "/dev/nvme0n1", Size: 512 << 30, Windows: true,
		Partitions: []Partition{
			{Path: "p3", Start: 2_000_000, Size: 200 << 30, FSType: "ntfs", Label: "Windows"},
			{Path: "p1", Start: 2048, Size: 100 << 20, FSType: "vfat"},
		},
		Free: []Free{{Start: 500_000_000, End: 1_000_000_000, Sectors: 500_000_000}}}
}

func kinds(rs []Span) []string {
	var out []string
	for _, r := range rs {
		out = append(out, r.Kind)
	}
	return out
}

func TestLayoutFollowsTheDiskInOrder(t *testing.T) {
	if got := kinds(dualBoot().Layout()); !reflect.DeepEqual(got, []string{"efi", "windows", "free"}) {
		t.Fatalf("layout = %v", got)
	}
	if got := (Disk{Size: 100 << 30}).Layout(); len(got) != 1 || got[0].Kind != "free" || got[0].Bytes != 100<<30 {
		t.Fatalf("an empty disk is one free region: %v", got)
	}
}

func TestPlannedReplacesOnlyTheFreeRegionAlongside(t *testing.T) {
	got := dualBoot().Planned("alongside")
	if k := kinds(got); !reflect.DeepEqual(k, []string{"efi", "windows", "efi", "maxor"}) {
		t.Fatalf("alongside = %v", k)
	}
	if !got[2].New || !got[3].New || got[1].New {
		t.Fatalf("only boot and Maxor OS are new: %+v", got)
	}
	if w := dualBoot().Planned("whole"); !reflect.DeepEqual(kinds(w), []string{"efi", "maxor"}) || w[1].Bytes != 511<<30 {
		t.Fatalf("whole = %+v", w)
	}
}

func TestCellsAreProportionalAndNothingVanishes(t *testing.T) {
	c := Cells([]Span{{Bytes: 100 << 20}, {Bytes: 200 << 30}, {Bytes: 300 << 30}}, 50)
	if c[0] != 1 || c[0]+c[1]+c[2] != 50 || c[1] >= c[2] {
		t.Fatalf("cells = %v", c)
	}
}
