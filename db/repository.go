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
	ID          int       `json:"id"`
	ChannelID   int64     `json:"channel_id"`
	Text        string    `json:"text"`
	ContextText string    `json:"context_text"`
	Date        time.Time `json:"date"`
	Embedding   []float32 `json:"embedding,omitempty"`
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

func (r *Repository) InsertMessage(id int, channelID int64, text string, contextText string, date time.Time, embedding []float32) error {
	var embStr sql.NullString
	if len(embedding) > 0 {
		embStr.String = formatVector(embedding)
		embStr.Valid = true
	}

	query := `INSERT INTO messages (id, channel_id, message, context_text, date, embedding) VALUES ($1, $2, $3, $4, $5, $6)
              ON CONFLICT(id, channel_id) DO UPDATE SET message=EXCLUDED.message, context_text=EXCLUDED.context_text, date=EXCLUDED.date, embedding=COALESCE(EXCLUDED.embedding, messages.embedding)`
	_, err := r.DB.Exec(query, id, channelID, text, contextText, date, embStr)
	return err
}

func (r *Repository) UpdateMessageEmbedding(id int, channelID int64, embedding []float32) error {
	query := `UPDATE messages SET embedding = $1 WHERE id = $2 AND channel_id = $3`
	_, err := r.DB.Exec(query, formatVector(embedding), id, channelID)
	return err
}

func (r *Repository) SearchMessages(queryEmbedding []float32, limit int) ([]Message, error) {
	// Using cosine distance <=> (smaller is more similar)
	query := `SELECT id, channel_id, message, context_text, date FROM messages 
              WHERE embedding IS NOT NULL 
              ORDER BY embedding <=> $1 LIMIT $2`
	rows, err := r.DB.Query(query, formatVector(queryEmbedding), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ChannelID, &m.Text, &m.ContextText, &m.Date); err != nil {
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
	rows, err := r.DB.Query("SELECT id, channel_id, message, context_text, date FROM messages WHERE channel_id = $1 ORDER BY date DESC", channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ChannelID, &m.Text, &m.ContextText, &m.Date); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}
