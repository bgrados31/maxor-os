package install

import "testing"

func TestGPUOptionsOnlyWhereThereIsARealChoice(t *testing.T) {
	for _, c := range []struct {
		name    string
		vendors []string
	}{
		{"intel only", []string{"intel"}}, {"amd only", []string{"amd"}}, {"nvidia only", []string{"nvidia"}},
		{"virtual machine", []string{"virtio"}}, {"nothing detected", nil},
	} {
		if o := GPUOptions(c.vendors, true); o != nil {
			t.Fatalf("%s has a single way to go, not %v", c.name, o)
		}
	}
}

func TestGPUOptionsRecommendHybridOnALaptopAndNvidiaOnADesktop(t *testing.T) {
	lap := GPUOptions([]string{"intel", "nvidia"}, true)
	if len(lap) != 3 || lap[0].Mode != "hybrid" || !lap[0].Recommended || RecommendedGPU(lap) != "hybrid" {
		t.Fatalf("a hybrid laptop: %+v", lap)
	}
	desk := GPUOptions([]string{"amd", "nvidia"}, false)
	if len(desk) != 3 || desk[0].Mode != "nvidia" || RecommendedGPU(desk) != "nvidia" {
		t.Fatalf("a desktop with an iGPU and an NVIDIA card: %+v", desk)
	}
	n := 0
	for _, o := range lap {
		if o.Recommended {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("exactly one option is recommended, got %d", n)
	}
	if RecommendedGPU(nil) != "auto" {
		t.Fatal("no choice means auto")
	}
}

func TestGPUNameIsReadable(t *testing.T) {
	for _, c := range []struct{ vendor, id, want string }{
		{"nvidia", "10de:28e0", "NVIDIA graphics"},
		{"other", "1af4:1050", "VirtIO GPU (virtual)"},
		{"other", "15ad:9999", "VMware graphics (virtual)"},
		{"other", "abcd:0001", "Graphics adapter (abcd:0001)"},
	} {
		if got := GPUName(c.vendor, c.id); got != c.want {
			t.Errorf("GPUName(%q, %q) = %q, want %q", c.vendor, c.id, got, c.want)
		}
	}
}
