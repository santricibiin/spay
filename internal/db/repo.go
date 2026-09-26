package db

import (
	"context"
	"errors"
	"time"

	"github.com/hirotomasato/paygatemeapp/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repo is the gateway's PostgreSQL repository.
type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// ---- Stores ----

func (r *Repo) CreateStore(ctx context.Context, s domain.Store) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO stores (id, name, key_hash, key_enc, webhook_url, active)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		s.ID, s.Name, s.KeyHash, s.KeyEnc, s.WebhookURL, s.Active,
	)
	return err
}

func (r *Repo) GetStoreByKeyHash(ctx context.Context, hash string) (*domain.Store, error) {
	return r.scanStore(ctx, `SELECT id, name, key_hash, key_enc, webhook_url, active, created_at, updated_at
		FROM stores WHERE key_hash = $1`, hash)
}

func (r *Repo) GetStoreByID(ctx context.Context, id string) (*domain.Store, error) {
	return r.scanStore(ctx, `SELECT id, name, key_hash, key_enc, webhook_url, active, created_at, updated_at
		FROM stores WHERE id = $1`, id)
}

func (r *Repo) ListStores(ctx context.Context) ([]domain.Store, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, key_hash, key_enc, webhook_url, active, created_at, updated_at
		 FROM stores ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Store{}
	for rows.Next() {
		s, err := scanStoreRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (r *Repo) UpdateStore(ctx context.Context, s domain.Store) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE stores SET name = $2, key_hash = $3, key_enc = $4, webhook_url = $5, active = $6, updated_at = now()
		 WHERE id = $1`,
		s.ID, s.Name, s.KeyHash, s.KeyEnc, s.WebhookURL, s.Active,
	)
	return err
}

func (r *Repo) scanStore(ctx context.Context, query string, arg any) (*domain.Store, error) {
	row := r.pool.QueryRow(ctx, query, arg)
	s, err := scanStoreRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanStoreRow(row rowScanner) (*domain.Store, error) {
	var s domain.Store
	err := row.Scan(&s.ID, &s.Name, &s.KeyHash, &s.KeyEnc, &s.WebhookURL, &s.Active, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ---- Transactions ----

func (r *Repo) CreateTransaction(ctx context.Context, t domain.Transaction) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO transactions (id, store_id, amount, unique_amount, reference, idempotency_key,
		 status, qr_string, provider, provider_tx_id, expires_at, paid_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		t.ID, t.StoreID, t.Amount, t.UniqueAmount, t.Reference, t.IdempotencyKey,
		string(t.Status), t.QRString, t.Provider, t.ProviderTxID, t.ExpiresAt, t.PaidAt, t.CreatedAt,
	)
	return err
}

func (r *Repo) GetTransactionByID(ctx context.Context, id string) (*domain.Transaction, error) {
	t, err := r.scanTransaction(ctx,
		`SELECT id, store_id, amount, unique_amount, reference, idempotency_key, status,
		 qr_string, provider, provider_tx_id, expires_at, paid_at, created_at
		 FROM transactions WHERE id = $1`, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return t, err
}

func (r *Repo) GetTransactionByIdempotency(ctx context.Context, storeID, key string) (*domain.Transaction, error) {
	t, err := r.scanTransaction(ctx,
		`SELECT id, store_id, amount, unique_amount, reference, idempotency_key, status,
		 qr_string, provider, provider_tx_id, expires_at, paid_at, created_at
		 FROM transactions WHERE store_id = $1 AND idempotency_key = $2`, storeID, key)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return t, err
}

func (r *Repo) UpdateTransactionStatus(ctx context.Context, id string, status domain.TransactionStatus, providerTxID string, paidAt *time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE transactions SET status = $2, provider_tx_id = $3, paid_at = $4 WHERE id = $1`,
		id, string(status), providerTxID, paidAt,
	)
	return err
}

func (r *Repo) ListTransactions(ctx context.Context, storeID string, limit int) ([]domain.Transaction, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, store_id, amount, unique_amount, reference, idempotency_key, status,
		 qr_string, provider, provider_tx_id, expires_at, paid_at, created_at
		 FROM transactions WHERE ($1 = '' OR store_id = $1) ORDER BY created_at DESC LIMIT $2`,
		storeID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Transaction{}
	for rows.Next() {
		t, err := scanTransactionRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (r *Repo) scanTransaction(ctx context.Context, query string, args ...any) (*domain.Transaction, error) {
	t, err := scanTransactionRow(r.pool.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

func scanTransactionRow(row rowScanner) (*domain.Transaction, error) {
	var t domain.Transaction
	var status string
	err := row.Scan(&t.ID, &t.StoreID, &t.Amount, &t.UniqueAmount, &t.Reference, &t.IdempotencyKey,
		&status, &t.QRString, &t.Provider, &t.ProviderTxID, &t.ExpiresAt, &t.PaidAt, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	t.Status = domain.TransactionStatus(status)
	return &t, nil
}

// ---- Webhook deliveries ----

func (r *Repo) CreateDelivery(ctx context.Context, d domain.WebhookDelivery) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO webhook_deliveries (id, transaction_id, store_id, event, url, status, attempt, created_at)
		 VALUES ($1, $2, $3, $4, $5, 'pending', 0, now())`,
		d.ID, d.TransactionID, d.StoreID, d.Event, d.URL,
	)
	return err
}

