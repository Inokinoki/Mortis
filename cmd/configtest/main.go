// Config test program
package main

import (
	"fmt"
	"os"

	"github.com/Inokinoki/mortis/pkg/config"
)

func main() {
	// Test default config
	fmt.Println("=== Testing Default Config ===")
	cfg := config.DefaultConfig()

	fmt.Printf("Default config has %d providers\n", len(cfg.Providers))
	for id := range cfg.Providers {
		fmt.Printf("  - %s\n", id)
	}

	// Test Save
	fmt.Println("\n=== Testing Save ===")
	tmpFile := "/tmp/mortis-test-config.json"
	if err := cfg.Save(tmpFile); err != nil {
		fmt.Printf("Error saving: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Saved to %s\n", tmpFile)

	// Test Load
	fmt.Println("\n=== Testing Load ===")
	loaded, err := config.LoadConfig(tmpFile)
	if err != nil {
		fmt.Printf("Error loading: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded config: gateway name = %s\n", loaded.Gateway.Name)
	fmt.Printf("Loaded config: %d providers\n", len(loaded.Providers))

	// Clean up
	os.Remove(tmpFile)

	fmt.Println("\n✓ All config tests passed!")
}
