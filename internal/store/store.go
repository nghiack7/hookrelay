package store

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Endpoint struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	APIKey    string    `json:"api_key"`
	CreatedAt time.Time `json:"created_at"`
}

type Request struct {
	ID         string            `json:"id"`
	EndpointID string            `json:"endpoint_id"`
	Method     string            `json:"method"`
	Path       string            `json:"path"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Query      string            `json:"query"`
	IP         string            `json:"ip"`
	Size       int               `json:"size"`
	ReceivedAt time.Time         `json:"received_at"`
}

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS endpoints (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL DEFAULT '',
			api_key TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_endpoints_api_key ON endpoints(api_key);

		CREATE TABLE IF NOT EXISTS requests (
			id TEXT PRIMARY KEY,
			endpoint_id TEXT NOT NULL,
			method TEXT NOT NULL,
			path TEXT NOT NULL DEFAULT '',
			headers TEXT NOT NULL DEFAULT '{}',
			body TEXT NOT NULL DEFAULT '',
			query TEXT NOT NULL DEFAULT '',
			ip TEXT NOT NULL DEFAULT '',
			size INTEGER NOT NULL DEFAULT 0,
			received_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (endpoint_id) REFERENCES endpoints(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_requests_endpoint_id ON requests(endpoint_id);
		CREATE INDEX IF NOT EXISTS idx_requests_received_at ON requests(received_at);

		CREATE TABLE IF NOT EXISTS users (
			api_key TEXT PRIMARY KEY,
			plan TEXT NOT NULL DEFAULT 'free',
			stripe_id TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_users_stripe_id ON users(stripe_id);
	`)
	return err
}
