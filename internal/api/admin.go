package api

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/dchest/captcha"
	"github.com/google/uuid"
	"github.com/hirotomasato/paygatemeapp/internal/domain"
	"github.com/hirotomasato/paygatemeapp/internal/key"
	"github.com/hirotomasato/paygatemeapp/internal/notify/telegram"
	"github.com/hirotomasato/paygatemeapp/internal/qr"
	"golang.org/x/crypto/bcrypt"
)

// ---- auth ----

func (s *Server) adminLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password     string `json:"password"`
		CaptchaID    string `json:"captcha_id"`
		CaptchaValue string `json:"captcha_value"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if !captcha.VerifyString(req.CaptchaID, req.CaptchaValue) {
		writeError(w, http.StatusUnauthorized, "invalid captcha")
		return
	}

	if s.cfg.AdminPassword == "" {
		writeError(w, http.StatusServiceUnavailable, "admin password not configured")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(s.cfg.AdminPassword), []byte(req.Password)) != nil {
		writeError(w, http.StatusUnauthorized, "wrong password")
		return
	}
	token, err := signJWT(s.cfg.AdminJWTSecret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

// adminCaptcha serves a fresh captcha PNG image. The captcha ID is returned
// in the X-Captcha-Id header.
func (s *Server) adminCaptcha(w http.ResponseWriter, r *http.Request) {
	id := captcha.New()
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("X-Captcha-Id", id)
	w.Header().Set("Cache-Control", "no-store, no-cache")
	captcha.WriteImage(w, id, 200, 64)
}

// ---- stores ----

type storeResponse struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	WebhookURL string    `json:"webhook_url"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
}

func toStoreResponse(s domain.Store) storeResponse {
	return storeResponse{
		ID:         s.ID,
		Name:       s.Name,
		WebhookURL: s.WebhookURL,
		Active:     s.Active,
		CreatedAt:  s.CreatedAt,
	}
}

func (s *Server) adminListStores(w http.ResponseWriter, r *http.Request) {
	stores, err := s.repo.ListStores(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]storeResponse, 0, len(stores))
	for _, st := range stores {
		out = append(out, toStoreResponse(st))
	}
	writeJSON(w, http.StatusOK, out)
}

// adminCreateStore adds a store and returns its server key exactly once.
func (s *Server) adminCreateStore(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string `json:"name"`
		WebhookURL string `json:"webhook_url"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.WebhookURL != "" && !validWebhookURL(req.WebhookURL) {
		writeError(w, http.StatusBadRequest, "webhook_url must be https")
		return
	}

	plain, hash, enc, err := key.Generate(s.cfg.SessionEncryptKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	store := domain.Store{
		ID:         uuid.NewString(),
		Name:       req.Name,
		KeyHash:    hash,
		KeyEnc:     enc,
		WebhookURL: req.WebhookURL,
		Active:     true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := s.repo.CreateStore(r.Context(), store); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"store":       toStoreResponse(store),
		"server_key":  plain,
		"note":        "Store this key now; it is shown only once.",
	})
}

func (s *Server) adminUpdateStore(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	store, err := s.repo.GetStoreByID(r.Context(), id)
	if err != nil || store == nil {
		writeError(w, http.StatusNotFound, "store not found")
		return
	}

	var req struct {
		Name       string `json:"name"`
		WebhookURL string `json:"webhook_url"`
		Active     *bool  `json:"active"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Name != "" {
		store.Name = req.Name
	}
	if req.WebhookURL != "" {
		if !validWebhookURL(req.WebhookURL) {
			writeError(w, http.StatusBadRequest, "webhook_url must be https")
			return
		}
		store.WebhookURL = req.WebhookURL
	}
	if req.Active != nil {
		store.Active = *req.Active
	}

	if err := s.repo.UpdateStore(r.Context(), *store); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toStoreResponse(*store))
}

// adminRotateStoreKey generates a new server key for a store.
func (s *Server) adminRotateStoreKey(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	store, err := s.repo.GetStoreByID(r.Context(), id)
	if err != nil || store == nil {
		writeError(w, http.StatusNotFound, "store not found")
		return
	}

	plain, hash, enc, err := key.Generate(s.cfg.SessionEncryptKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	store.KeyHash = hash
	store.KeyEnc = enc

	if err := s.repo.UpdateStore(r.Context(), *store); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"server_key": plain,
		"note":       "Old key invalidated. Store this key now; it is shown only once.",
	})
}

