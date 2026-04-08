package store

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (s *Store) SaveRequest(endpointID, method, path, body, query, ip string, headers map[string]string) (*Request, error) {
	headersJSON, err := json.Marshal(headers)
	if err != nil {
		return nil, fmt.Errorf("marshal headers: %w", err)
	}

	req := &Request{
		ID:         uuid.New().String(),
		EndpointID: endpointID,
		Method:     method,
		Path:       path,
		Headers:    headers,
		Body:       body,
		Query:      query,
		IP:         ip,
		Size:       len(body),
		ReceivedAt: time.Now().UTC(),
	}

	_, err = s.db.Exec(
		`INSERT INTO requests (id, endpoint_id, method, path, headers, body, query, ip, size, received_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.ID, req.EndpointID, req.Method, req.Path, string(headersJSON),
		req.Body, req.Query, req.IP, req.Size, req.ReceivedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("save request: %w", err)
	}

	// Enforce retention: keep only last 100 per endpoint
	_, _ = s.db.Exec(`
		DELETE FROM requests WHERE endpoint_id = ? AND id NOT IN (
			SELECT id FROM requests WHERE endpoint_id = ? ORDER BY received_at DESC LIMIT 100
		)`, endpointID, endpointID)

	return req, nil
}

func (s *Store) ListRequests(endpointID string, limit int) ([]Request, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	rows, err := s.db.Query(
		`SELECT id, endpoint_id, method, path, headers, body, query, ip, size, received_at
		 FROM requests WHERE endpoint_id = ? ORDER BY received_at DESC LIMIT ?`,
		endpointID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list requests: %w", err)
	}
	defer rows.Close()

	var requests []Request
	for rows.Next() {
		var r Request
		var headersStr string
		if err := rows.Scan(&r.ID, &r.EndpointID, &r.Method, &r.Path, &headersStr,
			&r.Body, &r.Query, &r.IP, &r.Size, &r.ReceivedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(headersStr), &r.Headers)
		requests = append(requests, r)
	}
	return requests, rows.Err()
}

func (s *Store) GetRequest(id string) (*Request, error) {
	var r Request
	var headersStr string
	err := s.db.QueryRow(
		`SELECT id, endpoint_id, method, path, headers, body, query, ip, size, received_at
		 FROM requests WHERE id = ?`, id,
	).Scan(&r.ID, &r.EndpointID, &r.Method, &r.Path, &headersStr,
		&r.Body, &r.Query, &r.IP, &r.Size, &r.ReceivedAt)
	if err != nil {
		return nil, fmt.Errorf("get request %s: %w", id, err)
	}
	json.Unmarshal([]byte(headersStr), &r.Headers)
	return &r, nil
}
