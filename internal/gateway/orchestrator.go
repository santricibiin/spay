package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hirotomasato/paygateme/core"
	"github.com/hirotomasato/paygateme/payment"
	"github.com/hirotomasato/paygateme/shopee"
	"github.com/hirotomasato/paygateme/utils"
	"github.com/hirotomasato/paygatemeapp/internal/db"
	"github.com/hirotomasato/paygatemeapp/internal/domain"
	"github.com/hirotomasato/paygatemeapp/internal/secrets"
)

// EventHandler is called when a transaction's status changes.
// The handler is responsible for enqueueing webhooks and any other side effects.
type EventHandler func(ctx context.Context, tx domain.Transaction) error

// Orchestrator ties the Shopee provider, payment.Service, and DB together.
// It owns the provider lifecycle (OTP login, session restore, token refresh).
type Orchestrator struct {
	repo       *db.Repo
	pgStore    *db.PgPaymentStore
	secretsKey string
	staticQris string
	logger     *log.Logger
	pLogger    utils.Logger // for paygateme internals

	mu            sync.Mutex
	provider      *shopee.Provider
	svc           *payment.Service
	otpChallenges map[string]*shopee.OtpChallenge

	onEvent EventHandler
	onExpired func()
	started  bool
}

// NewOrchestrator creates an orchestrator. It does not start the provider
// — call Start() to restore a persisted session, or login via the OTP flow.
func NewOrchestrator(repo *db.Repo, pgStore *db.PgPaymentStore, secretsKey, staticQris string, logger *log.Logger) *Orchestrator {
	if logger == nil {
		logger = log.Default()
	}
	return &Orchestrator{
		repo:          repo,
		pgStore:       pgStore,
		secretsKey:    secretsKey,
		staticQris:    staticQris,
		logger:        logger,
		pLogger:       utils.NewStdLogger(logger, utils.LevelError),
		otpChallenges: make(map[string]*shopee.OtpChallenge),
	}
}

// SetEventHandler sets the callback invoked when a transaction settles or
// expires. The handler receives the domain.Transaction with updated status.
func (o *Orchestrator) SetEventHandler(fn EventHandler) {
	o.onEvent = fn
}

// SetOnExpired sets a callback invoked when the Shopee session expires and
// cannot be renewed (OTP re-login required).
func (o *Orchestrator) SetOnExpired(fn func()) {
	o.onExpired = fn
}

// IsAuthenticated reports whether the provider is logged in.
func (o *Orchestrator) IsAuthenticated() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.provider == nil {
		return false
	}
	return o.provider.Authenticated()
}

// RequestOtp initiates an OTP login for a phone number. Password is optional
// (required only for password-protected accounts). Returns a request ID.
func (o *Orchestrator) RequestOtp(ctx context.Context, phone, password string) (string, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	prov := shopee.NewProvider(shopee.ProviderConfig{
		Store:            o.pgStore,
		StaticQris:       o.staticQris,
		OnSessionUpdated: o.onSessionUpdated,
		DeviceReport:     shopee.DeviceRiskBlob,
		Logger:           o.pLogger,
	})

	challenge, err := prov.RequestOtp(ctx, phone, shopee.OtpRequestOptions{
		Password: password,
	})
	if err != nil {
		// Provide friendlier messages for known Shopee error codes.
		msg := err.Error()
		if strings.Contains(msg, "error 10002") {
			return "", errors.New("password required: this Shopee account has a password. Please enter it.")
		}
		if strings.Contains(msg, "error 48401004") {
			return "", errors.New("wrong password: the Shopee password you entered is incorrect.")
		}
		return "", err
	}

	reqID := uuid.NewString()
	o.otpChallenges[reqID] = challenge
	return reqID, nil
}

// VerifyOtp completes the OTP login. On success, the provider is activated
// and the payment service starts polling.
func (o *Orchestrator) VerifyOtp(ctx context.Context, requestID, otp string) error {
	o.mu.Lock()
	challenge, ok := o.otpChallenges[requestID]
	delete(o.otpChallenges, requestID)
	o.mu.Unlock()

	if !ok {
		return domain.ErrChallengeExpired
	}

	// Build a fresh provider for the final login.
	prov := shopee.NewProvider(shopee.ProviderConfig{
		Store:            o.pgStore,
		StaticQris:       o.staticQris,
		OnSessionUpdated: o.onSessionUpdated,
		DeviceReport:     shopee.DeviceRiskBlob,
		Logger:           o.pLogger,
	})

	outcome, err := prov.LoginWithOtp(ctx, shopee.LoginWithOtpInput{
		Challenge: *challenge,
		OTP:       otp,
	})
	if err != nil {
		return err
	}
	if outcome.Status != shopee.LoginComplete {
		return domain.ErrLoginFailed
	}

	// Persist session.
	if err := o.persistSession(ctx, prov.ExportSession()); err != nil {
		return err
	}

	// Build the payment service.
	svc, err := prov.Payments()
	if err != nil {
		return err
	}

	o.mu.Lock()
	o.provider = prov
	o.svc = svc
	o.mu.Unlock()

	o.registerCallbacks()
	o.startService()
	return nil
}

