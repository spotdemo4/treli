package settings

type proc struct {
	Shell       string   `yaml:"onstart,omitempty"`
	Cwd         string   `yaml:"cwd,omitempty"`
	Exts        []string `yaml:"exts,omitempty"`
	AutoStart   bool     `yaml:"autostart,omitempty"`
	AutoRestart bool     `yaml:"autorestart,omitempty"`
}

type Settings struct {
	Procs map[string]proc `yaml:"procs"`
}
