package main

import (
	"context"
	"fmt"
	"log"

	"github.com/denisAlshanov/stPlaner/internal/config"
	"github.com/denisAlshanov/stPlaner/internal/services/telegram"
)

func main() {
	fmt.Println("Telegram Client Test")
	fmt.Println("===================")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Check if API credentials are configured
	if cfg.Telegram.APIId == 0 || cfg.Telegram.APIHash == "" {
		fmt.Println("TELEGRAM_API_ID and TELEGRAM_API_HASH not set - will use simplified client")
		fmt.Println("Get credentials from https://my.telegram.org to enable real media downloads")
	}

	fmt.Printf("API ID: %d\n", cfg.Telegram.APIId)
	fmt.Printf("API Hash: %s\n", cfg.Telegram.APIHash)
	fmt.Println()

	// Create Telegram client
	client, err := telegram.NewClient(&cfg.Telegram)
	if err != nil {
		log.Fatalf("Failed to create Telegram client: %v", err)
	}

	// Try to connect
	fmt.Println("Attempting to connect...")
	if err := client.Connect(context.Background()); err != nil {
		log.Fatalf("Connection failed: %v", err)
	}

	fmt.Println("Connection successful!")
	fmt.Println("Service can now grab media from public Telegram channels.")

	// Test channel resolution
	fmt.Println("\nTesting channel resolution...")
	_, _, err = client.ParseTelegramLink("https://t.me/telegram/123")
	if err != nil {
		fmt.Printf("Link parsing test failed: %v\n", err)
	} else {
		fmt.Println("Link parsing test successful!")
	}

	client.Close()
}
