package main

import (
	"flag"
	"log"
	"service1/cmd/service1/config"
	"service1/cmd/service1/server"
)

func main() {
	configPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	config, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	server := server.NewServer(config)

	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
