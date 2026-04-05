package tg

import (
	"context"
	"fmt"
	"log"
	"omnigram/auth"
	"omnigram/db"
)

// DBSessionStorage implements telegram.SessionStorage using encrypted Postgres storage.
type DBSessionStorage struct {
	repo      *db.Repository
	sessionID string
	encKey    []byte
}

func NewDBSessionStorage(repo *db.Repository, sessionID string, encKey []byte) *DBSessionStorage {
	return &DBSessionStorage{
		repo:      repo,
		sessionID: sessionID,
		encKey:    encKey,
	}
}

func (s *DBSessionStorage) LoadSession(_ context.Context) ([]byte, error) {
	log.Printf("[SessionStorage] Loading session for ID: %s", s.sessionID)
	encData, nonce, err := s.repo.LoadSession(s.sessionID)
	if err != nil {
		return nil, fmt.Errorf("loading session %s from db: %w", s.sessionID, err)
	}
	if encData == nil {
		log.Printf("[SessionStorage] No session found for ID: %s", s.sessionID)
		return nil, nil // no session yet
	}
	plaintext, err := auth.Decrypt(s.encKey, encData, nonce)
	if err != nil {
		return nil, fmt.Errorf("decrypting session %s: %w", s.sessionID, err)
	}
	log.Printf("[SessionStorage] Session decrypted successfully for ID: %s", s.sessionID)
	return plaintext, nil
}

func (s *DBSessionStorage) StoreSession(_ context.Context, data []byte) error {
	log.Printf("[SessionStorage] Storing session for ID: %s (size: %d bytes)", s.sessionID, len(data))
	ciphertext, nonce, err := auth.Encrypt(s.encKey, data)
	if err != nil {
		return fmt.Errorf("encrypting session %s: %w", s.sessionID, err)
	}
	err = s.repo.SaveSession(s.sessionID, ciphertext, nonce)
	if err == nil {
		log.Printf("[SessionStorage] Session stored successfully in DB for ID: %s", s.sessionID)
	}
	return err
}
