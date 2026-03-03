// Mortis - Personal AI Gateway in Go
// A multi-provider LLM gateway inspired by Moltis
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Inokinoki/mortis/pkg/config"
	"github.com/Inokinoki/mortis/pkg/gateway"
	"github.com/joho/godotenv"
)

var (
	// Version is the application version
	Version = "0.1.0"
	// ConfigFile is the path to the config file
	ConfigFile = flag.String("config", "", "Path to config file")
	// ConfigDir is the directory for config files
	ConfigDir = flag.String("config-dir", "", "Directory for config files")
	// DataDir is the directory for data files
	DataDir = flag.String("data-dir", "", "Directory for data files")
	// VersionFlag prints the version
	VersionFlag = flag.Bool("version", false, "Print version information")
)

func main() {
	flag.Usage = func() {
		fmt.Printf("Mortis v%s - Personal AI Gateway\n\n", Version)
		fmt.Printf("Usage: mortis [options]\n\n")
		fmt.Printf("Options:\n")
		flag.PrintDefaults()
		fmt.Printf("\nEnvironment variables:\n")
		fmt.Printf("  MORTIS_CONFIG_DIR   Directory for config files\n")
		fmt.Printf("  MORTIS_DATA_DIR      Directory for data files\n")
		fmt.Printf("\nDocumentation: https://docs.mortis.org\n")
		fmt.Printf("License: MIT\n")
	}

	flag.Parse()

	if *VersionFlag {
		fmt.Printf("Mortis v%s\n", Version)
		os.Exit(0)
	}

	// Load .env file
	godotenv.Load()

	// Resolve config directory
	configDir := *ConfigDir
	if configDir == "" {
		configDir = os.Getenv("MORTIS_CONFIG_DIR")
		if configDir == "" {
			homeDir, _ := os.UserHomeDir()
			configDir = fmt.Sprintf("%s/.mortis", homeDir)
		}
	}

	// Resolve data directory
	dataDir := *DataDir
	if dataDir == "" {
		dataDir = os.Getenv("MORTIS_DATA_DIR")
		if dataDir == "" {
			homeDir, _ := os.UserHomeDir()
			dataDir = fmt.Sprintf("%s/.mortis", homeDir)
		}
	}

	// Ensure directories exist
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		log.Fatalf("Failed to create config directory: %v", err)
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Resolve config file
	configFile := *ConfigFile
	if configFile == "" {
		configFile = fmt.Sprintf("%s/mortis.json", configDir)
	}

	// Load configuration
	var cfg *config.Config
	if _, err := os.Stat(configFile); err == nil {
		// Config exists, load it
		var err error
		cfg, err = config.LoadConfig(configFile)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
	} else {
		// Create default config
		cfg = config.DefaultConfig()
		if err := cfg.Save(configFile); err != nil {
			log.Fatalf("Failed to save config: %v", err)
		}
		log.Printf("Created default config at %s", configFile)
	}

	// Create gateway server
	server := gateway.New(cfg)

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	// Start server
	if err := server.Start(ctx); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
