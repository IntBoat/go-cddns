package main

import (
	"flag"
	"github.com/spf13/viper"
)

func parseCommandLineFlags() {
	var filename string
	flag.StringVar(&filename, "config", "./config.json", "Path to config file")

	flag.Parse()

	viper.SetConfigFile(filename)
}