func (r *Repo) ListDeliveriesByStore(ctx context.Context, storeID string, limit int) ([]domain.WebhookDelivery, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, transaction_id, store_id, event, url, status, status_code, response, attempt, next_retry_at, created_at
		 FROM webhook_deliveries WHERE store_id = $1 ORDER BY created_at DESC LIMIT $2`,
		storeID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.WebhookDelivery{}
	for rows.Next() {
		d, err := scanDeliveryRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (r *Repo) ListPendingDeliveries(ctx context.Context) ([]domain.WebhookDelivery, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, transaction_id, store_id, event, url, status, status_code, response, attempt, next_retry_at, created_at
		 FROM webhook_deliveries
		 WHERE status = 'pending' AND (next_retry_at IS NULL OR next_retry_at <= now())
		 ORDER BY created_at LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.WebhookDelivery{}
	for rows.Next() {
		d, err := scanDeliveryRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (r *Repo) ListDeliveriesByTransaction(ctx context.Context, txID string) ([]domain.WebhookDelivery, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, transaction_id, store_id, event, url, status, status_code, response, attempt, next_retry_at, created_at
		 FROM webhook_deliveries WHERE transaction_id = $1 ORDER BY created_at`, txID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.WebhookDelivery{}
	for rows.Next() {
		d, err := scanDeliveryRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (r *Repo) MarkDeliverySuccess(ctx context.Context, id string, statusCode int, response string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE webhook_deliveries SET status = 'success', status_code = $2, response = $3 WHERE id = $1`,
		id, statusCode, response,
	)
	return err
}

func (r *Repo) MarkDeliveryFailed(ctx context.Context, id string, statusCode int, response string, nextRetryAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE webhook_deliveries SET status = 'pending', status_code = $2, response = $3, attempt = attempt + 1, next_retry_at = $4 WHERE id = $1`,
		id, statusCode, response, nextRetryAt,
	)
	return err
}

func (r *Repo) MarkDeliveryDead(ctx context.Context, id string, statusCode int, response string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE webhook_deliveries SET status = 'failed', status_code = $2, response = $3 WHERE id = $1`,
		id, statusCode, response,
	)
	return err
}

func scanDeliveryRow(row rowScanner) (*domain.WebhookDelivery, error) {
	var d domain.WebhookDelivery
	var status string
	var statusCode *int
	err := row.Scan(&d.ID, &d.TransactionID, &d.StoreID, &d.Event, &d.URL, &status,
		&statusCode, &d.Response, &d.Attempt, &d.NextRetryAt, &d.CreatedAt)
	if err != nil {
		return nil, err
	}
	d.Status = domain.WebhookStatus(status)
	if statusCode != nil {
		d.StatusCode = *statusCode
	}
	return &d, nil
}

// ---- Provider sessions ----

func (r *Repo) SaveSession(ctx context.Context, provider string, encrypted []byte, expiresAt *time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO provider_sessions (provider, session, expires_at) VALUES ($1, $2, $3)
		 ON CONFLICT (provider) DO UPDATE SET session = $2, expires_at = $3, updated_at = now()`,
		provider, encrypted, expiresAt,
	)
	return err
}

func (r *Repo) LoadSession(ctx context.Context, provider string) ([]byte, error) {
	var data []byte
	err := r.pool.QueryRow(ctx,
		`SELECT session FROM provider_sessions WHERE provider = $1`, provider).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return data, err
}

func (r *Repo) DeleteSession(ctx context.Context, provider string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM provider_sessions WHERE provider = $1`, provider)
	return err
}

// ---- Settings ----

func (r *Repo) GetSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := r.pool.QueryRow(ctx, `SELECT value FROM settings WHERE key = $1`, key).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return value, err
}

func (r *Repo) SetSetting(ctx context.Context, key, value string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO settings (key, value) VALUES ($1, $2)
		 ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = now()`,
		key, value,
	)
	return err
}

// DeleteStore hard-deletes a store and its data (transactions, deliveries, payments).
func (r *Repo) DeleteStore(ctx context.Context, storeID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT id FROM transactions WHERE store_id = $1`, storeID)
	if err != nil {
		return err
	}
	var paymentIDs []string
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			rows.Close()
			return err
		}
		paymentIDs = append(paymentIDs, pid)
	}
	rows.Close()

	if _, err := tx.Exec(ctx, `DELETE FROM webhook_deliveries WHERE store_id = $1`, storeID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM transactions WHERE store_id = $1`, storeID); err != nil {
		return err
	}
	for _, pid := range paymentIDs {
		if _, err := tx.Exec(ctx, `DELETE FROM payments WHERE id = $1`, pid); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM stores WHERE id = $1`, storeID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}