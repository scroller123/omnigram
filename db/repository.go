package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Channel struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Username string `json:"username"`
}

type Message struct {
	ID                int       `json:"id"`
	ChannelID         int64     `json:"channel_id"`
	Text              string    `json:"text,omitempty"`
	ContextText       string    `json:"context_text,omitempty"`
	Date              time.Time `json:"date"`
	Embedding         []float32 `json:"embedding,omitempty"`
	EncryptedData     []byte    `json:"encrypted_data,omitempty"`
	Nonce             []byte    `json:"nonce,omitempty"`
	EncryptedContext  []byte    `json:"encrypted_context,omitempty"`
	NonceContext      []byte    `json:"nonce_context,omitempty"`
	ChannelUsername   string    `json:"channel_username,omitempty"`
}

type EnrichmentTask struct {
	ID                int       `json:"id"`
	DialogID          int64     `json:"dialog_id"`
	SessionID         string    `json:"session_id"`
	Status            string    `json:"status"`
	TotalMessages     int       `json:"total_messages"`
	ProcessedMessages int       `json:"processed_messages"`
	LastMessageID     int       `json:"last_message_id"`
	ErrorMessage      string    `json:"error_message,omitempty"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Repository struct {
	DB *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) InsertChannel(id int64, title, username string) error {
	query := `INSERT INTO channels (id, title, username) VALUES ($1, $2, $3)
              ON CONFLICT(id) DO UPDATE SET title=EXCLUDED.title, username=EXCLUDED.username`
	_, err := r.DB.Exec(query, id, title, username)
	return err
}

func (r *Repository) InsertMessage(id int, channelID int64, date time.Time, embedding []float32, encryptedData []byte, nonce []byte, encContext []byte, nonceContext []byte) error {
	var embStr sql.NullString
	if len(embedding) > 0 {
		embStr.String = formatVector(embedding)
		embStr.Valid = true
	}

	query := `INSERT INTO messages (id, channel_id, date, embedding, encrypted_data, nonce, encrypted_context, nonce_context) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
              ON CONFLICT(id, channel_id) DO UPDATE SET 
              date=EXCLUDED.date, 
              embedding=COALESCE(EXCLUDED.embedding, messages.embedding),
              encrypted_data=EXCLUDED.encrypted_data,
              nonce=EXCLUDED.nonce,
              encrypted_context=EXCLUDED.encrypted_context,
              nonce_context=EXCLUDED.nonce_context`
	_, err := r.DB.Exec(query, id, channelID, date, embStr, encryptedData, nonce, encContext, nonceContext)
	return err
}

func (r *Repository) UpdateMessageEmbedding(id int, channelID int64, embedding []float32) error {
	query := `UPDATE messages SET embedding = $1 WHERE id = $2 AND channel_id = $3`
	_, err := r.DB.Exec(query, formatVector(embedding), id, channelID)
	return err
}

func (r *Repository) SearchMessages(queryEmbedding []float32, limit int) ([]Message, error) {
	// Using cosine distance <=> (smaller is more similar)
	query := `SELECT m.id, m.channel_id, m.date, m.encrypted_data, m.nonce, m.encrypted_context, m.nonce_context, c.username 
              FROM messages m
              LEFT JOIN channels c ON m.channel_id = c.id
              WHERE m.embedding IS NOT NULL 
              ORDER BY m.embedding <=> $1 LIMIT $2`
	rows, err := r.DB.Query(query, formatVector(queryEmbedding), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ChannelID, &m.Date, &m.EncryptedData, &m.Nonce, &m.EncryptedContext, &m.NonceContext, &m.ChannelUsername); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}

func formatVector(v []float32) string {
	var s []string
	for _, f := range v {
		s = append(s, fmt.Sprintf("%f", f))
	}
	return "[" + strings.Join(s, ",") + "]"
}

func (r *Repository) GetChannels() ([]Channel, error) {
	rows, err := r.DB.Query("SELECT id, title, username FROM channels")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []Channel
	for rows.Next() {
		var c Channel
		if err := rows.Scan(&c.ID, &c.Title, &c.Username); err != nil {
			return nil, err
		}
		channels = append(channels, c)
	}
	return channels, nil
}

func (r *Repository) GetMessages(channelID int64) ([]Message, error) {
	rows, err := r.DB.Query(`SELECT m.id, m.channel_id, m.date, m.encrypted_data, m.nonce, m.encrypted_context, m.nonce_context, c.username 
                             FROM messages m 
                             LEFT JOIN channels c ON m.channel_id = c.id 
                             WHERE m.channel_id = $1 ORDER BY m.date DESC`, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ChannelID, &m.Date, &m.EncryptedData, &m.Nonce, &m.EncryptedContext, &m.NonceContext, &m.ChannelUsername); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}
func (r *Repository) UpsertEnrichmentTask(dialogID int64, sessionID string, status string, total int) error {
	query := `INSERT INTO enrichment_tasks (dialog_id, session_id, status, total_messages, updated_at) 
              VALUES ($1, $2, $3, $4, NOW())
              ON CONFLICT(dialog_id, session_id) DO UPDATE SET 
              status=EXCLUDED.status, 
              total_messages=EXCLUDED.total_messages,
              updated_at=NOW()`
	_, err := r.DB.Exec(query, dialogID, sessionID, status, total)
	return err
}

func (r *Repository) UpdateEnrichmentProgress(dialogID int64, sessionID string, processed int, lastID int, status string, eraMsg string) error {
	query := `UPDATE enrichment_tasks SET 
              processed_messages = $1, 
              last_message_id = $2, 
              status = $3, 
              error_message = $4,
              updated_at = NOW() 
              WHERE dialog_id = $5 AND session_id = $6`
	_, err := r.DB.Exec(query, processed, lastID, status, eraMsg, dialogID, sessionID)
	return err
}

func (r *Repository) GetEnrichmentTask(dialogID int64, sessionID string) (*EnrichmentTask, error) {
	query := `SELECT id, dialog_id, session_id, status, total_messages, processed_messages, last_message_id, error_message, updated_at 
              FROM enrichment_tasks WHERE dialog_id = $1 AND session_id = $2`
	var t EnrichmentTask
	var errMsg sql.NullString
	err := r.DB.QueryRow(query, dialogID, sessionID).Scan(&t.ID, &t.DialogID, &t.SessionID, &t.Status, &t.TotalMessages, &t.ProcessedMessages, &t.LastMessageID, &errMsg, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if errMsg.Valid {
		t.ErrorMessage = errMsg.String
	}
	return &t, nil
}

func (r *Repository) GetAllEnrichmentTasksBySession(sessionID string) ([]EnrichmentTask, error) {
	query := `SELECT id, dialog_id, session_id, status, total_messages, processed_messages, last_message_id, error_message, updated_at 
              FROM enrichment_tasks WHERE session_id = $1`
	rows, err := r.DB.Query(query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []EnrichmentTask
	for rows.Next() {
		var t EnrichmentTask
		var errMsg sql.NullString
		if err := rows.Scan(&t.ID, &t.DialogID, &t.SessionID, &t.Status, &t.TotalMessages, &t.ProcessedMessages, &t.LastMessageID, &errMsg, &t.UpdatedAt); err != nil {
			return nil, err
		}
		if errMsg.Valid {
			t.ErrorMessage = errMsg.String
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}
func (r *Repository) GetAllProcessingTasks() ([]EnrichmentTask, error) {
	query := `SELECT id, dialog_id, session_id, status, total_messages, processed_messages, last_message_id, error_message, updated_at 
              FROM enrichment_tasks WHERE status = 'processing'`
	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []EnrichmentTask
	for rows.Next() {
		var t EnrichmentTask
		var errMsg sql.NullString
		if err := rows.Scan(&t.ID, &t.DialogID, &t.SessionID, &t.Status, &t.TotalMessages, &t.ProcessedMessages, &t.LastMessageID, &errMsg, &t.UpdatedAt); err != nil {
			return nil, err
		}
		if errMsg.Valid {
			t.ErrorMessage = errMsg.String
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}
