package config

import "time"

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
type GophInboxConfig struct {
	Database        string        `env:"GOPH_DATABASE" envDefault:"goph"`
	Brokers         string        `env:"GOPH_BROKERS"`
	BatchMaxSize    int           `env:"GOPH_BATCH_MAX_SIZE" envDefault:"1024"`
	DeliveryTimeout time.Duration `env:"GOPH_DELIVERY_TIMEOUT" envDefault:"30s"`
	Group           string        `env:"GOPH_GROUP" envDefault:"profile"`
	AutoCommit      bool          `env:"GOPH_AUTOCOMMIT" envDefault:"false"`
	Bucket          string        `env:"GOPH_BUCKET" envDefault:"goph"`
	Region          string        `env:"GOPH_REGION" envDefault:"us-east-1"`
	Endpoint        string        `env:"GOPH_ENDPOINT"`
	AccessKey       string        `env:"GOPH_ACCESS_KEY"`
	SecretKey       string        `env:"GOPH_SECRET_KEY"`
	UseSSL          bool          `env:"GOPH_USE_SSL" envDefault:"false"`
}
type GophOutboxConfig struct {
	Database        string        `env:"GOPH_DATABASE" envDefault:"goph"`
	Brokers         string        `env:"GOPH_BROKERS"`
	BatchMaxSize    int           `env:"GOPH_BATCH_MAX_SIZE" envDefault:"1024"`
	DeliveryTimeout time.Duration `env:"GOPH_DELIVERY_TIMEOUT" envDefault:"10s"`
	BatchSize       int           `env:"GOPH_BATCH_SIZE" envDefault:"100"`
	WaitTime        time.Duration `env:"GOPH_WAIT_TIME" envDefault:"10"`
	Workers         int           `env:"GOPH_WORKERS" envDefault:"0"`
}
