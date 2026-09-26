package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/hirotomasato/paygatemeapp/internal/domain"
)

type transactionResponse struct {
	TransactionID string     `json:"transaction_id"`
	Amount        int64      `json:"amount"`
	UniqueAmount  int64      `json:"unique_amount"`
	Reference     string     `json:"reference"`
	Status        string     `json:"status"`
	QRString      string     `json:"qr_string"`
	Provider      string     `json:"provider"`
	ExpiresAt     time.Time  `json:"expires_at"`
	PaidAt        *time.Time `json:"paid_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

func toTransactionResponse(tx domain.Transaction) transactionResponse {
	return transactionResponse{
		TransactionID: tx.ID,
		Amount:        tx.Amount,
		UniqueAmount:  tx.UniqueAmount,
		Reference:     tx.Reference,
		Status:        string(tx.Status),
		QRString:      tx.QRString,
		Provider:      tx.Provider,
		ExpiresAt:     tx.ExpiresAt,
		PaidAt:        tx.PaidAt,
		CreatedAt:     tx.CreatedAt,
	}
}

func (s *Server) createTransaction(w http.ResponseWriter, r *http.Request) {
	store := storeFromCtx(r.Context())

	var req struct {
		Amount    int64  `json:"amount"`
		Reference string `json:"reference"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be positive")
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		writeError(w, http.StatusBadRequest, "Idempotency-Key header required")
		return
	}

	// Check idempotency.
	existing, _ := s.repo.GetTransactionByIdempotency(r.Context(), store.ID, idempotencyKey)
	if existing != nil {
		writeJSON(w, http.StatusOK, toTransactionResponse(*existing))
		return
	}

	// Create payment via the provider.
	p, err := s.gw.CreatePayment(r.Context(), req.Amount, req.Reference, store.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotAuthenticated) {
			writeError(w, http.StatusServiceUnavailable, "gateway not authenticated")
			return
		}
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	// Persist the transaction record.
	tx := domain.Transaction{
		ID:             p.ID,
		StoreID:        store.ID,
		Amount:         p.BaseAmount,
		UniqueAmount:   p.UniqueAmount,
		Reference:      p.Reference,
		IdempotencyKey: idempotencyKey,
		Status:         domain.StatusPending,
		QRString:       p.QRString,
		Provider:       "shopee",
		ExpiresAt:      p.ExpiresAt,
		CreatedAt:      p.CreatedAt,
	}
	if err := s.repo.CreateTransaction(r.Context(), tx); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, toTransactionResponse(tx))
}

func (s *Server) getTransaction(w http.ResponseWriter, r *http.Request) {
	store := storeFromCtx(r.Context())
	tx, err := s.repo.GetTransactionByID(r.Context(), r.PathValue("id"))
	if err != nil || tx == nil {
		writeError(w, http.StatusNotFound, "transaction not found")
		return
	}
	if tx.StoreID != store.ID {
		writeError(w, http.StatusNotFound, "transaction not found")
		return
	}
	writeJSON(w, http.StatusOK, toTransactionResponse(*tx))
}

func (s *Server) cancelTransaction(w http.ResponseWriter, r *http.Request) {
	store := storeFromCtx(r.Context())
	tx, err := s.repo.GetTransactionByID(r.Context(), r.PathValue("id"))
	if err != nil || tx == nil {
		writeError(w, http.StatusNotFound, "transaction not found")
		return
	}
	if tx.StoreID != store.ID {
		writeError(w, http.StatusNotFound, "transaction not found")
		return
	}
	if tx.Status != domain.StatusPending {
		writeError(w, http.StatusBadRequest, "only pending transactions can be cancelled")
		return
	}

	p, err := s.gw.CancelPayment(r.Context(), tx.ID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if p == nil {
		writeError(w, http.StatusNotFound, "payment not found")
		return
	}

	if err := s.repo.UpdateTransactionStatus(r.Context(), tx.ID, domain.StatusCancelled, "", nil); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tx.Status = domain.StatusCancelled
	writeJSON(w, http.StatusOK, toTransactionResponse(*tx))
}

func validWebhookURL(url string) bool {
	return len(url) >= 8 && (url[:8] == "https://" || url[:7] == "http://")
}