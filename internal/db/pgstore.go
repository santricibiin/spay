package db

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/hirotomasato/paygateme/core"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgPaymentStore implements core.PaymentStore backed by PostgreSQL.
type PgPaymentStore struct {
	pool *pgxpool.Pool
}

// NewPgPaymentStore returns a store that serializes core.Payment as JSONB.
func NewPgPaymentStore(pool *pgxpool.Pool) *PgPaymentStore {
	return &PgPaymentStore{pool: pool}
}

// Create inserts a payment; uses upsert so Update can delegate here.
func (s *PgPaymentStore) Create(ctx context.Context, p core.Payment) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO payments (id, status, data) VALUES ($1, $2, $3)
		 ON CONFLICT (id) DO UPDATE SET status = $2, data = $3, updated_at = now()`,
		p.ID, string(p.Status), data,
	)
	return err
}

// Update delegates to Create (upsert). The payment.Service guarantees
// that the payment exists before calling Update.
func (s *PgPaymentStore) Update(ctx context.Context, p core.Payment) error {
	return s.Create(ctx, p)
}

// Get returns a payment by ID, or nil if not found.
func (s *PgPaymentStore) Get(ctx context.Context, id string) (*core.Payment, error) {
	var data []byte
	err := s.pool.QueryRow(ctx, `SELECT data FROM payments WHERE id = $1`, id).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var p core.Payment
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// ListActive returns all pending payments, optionally filtered by scope.
func (s *PgPaymentStore) ListActive(ctx context.Context, scope *core.PaymentScope) ([]core.Payment, error) {
	rows, err := s.pool.Query(ctx, `SELECT data FROM payments WHERE status = 'pending'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []core.Payment{}
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var p core.Payment
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		if scope != nil && p.Scope != nil && !core.SameScope(*p.Scope, *scope) {
			continue
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

var _ core.PaymentStore = (*PgPaymentStore)(nil)