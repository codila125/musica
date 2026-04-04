package config

const CurrentVersion = 1

type ServerConfig struct {
	Type     string `yaml:"type"`
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

func (s ServerConfig) Redacted() ServerConfig {
	out := s
	if out.Password != "" {
		out.Password = "***"
	}
	return out
}

type Config struct {
	Version       int            `yaml:"version,omitempty"`
	DefaultServer string         `yaml:"default_server"`
	Servers       []ServerConfig `yaml:"servers"`
}
