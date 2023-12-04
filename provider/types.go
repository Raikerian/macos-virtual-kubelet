package provider

const (
	// OperatingSystemMacOS is the configuration value for defining MacOS.
	OperatingSystemMacOS = "macos"
	// OperatingSystemLinux is the configuration value for defining Linux.
	OperatingSystemLinux = "linux"
	// OperatingSystemWindows is the configuration value for defining Windows.
	OperatingSystemWindows = "windows"
)

type OperatingSystems map[string]bool //nolint:golint

var (
	// ValidOperatingSystems defines the group of operating systems
	// that can be used as a kubelet node.
	ValidOperatingSystems = OperatingSystems{
		OperatingSystemMacOS:   true,
		OperatingSystemLinux:   false,
		OperatingSystemWindows: false,
	}
)

func (o OperatingSystems) Names() []string { //nolint:golint
	keys := make([]string, 0, len(o))
	for k := range o {
		keys = append(keys, k)
	}
	return keys
}
