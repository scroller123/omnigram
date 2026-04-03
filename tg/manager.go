package tg

import (
	"context"
	"fmt"
	"sync"
	"telegai/ai"
	"telegai/auth"
	"telegai/db"
)

type Manager struct {
	apps map[string]*App
	mu   sync.RWMutex
	ctx  context.Context

	apiID            int
	apiHash          string
	encryptionSecret []byte
	repo             *db.Repository
	gemini           *ai.GeminiClient
	embedder         ai.Embedder
}

func NewManager(ctx context.Context, apiID int, apiHash string, encryptionSecret []byte, repo *db.Repository, gemini *ai.GeminiClient, embedder ai.Embedder) *Manager {
	return &Manager{
		apps:             make(map[string]*App),
		ctx:              ctx,
		apiID:            apiID,
		apiHash:          apiHash,
		encryptionSecret: encryptionSecret,
		repo:             repo,
		gemini:           gemini,
		embedder:         embedder,
	}
}

func (m *Manager) GetApp(ctx context.Context, sessionID string) (*App, error) {
	m.mu.RLock()
	app, ok := m.apps[sessionID]
	m.mu.RUnlock()

	if ok {
		return app, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double check
	if app, ok := m.apps[sessionID]; ok {
		return app, nil
	}

	// Derive key for this session
	derivedKey := auth.DeriveKey(m.encryptionSecret, sessionID)
	
	// Create storage for this session
	storage := NewDBSessionStorage(m.repo, sessionID, derivedKey)
	
	app = NewApp(m.apiID, m.apiHash, storage, m.repo, m.gemini, m.embedder)
	
	// Start the app in a background goroutine tied to the manager's context
	go func() {
		fmt.Printf("Starting Telegram client for session %s\n", sessionID)
		if err := app.Start(m.ctx); err != nil {
			fmt.Printf("App for session %s stopped: %v\n", sessionID, err)
		}
	}()

	m.apps[sessionID] = app
	return app, nil
}
