package install

import "strings"

// GPUOption is one way to use the graphics hardware the machine has.
type GPUOption struct {
	Mode        string // "integrated" | "nvidia" | "hybrid": a value of hardware.gpu in the answers
	Title       string
	Description string
	Recommended bool
}

func has(vendors []string, v string) bool {
	for _, x := range vendors {
		if x == v {
			return true
		}
	}
	return false
}

// GPUOptions says which choices make sense for this hardware, with the recommended one marked. Most machines
// have a single way to go and get no options (the drivers are simply picked); the choice exists where it matters:
// an NVIDIA GPU next to an Intel or AMD one.
func GPUOptions(vendors []string, laptop bool) []GPUOption {
	if !has(vendors, "nvidia") || !(has(vendors, "intel") || has(vendors, "amd")) {
		return nil
	}
	hybrid := GPUOption{"hybrid", "Hybrid", "The integrated GPU draws the desktop and the NVIDIA GPU works on demand (PRIME offload). Best battery and temperature.", laptop}
	nvidia := GPUOption{"nvidia", "NVIDIA only", "Everything runs on the NVIDIA GPU. Fastest, but it uses more power.", !laptop}
	integrated := GPUOption{"integrated", "Integrated only", "No NVIDIA driver: the coolest and the longest battery, without NVIDIA acceleration.", false}
	if laptop {
		return []GPUOption{hybrid, nvidia, integrated}
	}
	return []GPUOption{nvidia, hybrid, integrated}
}

// RecommendedGPU is the mode marked as recommended in these options, or "auto" if there is no choice to make.
func RecommendedGPU(opts []GPUOption) string {
	for _, o := range opts {
		if o.Recommended {
			return o.Mode
		}
	}
	return "auto"
}

// GPUName is how a graphics device is shown to a person: its maker, and for virtual machines what the device is.
// The PCI id stays only for hardware these tables do not know.
func GPUName(vendor, id string) string {
	switch vendor {
	case "intel":
		return "Intel graphics"
	case "amd":
		return "AMD Radeon graphics"
	case "nvidia":
		return "NVIDIA graphics"
	}
	if n, ok := knownGPUs[id]; ok {
		return n
	}
	maker, _, _ := strings.Cut(id, ":")
	if n, ok := gpuMakers[maker]; ok {
		return n
	}
	return "Graphics adapter (" + id + ")"
}

// knownGPUs are single devices worth naming exactly: the virtual ones a test or a VM shows.
var knownGPUs = map[string]string{
	"1af4:1050": "VirtIO GPU (virtual)",
	"1234:1111": "QEMU standard VGA (virtual)",
	"1b36:0100": "QXL (virtual)",
	"15ad:0405": "VMware SVGA (virtual)",
	"80ee:beef": "VirtualBox graphics (virtual)",
}

// gpuMakers name the rest by their PCI vendor id.
var gpuMakers = map[string]string{
	"1af4": "VirtIO graphics (virtual)",
	"1234": "QEMU graphics (virtual)",
	"1b36": "QEMU graphics (virtual)",
	"15ad": "VMware graphics (virtual)",
	"80ee": "VirtualBox graphics (virtual)",
	"1414": "Hyper-V graphics (virtual)",
	"1a03": "ASPEED graphics",
	"102b": "Matrox graphics",
	"5143": "Qualcomm Adreno graphics",
}
