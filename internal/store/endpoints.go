package store

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (s *Store) CreateEndpoint(apiKey, name string) (*Endpoint, error) {
	ep := &Endpoint{
		ID:        uuid.New().String()[:8],
		Name:      name,
		APIKey:    apiKey,
		CreatedAt: time.Now().UTC(),
	}

	_, err := s.db.Exec(
		"INSERT INTO endpoints (id, name, api_key, created_at) VALUES (?, ?, ?, ?)",
		ep.ID, ep.Name, ep.APIKey, ep.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create endpoint: %w", err)
	}
	return ep, nil
}

func (s *Store) GetEndpoint(id string) (*Endpoint, error) {
	ep := &Endpoint{}
	err := s.db.QueryRow(
		"SELECT id, name, api_key, created_at FROM endpoints WHERE id = ?", id,
	).Scan(&ep.ID, &ep.Name, &ep.APIKey, &ep.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get endpoint %s: %w", id, err)
	}
	return ep, nil
}

func (s *Store) ListEndpoints(apiKey string) ([]Endpoint, error) {
	rows, err := s.db.Query(
		"SELECT id, name, api_key, created_at FROM endpoints WHERE api_key = ? ORDER BY created_at DESC",
		apiKey,
	)
	if err != nil {
		return nil, fmt.Errorf("list endpoints: %w", err)
	}
	defer rows.Close()

	var endpoints []Endpoint
	for rows.Next() {
		var ep Endpoint
		if err := rows.Scan(&ep.ID, &ep.Name, &ep.APIKey, &ep.CreatedAt); err != nil {
			return nil, err
		}
		endpoints = append(endpoints, ep)
	}
	return endpoints, rows.Err()
}

func (s *Store) DeleteEndpoint(id, apiKey string) error {
	result, err := s.db.Exec("DELETE FROM endpoints WHERE id = ? AND api_key = ?", id, apiKey)
	if err != nil {
		return fmt.Errorf("delete endpoint: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("endpoint %s not found or not owned by this API key", id)
	}
	return nil
}

func (s *Store) CountEndpoints(apiKey string) (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM endpoints WHERE api_key = ?", apiKey).Scan(&count)
	return count, err
}
