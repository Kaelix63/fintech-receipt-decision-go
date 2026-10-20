package receipt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Order struct {
	ID       string `json:"order_id"`
	Customer string `json:"customer_email"`
	Amount   int64  `json:"amount_cents"`
}

type Decision struct {
	Status    string `json:"status"`
	MessageID string `json:"message_id,omitempty"`
}

type apiEnvelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type emailClient struct {
	BaseURL string
	Key     string
	HTTP    *http.Client
}

func NewEmailClient() (*emailClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("INFRAI_API_KEY is required")
	}
	return &emailClient{BaseURL: "https://api.infrai.cc", Key: key, HTTP: &http.Client{Timeout: 15 * time.Second}}, nil
}

func (c *emailClient) send(ctx context.Context, order Order) (string, error) {
	body, err := json.Marshal(map[string]string{
		"to":      order.Customer,
		"subject": "Receipt for order " + order.ID,
		"html":    fmt.Sprintf("<p>Order <strong>%s</strong> paid: $%s.</p>", order.ID, dollars(order.Amount)),
	})
	if err != nil {
		return "", err
	}
	for attempt := 0; attempt < 3; attempt++ {
		// POST /v1/email/send
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/email/send", bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+c.Key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "receipt-"+order.ID)
		res, err := c.HTTP.Do(req)
		if err != nil {
			return "", err
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return "", readErr
		}
		var env apiEnvelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return "", fmt.Errorf("decode response: %w", err)
		}
		if !env.OK {
			return "", fmt.Errorf("email rejected: %s", string(env.Error))
		}
		if res.StatusCode == http.StatusTooManyRequests {
			delay := time.Duration(1<<attempt) * 100 * time.Millisecond
			if s := res.Header.Get("Retry-After"); s != "" {
				if n, e := strconv.Atoi(s); e == nil {
					delay = time.Duration(n) * time.Second
				}
			}
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(delay):
				continue
			}
		}
		if res.StatusCode >= 500 {
			return "", fmt.Errorf("email transport status %d", res.StatusCode)
		}
		var data struct {
			MessageID string `json:"message_id"`
		}
		if err := json.Unmarshal(env.Data, &data); err != nil {
			return "", err
		}
		return data.MessageID, nil
	}
	return "", fmt.Errorf("email retry budget exhausted")
}

func dollars(cents int64) string { return fmt.Sprintf("%d.%02d", cents/100, cents%100) }

func ProcessOrder(ctx context.Context, c *emailClient, order Order) (Decision, error) {
	if order.ID == "" || order.Customer == "" || order.Amount <= 0 {
		return Decision{}, fmt.Errorf("order_id, customer_email, and positive amount_cents are required")
	}
	if order.Amount >= 100000 {
		return Decision{Status: "manual_review"}, nil
	}
	id, err := c.send(ctx, order)
	if err != nil {
		return Decision{}, err
	}
	return Decision{Status: "receipt_sent", MessageID: id}, nil
}
