package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"omnigram/ai"

	"omnigram/api"
	"omnigram/auth"
	"omnigram/db"
	"omnigram/tg"

	"github.com/joho/godotenv"
)

// @title Omnigram API
// @version 1.0
// @description API for managing Telegram channel grabs.
// @host localhost:8080
// @BasePath /

func main() {
	// Load .env from the repo root (one level up when running from server/ locally)
	// In Docker, env vars are injected via docker-compose env_file; godotenv errors are ignored.
	_ = godotenv.Load("../.env")
	_ = godotenv.Load() // also try CWD for flexibility

	apiIDStr := os.Getenv("API_ID")
	apiHash := os.Getenv("API_HASH")

	if apiIDStr == "" || apiHash == "" {
		log.Fatal("API_ID and API_HASH environment variables are required. Get them from https://my.telegram.org")
	}

	/*
		TIP:
		You can generate a valid ENCRYPTION_SECRET by running: openssl rand -hex 32 in your terminal.
	*/

	encryptionSecretHex := os.Getenv("ENCRYPTION_SECRET")
	if encryptionSecretHex == "" {
		log.Fatal("ENCRYPTION_SECRET environment variable is required (64 hex characters for AES-256).")
	}
	encKey, err := auth.NewAESKeyFromHex(encryptionSecretHex)
	if err != nil {
		log.Fatalf("Invalid ENCRYPTION_SECRET: %v", err)
	}

	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("JWT_SECRET environment variable is required.")
	}

	apiID, err := strconv.Atoi(apiIDStr)
	if err != nil {
		log.Fatalf("Invalid API_ID: %v", err)
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		host := os.Getenv("DB_HOST")
		if host == "" {
			host = "localhost"
		}
		user := os.Getenv("POSTGRES_USER")
		if user == "" {
			user = "postgres"
		}
		pass := os.Getenv("POSTGRES_PASSWORD")
		if pass == "" {
			pass = "postgres"
		}
		dbName := os.Getenv("POSTGRES_DB")
		if dbName == "" {
			dbName = "omnigram"
		}
		databaseURL = fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable", user, pass, host, dbName)
	}

	database, err := db.InitDB(databaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Run migrations
	if err := db.RunMigrations(database); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	repo := db.NewRepository(database)

	var gemini *ai.GeminiClient
	geminiKey := os.Getenv("GEMINI_API_KEY")
	if geminiKey != "" {
		g, err := ai.NewGeminiClient(context.Background(), geminiKey)
		if err != nil {
			log.Printf("Failed to initialize Gemini: %v", err)
		} else {
			gemini = g
			defer gemini.Close()
		}
	}

	var embedder ai.Embedder
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL != "" {
		// embedder = ai.NewLocalEmbeddingClient(ollamaURL, "all-minilm")
		embedder = ai.NewLocalEmbeddingClient(ollamaURL, "nomic-fast")
		//embedder = ai.NewLocalEmbeddingClient(ollamaURL /*"nomic-embed-text"*/, "qwen3-embedding:0.6b")
		log.Printf("Using local embedding (Ollama) at %s", ollamaURL)
	} else if gemini != nil {
		embedder = gemini
		log.Printf("OLLAMA_URL not set, falling back to Gemini embeddings")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// New session manager handles per-session independently encrypted clients
	manager := tg.NewManager(ctx, apiID, apiHash, encKey, repo, gemini, embedder)

	// Resume any processing tasks from previous run
	if err := manager.ResumeProcessingTasks(); err != nil {
		log.Printf("Failed to resume tasks: %v", err)
	}

	server := api.NewServer(repo, manager)

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: server.Router(),
	}

	go func() {
		fmt.Println("API server listening on http://localhost:8080")
		fmt.Println("Swagger UI available at http://localhost:8080/swagger/")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("API server error: %v", err)
		}
	}()

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
	<-sigint
	fmt.Println("\nShutting down...")

	cancel()
	httpServer.Shutdown(context.Background())
}
