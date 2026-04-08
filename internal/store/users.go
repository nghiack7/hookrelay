package store

import (
	"fmt"
	"time"
)

type User struct {
	APIKey      string    `json:"api_key"`
	Plan        string    `json:"plan"`
	StripeID    string    `json:"stripe_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s *Store) UpsertUser(apiKey, plan string) error {
	_, err := s.db.Exec(`
		INSERT INTO users (api_key, plan, created_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(api_key) DO UPDATE SET plan = excluded.plan`,
		apiKey, plan,
	)
	return err
}

func (s *Store) GetUser(apiKey string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		"SELECT api_key, plan, stripe_id, created_at FROM users WHERE api_key = ?", apiKey,
	).Scan(&u.APIKey, &u.Plan, &u.StripeID, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

func (s *Store) SetStripeID(apiKey, stripeID string) error {
	_, err := s.db.Exec("UPDATE users SET stripe_id = ? WHERE api_key = ?", stripeID, apiKey)
	return err
}

func (s *Store) UpdatePlan(apiKey, plan string) error {
	_, err := s.db.Exec("UPDATE users SET plan = ? WHERE api_key = ?", plan, apiKey)
	return err
}

func (s *Store) GetUserByStripeID(stripeID string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		"SELECT api_key, plan, stripe_id, created_at FROM users WHERE stripe_id = ?", stripeID,
	).Scan(&u.APIKey, &u.Plan, &u.StripeID, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by stripe: %w", err)
	}
	return u, nil
}
