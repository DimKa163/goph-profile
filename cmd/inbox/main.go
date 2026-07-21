package main

import (
	"github.com/DimKa163/goph-profile/app"
	"github.com/DimKa163/goph-profile/internal/config"
	"github.com/caarlos0/env/v11"
)

var (
	Name      string
	Version   string
	BuildDate string
	Commit    string
)

func main() {
	var conf config.GophConfig
	if err := env.Parse(&conf); err != nil {
		panic(err)
	}
	if err := app.RunInbox(conf, Name, Version, BuildDate, Commit); err != nil {
		panic(err)
	}
}
