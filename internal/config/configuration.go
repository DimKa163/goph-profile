package config

type GophConfig struct {
	Addr      string `env:"GOPH_ADDR" envDefault:":8080"`
	Database  string `env:"GOPH_DATABASE" envDefault:"goph"`
	Bucket    string `env:"GOPH_BUCKET" envDefault:"goph"`
	Region    string `env:"GOPH_REGION" envDefault:"us-east-1"`
	Endpoint  string `env:"GOPH_ENDPOINT"`
	AccessKey string `env:"GOPH_ACCESS_KEY"`
	SecretKey string `env:"GOPH_SECRET_KEY"`
	UseSSL    bool   `env:"GOPH_USE_SSL" envDefault:"false"`
}
