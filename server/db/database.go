package db

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

func InitDB(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	err = createTables(db)
	if err != nil {
		return nil, err
	}

	log.Println("Database initialized successfully.")
	return db, nil
}

func createTables(db *sql.DB) error {
	createExtension := `CREATE EXTENSION IF NOT EXISTS vector;`
	_, err := db.Exec(createExtension)
	if err != nil {
		return err
	}

	createChannelsTable := `
	CREATE TABLE IF NOT EXISTS channels (
		id BIGINT PRIMARY KEY,
		title TEXT NOT NULL,
		username TEXT
	);`

	createMessagesTable := `
	CREATE TABLE IF NOT EXISTS messages (
		id BIGINT,
		channel_id BIGINT,
		message TEXT,
		context_text TEXT,
		date TIMESTAMP,
		embedding vector(768),
		PRIMARY KEY(id, channel_id),
		FOREIGN KEY(channel_id) REFERENCES channels(id)
	);`

	_, err = db.Exec(createChannelsTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(createMessagesTable)
	if err != nil {
		return err
	}

	// Migration: ensure embedding and context_text columns exist
	alterMessagesTable := `
		ALTER TABLE messages ADD COLUMN IF NOT EXISTS embedding vector(768);
		ALTER TABLE messages ADD COLUMN IF NOT EXISTS context_text TEXT;
	`
	_, err = db.Exec(alterMessagesTable)
	return err
}