// Logout clears the session and stops the payment service.
func (o *Orchestrator) Logout(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.svc != nil {
		o.svc.Stop()
		o.svc = nil
	}
	o.provider = nil
	return o.repo.DeleteSession(ctx, "shopee")
}

// CreatePayment creates a payment for a store. The storeID is stored in the
// payment's metadata so the callback can route it correctly.
func (o *Orchestrator) CreatePayment(ctx context.Context, amount int64, reference, storeID string) (*core.Payment, error) {
	o.mu.Lock()
	svc := o.svc
	o.mu.Unlock()

	if svc == nil {
		return nil, domain.ErrNotAuthenticated
	}

	p, err := svc.CreatePayment(ctx, payment.CreatePaymentInput{
		Amount:    amount,
		Reference: reference,
		Metadata:  map[string]any{"store_id": storeID},
	})
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// CancelPayment cancels a pending payment.
func (o *Orchestrator) CancelPayment(ctx context.Context, id string) (*core.Payment, error) {
	o.mu.Lock()
	svc := o.svc
	o.mu.Unlock()

	if svc == nil {
		return nil, domain.ErrNotAuthenticated
	}
	return svc.CancelPayment(ctx, id)
}

// GetPayment returns a payment by ID.
func (o *Orchestrator) GetPayment(ctx context.Context, id string) (*core.Payment, error) {
	return o.pgStore.Get(ctx, id)
}

// SetStaticQris validates and persists the static QRIS payload, then updates
// the provider and its payment service. Requires an active session.
func (o *Orchestrator) SetStaticQris(ctx context.Context, payload string) error {
	o.mu.Lock()
	prov := o.provider
	svc := o.svc
	o.mu.Unlock()

	if prov == nil {
		return domain.ErrNotAuthenticated
	}

	if err := prov.SetStaticQris(payload); err != nil {
		return err
	}

	// Update the payment service so new payments use the new QRIS.
	if svc != nil {
		svc.SetStaticQris(payload)
	}

	// Persist to DB for survival across restarts.
	if err := o.repo.SetSetting(ctx, "static_qris", payload); err != nil {
		return err
	}

	o.mu.Lock()
	o.staticQris = payload
	o.mu.Unlock()

	return nil
}

// Start loads the persisted QRIS, restores a session if valid, and starts the
// session health-check loop.
func (o *Orchestrator) Start(ctx context.Context) error {
	// Load persisted static QRIS (overrides env default).
	if qrisFromDB, err := o.repo.GetSetting(ctx, "static_qris"); err == nil && qrisFromDB != "" {
		o.mu.Lock()
		o.staticQris = qrisFromDB
		o.mu.Unlock()
	}

	// Start the health-check loop regardless of session state.
	o.startHealthCheck()

	enc, err := o.repo.LoadSession(ctx, "shopee")
	if err != nil {
		return err
	}
	if enc == nil {
		return nil // no persisted session
	}

	data, err := secrets.Decrypt(o.secretsKey, enc)
	if err != nil {
		o.repo.DeleteSession(ctx, "shopee")
		return nil // corrupted session; delete and move on
	}

	var sess shopee.Session
	if err := json.Unmarshal(data, &sess); err != nil {
		o.repo.DeleteSession(ctx, "shopee")
		return nil
	}

	o.mu.Lock()
	qr := o.staticQris
	o.mu.Unlock()

	prov := shopee.NewProvider(shopee.ProviderConfig{
		Session:          &sess,
		Store:            o.pgStore,
		StaticQris:       qr,
		OnSessionUpdated: o.onSessionUpdated,
		DeviceReport:     shopee.DeviceRiskBlob,
		Logger:           o.pLogger,
	})

	if !prov.Authenticated() {
		// Try silent renewal before giving up.
		if _, rerr := prov.RefreshSession(ctx); rerr != nil {
			o.repo.DeleteSession(ctx, "shopee")
			return nil
		}
		o.persistSession(ctx, prov.ExportSession())
	}

	svc, err := prov.Payments()
	if err != nil {
		return err
	}

	o.mu.Lock()
	o.provider = prov
	o.svc = svc
	o.mu.Unlock()

	o.registerCallbacks()
	o.startService()
	o.logger.Print("gateway: restored shopee session")
	return nil
}

// ---- internal ----

func (o *Orchestrator) onSessionUpdated(sess shopee.Session) error {
	ctx := context.Background()
	return o.persistSession(ctx, &sess)
}

func (o *Orchestrator) persistSession(ctx context.Context, sess *shopee.Session) error {
	if sess == nil {
		return nil
	}
	data, err := json.Marshal(sess)
	if err != nil {
		return err
	}
	enc, err := secrets.Encrypt(o.secretsKey, data)
	if err != nil {
		return err
	}
	var expiresAt *time.Time
	if sess.ExpiresAt > 0 {
		t := time.Unix(sess.ExpiresAt, 0)
		expiresAt = &t
	}
	return o.repo.SaveSession(ctx, "shopee", enc, expiresAt)
}

func (o *Orchestrator) registerCallbacks() {
	o.mu.Lock()
	svc := o.svc
	o.mu.Unlock()

	if svc == nil {
		return
	}

	svc.OnPaid(func(p core.Payment) {
		o.handleEvent(p, domain.StatusSettlement)
	})
	svc.OnExpired(func(p core.Payment) {
		o.handleEvent(p, domain.StatusExpired)
	})
	svc.OnError(func(err error) {
		o.logger.Printf("gateway: payment error: %v", err)
	})
}

func (o *Orchestrator) handleEvent(p core.Payment, status domain.TransactionStatus) {
	storeID, _ := p.Metadata["store_id"].(string)
	if storeID == "" {
		o.logger.Printf("gateway: no store_id in metadata for payment %s", p.ID)
		return
	}

	ctx := context.Background()
	providerTxID := ""
	if p.Transaction != nil {
		providerTxID = p.Transaction.ID
	}

	var paidAt *time.Time
	if status == domain.StatusSettlement {
		now := time.Now()
		paidAt = &now
		if p.Transaction != nil {
			paidAt = &p.Transaction.Time
		}
	}

	if err := o.repo.UpdateTransactionStatus(ctx, p.ID, status, providerTxID, paidAt); err != nil {
		o.logger.Printf("gateway: update tx %s: %v", p.ID, err)
		return
	}

	tx, err := o.repo.GetTransactionByID(ctx, p.ID)
	if err != nil || tx == nil {
		o.logger.Printf("gateway: load tx %s after update: %v", p.ID, err)
		return
	}

	if o.onEvent != nil {
		if err := o.onEvent(ctx, *tx); err != nil {
			o.logger.Printf("gateway: event handler for %s: %v", p.ID, err)
		}
	}
}

func (o *Orchestrator) startService() {
	o.mu.Lock()
	svc := o.svc
	started := o.started
	o.started = true
	o.mu.Unlock()

	if svc != nil && !started {
		svc.Start()
	}
}

// startHealthCheck launches a background loop that detects session expiry and
// attempts silent renewal. If renewal fails, the session is cleared and the
// gateway reports disconnected (OTP re-login required).
func (o *Orchestrator) startHealthCheck() {
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			o.healthCheck()
		}
	}()
}

func (o *Orchestrator) healthCheck() {
	o.mu.Lock()
	prov := o.provider
	o.mu.Unlock()

	if prov == nil {
		return // not logged in
	}

	if prov.Authenticated() {
		return // still valid
	}

	ctx := context.Background()
	o.logger.Print("gateway: session expired, attempting silent renewal")

	if _, err := prov.RefreshSession(ctx); err != nil {
		o.logger.Printf("gateway: silent renewal failed: %v", err)
		o.mu.Lock()
		if o.svc != nil {
			o.svc.Stop()
			o.svc = nil
		}
		o.provider = nil
		cb := o.onExpired
		o.mu.Unlock()
		o.repo.DeleteSession(ctx, "shopee")
		if cb != nil {
			cb()
		}
		return
	}

	// Persist the renewed session.
	if err := o.persistSession(ctx, prov.ExportSession()); err != nil {
		o.logger.Printf("gateway: persist renewed session: %v", err)
	}
	o.logger.Print("gateway: session renewed")
}