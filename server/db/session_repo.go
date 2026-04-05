package db

import "database/sql"

func (r *Repository) LoadSession(id string) (encryptedData, nonce []byte, err error) {
	err = r.DB.QueryRow(
		`SELECT encrypted_data, nonce FROM tg_sessions WHERE id = $1`, id,
	).Scan(&encryptedData, &nonce)
	if err == sql.ErrNoRows {
		return nil, nil, nil
	}
	return
}

func (r *Repository) SaveSession(id string, encryptedData, nonce []byte) error {
	_, err := r.DB.Exec(`
		INSERT INTO tg_sessions (id, encrypted_data, nonce, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT(id) DO UPDATE SET encrypted_data=EXCLUDED.encrypted_data, nonce=EXCLUDED.nonce, updated_at=NOW()
	`, id, encryptedData, nonce)
	return err
}
