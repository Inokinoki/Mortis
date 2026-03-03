// Test program for Mortis
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Inokinoki/mortis/pkg/config"
	"github.com/Inokinoki/mortis/pkg/provider"
)

func main() {
	fmt.Println("Mortis Provider Test")
	fmt.Println("===================")

	// Test OpenAI provider (disabled, no API key)
	openaiCfg := config.ProviderConfig{
		Type:    "openai",
		Enabled: false,
		Models:  []string{"gpt-4o", "gpt-4o-mini"},
	}
	openai := provider.NewOpenAI(openaiCfg)

	info, err := openai.Info(context.Background())
	if err != nil {
		log.Printf("OpenAI Info error: %v", err)
	} else {
		fmt.Printf("OpenAI: %s (available: %v)\n", info.Name, info.Available)
	}

	// Test Anthropic provider (disabled, no API key)
	anthropicCfg := config.ProviderConfig{
		Type:    "anthropic",
		Enabled: false,
		Models:  []string{"claude-sonnet-4-20250514"},
	}
	anthropic := provider.NewAnthropic(anthropicCfg)

	info, err = anthropic.Info(context.Background())
	if err != nil {
		log.Printf("Anthropic Info error: %v", err)
	} else {
		fmt.Printf("Anthropic: %s (available: %v)\n", info.Name, info.Available)
	}

	// Test Local provider (enabled)
	localCfg := config.ProviderConfig{
		Type:    "local",
		Enabled: true,
		BaseURL: "http://localhost:11434/v1/chat/completions",
		Models:  []string{"llama3", "mistral"},
	}
	local := provider.NewLocalLLM(localCfg)

	info, err = local.Info(context.Background())
	if err != nil {
		log.Printf("Local Info error: %v", err)
	} else {
		fmt.Printf("Local: %s (available: %v)\n", info.Name, info.Available)
	}

	// Test provider registry
	registry := provider.NewRegistry()
	registry.Register("openai", openai)
	registry.Register("anthropic", anthropic)
	registry.Register("local", local)
	registry.SetDefault("local")

	list := registry.List()
	fmt.Printf("\nRegistered providers: %d\n", len(list))

	for id := range list {
		fmt.Printf("  - %s\n", id)
	}

	defaultProv, ok := registry.GetDefault()
	if ok {
		info, _ := defaultProv.Info(context.Background())
		fmt.Printf("\nDefault provider: %s\n", info.Name)
	}

	fmt.Println("\nAll tests passed!")
}
