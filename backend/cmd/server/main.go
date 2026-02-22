package main

import (
	"log"
	"net/http"

	"github.com/mdp/ai-chat-platform/backend/internal/api"
	"github.com/mdp/ai-chat-platform/backend/internal/config"
	"github.com/mdp/ai-chat-platform/backend/internal/domain"
	"github.com/mdp/ai-chat-platform/backend/internal/store"
)

func main() {
	log.Println("Starting AI Chat Platform Backend...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dbStore, err := store.NewPostgresStore(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Load domain adapter from config (nil if no path set — generic mode)
	adapter, err := domain.LoadAdapter(cfg.AdapterConfigPath)
	if err != nil {
		log.Fatalf("Failed to load adapter config: %v", err)
	}
	if adapter != nil {
		log.Printf("Domain adapter loaded: %s (%s) with %d tools", adapter.Domain(), adapter.DisplayName(), len(adapter.Tools()))
	} else {
		log.Println("No adapter config — running in generic mode")
	}

	handler := api.NewHandler(dbStore, cfg, adapter)

	log.Printf("Server listening on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
