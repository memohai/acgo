package network

// Network holds the metadata for a container network.
type Network struct {
	ID     string
	Name   string
	Scope  string
	Driver string
	Labels map[string]string
	IPAM   IPAMConfig
}

// IPAMConfig holds IP address management settings.
type IPAMConfig struct {
	Subnet  string
	Gateway string
}
