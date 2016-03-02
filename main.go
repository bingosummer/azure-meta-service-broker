package main

import (
	"flag"

	conf "github.com/bingosummer/azure-meta-service-broker/config"
	utils "github.com/bingosummer/azure-meta-service-broker/utils"
	webs "github.com/bingosummer/azure-meta-service-broker/web_server"
)

var (
	configPath string
)

func init() {
	// Get Config Path
	defaultConfigPath := utils.GetPath([]string{"data", "config.json"})
	flag.StringVar(&configPath, "c", defaultConfigPath, "use '-c' option to specify the config file path")
}

func main() {
	flag.Parse()

	// Load configuration
	_, err := conf.LoadConfig(configPath)
	if err != nil {
		panic("Error loading config file...")
	}

	// Start Server
	server := webs.NewServer()
	if server == nil {
		panic("Error creating a server...")
	}

	server.Start(conf.GetConfig().Port)
}
