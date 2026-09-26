package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hirotomasato/paygatemeapp/internal/db"
	"github.com/hirotomasato/paygatemeapp/internal/domain"
	"github.com/hirotomasato/paygatemeapp/internal/secrets"
)

// Payload is the JSON body sent to store webhook endpoints.
type Payload struct {
	Event         string     `json:"event"`
	TransactionID string     `json:"transaction_id"`
	Reference     string     `json:"reference"`
	Amount        int64      `json:"amount"`
	UniqueAmount  int64      `json:"unique_amount"`
	Status        string     `json:"status"`
	Provider      string     `json:"provider"`
	PaidAt        *time.Time `json:"paid_at,omitempty"`
}

// Dispatcher enqueues and delivers webhook notifications with retries.
type Dispatcher struct {
	repo       *db.Repo
	secretsKey string
	client     *http.Client
	logger     *log.Logger

	stopCh      chan struct{}
	wg          sync.WaitGroup
	retryDelays []time.Duration // index = attempt count
}

// NewDispatcher creates a webhook dispatcher. secretsKey is the AES-256
// key used to decrypt store server keys for signing.
func NewDispatcher(repo *db.Repo, secretsKey string, logger *log.Logger) *Dispatcher {
	if logger == nil {
		logger = log.Default()
	}
	return &Dispatcher{
		repo:       repo,
		secretsKey: secretsKey,
		client:     &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
		stopCh:     make(chan struct{}),
		retryDelays: []time.Duration{
			30 * time.Second,
			2 * time.Minute,
			10 * time.Minute,
			time.Hour,
			6 * time.Hour,
		},
	}
}

// Enqueue creates a pending delivery record for the store's webhook URL.
func (d *Dispatcher) Enqueue(ctx context.Context, tx domain.Transaction, event, url string) error {
	return d.repo.CreateDelivery(ctx, domain.WebhookDelivery{
		ID:            uuid.NewString(),
		TransactionID: tx.ID,
		StoreID:       tx.StoreID,
		Event:         event,
		URL:           url,
	})
}

// Start launches the background delivery worker.
func (d *Dispatcher) Start() {
	d.wg.Add(1)
	go d.worker()
}

// Stop signals the worker to stop and waits.
func (d *Dispatcher) Stop() {
	select {
	case <-d.stopCh:
	default:
		close(d.stopCh)
	}
	d.wg.Wait()
}

func (d *Dispatcher) worker() {
	defer d.wg.Done()
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-d.stopCh:
			return
		case <-ticker.C:
			d.processPending(context.Background())
		}
	}
}

func (d *Dispatcher) processPending(ctx context.Context) {
	deliveries, err := d.repo.ListPendingDeliveries(ctx)
	if err != nil {
		d.logger.Printf("webhook: list pending: %v", err)
		return
	}
	for _, delivery := range deliveries {
		if err := d.deliver(ctx, delivery); err != nil {
			d.logger.Printf("webhook: deliver %s: %v", delivery.ID, err)
		}
	}
}

func (d *Dispatcher) deliver(ctx context.Context, delivery domain.WebhookDelivery) error {
	store, err := d.repo.GetStoreByID(ctx, delivery.StoreID)
	if err != nil || store == nil {
		return fmt.Errorf("store %s: %w", delivery.StoreID, err)
	}
	tx, err := d.repo.GetTransactionByID(ctx, delivery.TransactionID)
	if err != nil || tx == nil {
		return fmt.Errorf("transaction %s: %w", delivery.TransactionID, err)
	}

	serverKey, err := secrets.Decrypt(d.secretsKey, store.KeyEnc)
	if err != nil {
		return fmt.Errorf("decrypt store key: %w", err)
	}

	payload := Payload{
		Event:         delivery.Event,
		TransactionID: tx.ID,
		Reference:     tx.Reference,
		Amount:        tx.Amount,
		UniqueAmount:  tx.UniqueAmount,
		Status:        string(tx.Status),
		Provider:      tx.Provider,
		PaidAt:        tx.PaidAt,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, delivery.URL, bytes.NewReader(body))
	if err != nil {
		return d.fail(ctx, delivery, 0, err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature", Sign(serverKey, body))
	req.Header.Set("X-Idempotency-Key", fmt.Sprintf("%s_%s", tx.ID, delivery.Event))

	resp, err := d.client.Do(req)
	if err != nil {
		return d.fail(ctx, delivery, 0, err.Error())
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return d.repo.MarkDeliverySuccess(ctx, delivery.ID, resp.StatusCode, string(respBody))
	}
	return d.fail(ctx, delivery, resp.StatusCode, string(respBody))
}

func (d *Dispatcher) fail(ctx context.Context, delivery domain.WebhookDelivery, code int, response string) error {
	if delivery.Attempt >= len(d.retryDelays) {
		return d.repo.MarkDeliveryDead(ctx, delivery.ID, code, response)
	}
	delay := d.retryDelays[delivery.Attempt]
	return d.repo.MarkDeliveryFailed(ctx, delivery.ID, code, response, time.Now().Add(delay))
}