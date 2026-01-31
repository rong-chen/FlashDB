package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	"flash-db/internal/crypto"
	"flash-db/models"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// Storage handles local SQLite storage for connection configurations
type Storage struct {
	db     *sql.DB
	crypto *crypto.AES256
}

// New creates a new Storage instance
func New(encryptionKey string) (*Storage, error) {
	// Get user config directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	// Create FlashDB directory
	dbDir := filepath.Join(configDir, "FlashDB")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, err
	}

	// Open SQLite database
	dbPath := filepath.Join(dbDir, "flashdb.sqlite")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	s := &Storage{
		db:     db,
		crypto: crypto.NewAES256(encryptionKey),
	}

	// Initialize schema
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

// initSchema creates the database tables
func (s *Storage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS connections (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		host TEXT NOT NULL,
		port INTEGER NOT NULL,
		username TEXT NOT NULL,
		password TEXT NOT NULL,
		database_name TEXT,
		ssl_mode TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_connections_name ON connections(name);
	`
	_, err := s.db.Exec(schema)
	return err
}

// Close closes the database connection
func (s *Storage) Close() error {
	return s.db.Close()
}

// CreateConnection creates a new connection
func (s *Storage) CreateConnection(req models.ConnectionRequest) (*models.Connection, error) {
	// Encrypt password
	encryptedPassword, err := s.crypto.Encrypt(req.Password)
	if err != nil {
		return nil, err
	}

	conn := &models.Connection{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Type:      req.Type,
		Host:      req.Host,
		Port:      req.Port,
		Username:  req.Username,
		Password:  encryptedPassword,
		Database:  req.Database,
		SSLMode:   req.SSLMode,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = s.db.Exec(`
		INSERT INTO connections (id, name, type, host, port, username, password, database_name, ssl_mode, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, conn.ID, conn.Name, conn.Type, conn.Host, conn.Port, conn.Username, conn.Password, conn.Database, conn.SSLMode, conn.CreatedAt, conn.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return conn, nil
}

// GetConnection retrieves a connection by ID
func (s *Storage) GetConnection(id string) (*models.Connection, error) {
	var conn models.Connection
	err := s.db.QueryRow(`
		SELECT id, name, type, host, port, username, password, database_name, ssl_mode, created_at, updated_at
		FROM connections WHERE id = ?
	`, id).Scan(&conn.ID, &conn.Name, &conn.Type, &conn.Host, &conn.Port, &conn.Username, &conn.Password, &conn.Database, &conn.SSLMode, &conn.CreatedAt, &conn.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return &conn, nil
}

// GetConnectionWithPassword retrieves a connection with decrypted password
func (s *Storage) GetConnectionWithPassword(id string) (*models.Connection, error) {
	conn, err := s.GetConnection(id)
	if err != nil {
		return nil, err
	}

	// Decrypt password
	decryptedPassword, err := s.crypto.Decrypt(conn.Password)
	if err != nil {
		return nil, err
	}
	conn.Password = decryptedPassword

	return conn, nil
}

// ListConnections returns all connections (without passwords)
func (s *Storage) ListConnections() ([]models.ConnectionInfo, error) {
	rows, err := s.db.Query(`
		SELECT id, name, type, host, port, username, database_name, ssl_mode, created_at, updated_at
		FROM connections ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connections []models.ConnectionInfo
	for rows.Next() {
		var conn models.ConnectionInfo
		err := rows.Scan(&conn.ID, &conn.Name, &conn.Type, &conn.Host, &conn.Port, &conn.Username, &conn.Database, &conn.SSLMode, &conn.CreatedAt, &conn.UpdatedAt)
		if err != nil {
			return nil, err
		}
		connections = append(connections, conn)
	}

	return connections, nil
}

// UpdateConnection updates an existing connection
func (s *Storage) UpdateConnection(id string, req models.ConnectionRequest) (*models.Connection, error) {
	now := time.Now()

	// If password is empty, update without changing password
	if req.Password == "" {
		_, err := s.db.Exec(`
			UPDATE connections SET name = ?, type = ?, host = ?, port = ?, username = ?, database_name = ?, ssl_mode = ?, updated_at = ?
			WHERE id = ?
		`, req.Name, req.Type, req.Host, req.Port, req.Username, req.Database, req.SSLMode, now, id)

		if err != nil {
			return nil, err
		}
	} else {
		// Encrypt new password
		encryptedPassword, err := s.crypto.Encrypt(req.Password)
		if err != nil {
			return nil, err
		}

		_, err = s.db.Exec(`
			UPDATE connections SET name = ?, type = ?, host = ?, port = ?, username = ?, password = ?, database_name = ?, ssl_mode = ?, updated_at = ?
			WHERE id = ?
		`, req.Name, req.Type, req.Host, req.Port, req.Username, encryptedPassword, req.Database, req.SSLMode, now, id)

		if err != nil {
			return nil, err
		}
	}

	return s.GetConnection(id)
}

// DeleteConnection deletes a connection
func (s *Storage) DeleteConnection(id string) error {
	_, err := s.db.Exec("DELETE FROM connections WHERE id = ?", id)
	return err
}
