package mount

// Mount describes a filesystem mount for a container.
type Mount struct {
	Type        string   `json:"type"`
	Source      string   `json:"source"`
	Target      string   `json:"target"`
	ReadOnly    bool     `json:"readonly,omitempty"`
	Options     []string `json:"options,omitempty"`
}
