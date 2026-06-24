package config

type GophConfig struct {
	Addr     string `env:"GOPH_ADDR" envDefault:":8080"`
	Database string `env:"GOPH_DATABASE" envDefault:"goph"`
}