// adminDeleteStore permanently deletes a store and all its data.
func (s *Server) adminDeleteStore(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	store, err := s.repo.GetStoreByID(r.Context(), id)
	if err != nil || store == nil {
		writeError(w, http.StatusNotFound, "store not found")
		return
	}
	if err := s.repo.DeleteStore(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

// ---- webhook logs ----

func (s *Server) adminWebhookLogs(w http.ResponseWriter, r *http.Request) {
	storeID := r.PathValue("id")
	txID := r.URL.Query().Get("transaction_id")

	if txID != "" {
		deliveries, err := s.repo.ListDeliveriesByTransaction(r.Context(), txID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, deliveries)
		return
	}

	deliveries, err := s.repo.ListDeliveriesByStore(r.Context(), storeID, 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, deliveries)
}

// ---- transactions ----

func (s *Server) adminListTransactions(w http.ResponseWriter, r *http.Request) {
	storeID := r.URL.Query().Get("store_id")
	txs, err := s.repo.ListTransactions(r.Context(), storeID, 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]transactionResponse, 0, len(txs))
	for _, tx := range txs {
		out = append(out, toTransactionResponse(tx))
	}
	writeJSON(w, http.StatusOK, out)
}

// ---- provider ----

func (s *Server) adminProviderStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": s.gw.IsAuthenticated()})
}

func (s *Server) adminProviderOtp(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Phone    string `json:"phone"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Phone == "" {
		writeError(w, http.StatusBadRequest, "phone is required")
		return
	}
	requestID, err := s.gw.RequestOtp(r.Context(), req.Phone, req.Password)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"request_id": requestID})
}

func (s *Server) adminProviderVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RequestID string `json:"request_id"`
		OTP       string `json:"otp"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := s.gw.VerifyOtp(r.Context(), req.RequestID, req.OTP); err != nil {
		if errors.Is(err, domain.ErrChallengeExpired) {
			writeError(w, http.StatusBadRequest, "challenge expired, request a new OTP")
			return
		}
		if errors.Is(err, domain.ErrLoginFailed) {
			writeError(w, http.StatusUnauthorized, "login failed, try again")
			return
		}
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": true})
}

func (s *Server) adminProviderLogout(w http.ResponseWriter, r *http.Request) {
	if err := s.gw.Logout(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": false})
}

// ---- Telegram settings ----

type telegramSettingsResponse struct {
	BotToken    string `json:"bot_token"`
	PaidChatID  string `json:"paid_chat_id"`
	AdminChatID string `json:"admin_chat_id"`
}

func (s *Server) adminGetTelegram(w http.ResponseWriter, r *http.Request) {
	bot, _ := s.repo.GetSetting(r.Context(), "telegram_bot_token")
	paid, _ := s.repo.GetSetting(r.Context(), "telegram_paid_chat_id")
	admin, _ := s.repo.GetSetting(r.Context(), "telegram_admin_chat_id")
	writeJSON(w, http.StatusOK, telegramSettingsResponse{
		BotToken:    bot,
		PaidChatID:  paid,
		AdminChatID: admin,
	})
}

func (s *Server) adminSaveTelegram(w http.ResponseWriter, r *http.Request) {
	var req telegramSettingsResponse
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	s.repo.SetSetting(r.Context(), "telegram_bot_token", req.BotToken)
	s.repo.SetSetting(r.Context(), "telegram_paid_chat_id", req.PaidChatID)
	s.repo.SetSetting(r.Context(), "telegram_admin_chat_id", req.AdminChatID)

	s.tg.SetConfig(telegram.Config{
		BotToken:    req.BotToken,
		PaidChatID:  req.PaidChatID,
		AdminChatID: req.AdminChatID,
	})

	writeJSON(w, http.StatusOK, map[string]bool{"saved": true})
}

func (s *Server) adminTestTelegram(w http.ResponseWriter, r *http.Request) {
	if err := s.tg.SendTestPhoto(r.Context(), "paygatemeapp"); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"sent": true})
}
func (s *Server) adminProviderQris(w http.ResponseWriter, r *http.Request) {
	// Limit to 5MB.
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid upload")
		return
	}
	file, _, err := r.FormFile("qr")
	if err != nil {
		writeError(w, http.StatusBadRequest, "qr image is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read image")
		return
	}

	payload, err := qr.Decode(data)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.gw.SetStaticQris(r.Context(), payload); err != nil {
		if errors.Is(err, domain.ErrNotAuthenticated) {
			writeError(w, http.StatusBadRequest, "login to Shopee first before uploading QRIS")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"qr_string": payload})
}