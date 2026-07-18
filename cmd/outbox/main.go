package main

import (
	"github.com/DimKa163/goph-profile/app"
	"github.com/DimKa163/goph-profile/internal/config"
	"github.com/caarlos0/env/v11"
)

func main() {
	var conf config.GophConfig
	if err := env.Parse(&conf); err != nil {
		panic(err)
	}
	name := "goph-outbox"
	version := "1.0.0"
	buildDate := "1970-01-01T00:00:00Z"
	if err := app.RunOutbox(conf, name, version, buildDate); err != nil {
		panic(err)
	}
}
