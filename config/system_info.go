package config

type SystemInfo struct {
	DefaultUser struct {
		Name string `yaml:"name"`
	} `yaml:"default_user"`
}
