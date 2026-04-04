package config

type ServerConfig struct {
	Type     string `yaml:"type"`
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Config struct {
	DefaultServer string         `yaml:"default_server"`
	Servers       []ServerConfig `yaml:"servers"`
}
