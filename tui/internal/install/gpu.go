package install

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
