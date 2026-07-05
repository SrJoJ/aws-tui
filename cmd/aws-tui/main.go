package main

import (
	"log"

	"github.com/SrJoJ/aws-tui/internal/tui"
	"github.com/SrJoJ/aws-tui/pkg/config"
	"github.com/SrJoJ/aws-tui/pkg/provider"
	"github.com/SrJoJ/aws-tui/pkg/providers/ec2"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize registry
	reg := provider.NewRegistry()

	// Register core provider modules based on config
	if ec2Cfg, ok := cfg.Providers["ec2"]; ok && ec2Cfg.Type == "builtin" {
		ec2Prov := ec2.NewProvider()
		if err := reg.Register(ec2Prov); err != nil {
			log.Fatalf("failed to register ec2 provider: %v", err)
		}
	}

	// Instantiate and run the TUI
	app := tui.NewApp(reg, cfg)
	if err := app.Run(); err != nil {
		log.Fatalf("aws-tui exited with error: %v", err)
	}
}
