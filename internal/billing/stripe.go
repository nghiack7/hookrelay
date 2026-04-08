package billing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type StripeConfig struct {
	SecretKey     string
	WebhookSecret string
	ProPriceID    string
	TeamPriceID   string
}

func LoadConfig() StripeConfig {
	return StripeConfig{
		SecretKey:     os.Getenv("STRIPE_SECRET_KEY"),
		WebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		ProPriceID:    os.Getenv("STRIPE_PRO_PRICE_ID"),
		TeamPriceID:   os.Getenv("STRIPE_TEAM_PRICE_ID"),
	}
}

func (c StripeConfig) Enabled() bool {
	return c.SecretKey != ""
}

// CreateCheckoutSession creates a Stripe Checkout session for upgrading
func (c StripeConfig) CreateCheckoutSession(apiKey string, plan Plan, successURL, cancelURL string) (string, error) {
	priceID := c.ProPriceID
	if plan == PlanTeam {
		priceID = c.TeamPriceID
	}

	body := fmt.Sprintf(
		"mode=subscription&line_items[0][price]=%s&line_items[0][quantity]=1&success_url=%s&cancel_url=%s&client_reference_id=%s",
		priceID, successURL, cancelURL, apiKey,
	)

	req, err := http.NewRequest("POST", "https://api.stripe.com/v1/checkout/sessions", strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.SecretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("stripe request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		URL   string `json:"url"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("stripe error: %s", result.Error.Message)
	}
	return result.URL, nil
}

// VerifyWebhookSignature verifies a Stripe webhook signature
func (c StripeConfig) VerifyWebhookSignature(payload []byte, sigHeader string) bool {
	if c.WebhookSecret == "" {
		return false
	}

	parts := strings.Split(sigHeader, ",")
	var timestamp, signature string
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			timestamp = kv[1]
		case "v1":
			signature = kv[1]
		}
	}

	if timestamp == "" || signature == "" {
		return false
	}

	signedPayload := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(c.WebhookSecret))
	mac.Write([]byte(signedPayload))
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature))
}

// WebhookEvent represents a Stripe webhook event
type WebhookEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// ParseWebhookEvent parses webhook body into event
func ParseWebhookEvent(body []byte) (*WebhookEvent, error) {
	var event WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

// CustomerPortalURL creates a billing portal session
func (c StripeConfig) CustomerPortalURL(customerID, returnURL string) (string, error) {
	body := fmt.Sprintf("customer=%s&return_url=%s", customerID, returnURL)

	req, err := http.NewRequest("POST", "https://api.stripe.com/v1/billing_portal/sessions", strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.SecretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		URL string `json:"url"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.URL, nil
}

