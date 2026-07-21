// Package main starts the HTTP server command.
package main

import (
	"github.com/DimKa163/goph-profile/app"
	"github.com/DimKa163/goph-profile/internal/config"
	"github.com/caarlos0/env/v11"
)

var (
	// Name is the application name set at build time.
	Name string
	// Version is the application version set at build time.
	Version string
	// BuildDate is the application build date set at build time.
	BuildDate string
	// Commit is the source commit set at build time.
	Commit string
)

func main() {
	var conf config.GophConfig
	if err := env.Parse(&conf); err != nil {
		panic(err)
	}
	if err := app.RunServer(conf, Name, Version, BuildDate, Commit); err != nil {
		panic(err)
	}
}
