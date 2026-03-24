package main

import (
	"flag"
	"log"
	service1 "service1/service"
)

func main() {
	configPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	config, err := service1.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	server := service1.NewServer(config)

	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
