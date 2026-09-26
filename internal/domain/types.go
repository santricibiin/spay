package domain

import (
	"errors"
	"time"
)

// Sentinel errors.
var (
	ErrNotAuthenticated  = errors.New("not authenticated")
	ErrChallengeExpired  = errors.New("otp challenge expired")
	ErrLoginFailed       = errors.New("login failed")
	ErrStoreNotFound     = errors.New("store not found")
	ErrTransactionNotFound = errors.New("transaction not found")
)

// Store is a merchant consumer of the gateway.
type Store struct {
	ID         string
	Name       string
	KeyHash    string // SHA-256 hex of the server key (for incoming auth)
	KeyEnc     []byte // AES-GCM encrypted plaintext (for signing outgoing webhooks)
	WebhookURL string
	Active     bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type TransactionStatus string

const (
	StatusPending    TransactionStatus = "pending"
	StatusSettlement TransactionStatus = "settlement"
	StatusExpired    TransactionStatus = "expired"
	StatusCancelled  TransactionStatus = "cancelled"
)

// Transaction is a payment created by a store.
type Transaction struct {
	ID             string
	StoreID        string
	Amount         int64
	UniqueAmount   int64
	Reference      string
	IdempotencyKey string
	Status         TransactionStatus
	QRString       string
	Provider       string
	ProviderTxID   string
	ExpiresAt      time.Time
	PaidAt         *time.Time
	CreatedAt      time.Time
}

type WebhookStatus string

const (
	WebhookPending WebhookStatus = "pending"
	WebhookSuccess WebhookStatus = "success"
	WebhookFailed  WebhookStatus = "failed"
)

// WebhookDelivery is one attempted notification to a store's webhook URL.
type WebhookDelivery struct {
	ID            string
	TransactionID string
	StoreID       string
	Event         string
	URL           string
	Status        WebhookStatus
	StatusCode    int
	Response      string
	Attempt       int
	NextRetryAt   *time.Time
	CreatedAt     time.Time
}
