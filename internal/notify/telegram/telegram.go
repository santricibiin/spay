package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/hirotomasato/paygatemeapp/internal/invoice"
)

// Config holds the bot destination settings.
type Config struct {
	BotToken    string
	PaidChatID  string // channel for paid invoices
	AdminChatID string // chat for alerts (session expiry, etc.)
}

// Notifier sends messages (invoice images + text alerts) via a Telegram bot.
type Notifier struct {
	mu   sync.RWMutex
	conf Config
	cl   *http.Client
}

// New creates a Notifier. An empty token disables sending (Send* are no-ops).
func New(conf Config) *Notifier {
	return &Notifier{conf: conf, cl: &http.Client{Timeout: 15 * time.Second}}
}

// SetConfig updates the bot configuration at runtime (thread-safe).
func (n *Notifier) SetConfig(conf Config) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.conf = conf
}

// Config returns the current configuration.
func (n *Notifier) Config() Config {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.conf
}

// SendInvoice sends a payment-receipt invoice image to the paid chat.
func (n *Notifier) SendInvoice(ctx context.Context, data invoice.Data) error {
	n.mu.RLock()
	conf := n.conf
	n.mu.RUnlock()
	if conf.BotToken == "" || conf.PaidChatID == "" {
		return nil
	}
	png, err := invoice.Generate(data)
	if err != nil {
		return fmt.Errorf("telegram: generate invoice: %w", err)
	}
	caption := fmt.Sprintf("💸 *%s* — %s\n━━━━━━━━━━━━\nStore: %s\nRef: %s\nDate: %s",
		formatRupiah(data.Amount),
		data.Provider,
		data.StoreName,
		data.Reference,
		data.PaidAt.Format("02 Jan 15:04 WIB"),
	)
	return n.sendPhoto(ctx, conf, png, caption)
}

// SendAlert sends a plain text alert to the admin chat.
func (n *Notifier) SendAlert(ctx context.Context, message string) error {
	n.mu.RLock()
	conf := n.conf
	n.mu.RUnlock()
	if conf.BotToken == "" || conf.AdminChatID == "" {
		return nil
	}
	return n.sendText(ctx, conf, message)
}

// SendTestPhoto sends a test invoice image to the paid chat, used by the
// "test connection" button in the settings panel.
func (n *Notifier) SendTestPhoto(ctx context.Context, storeName string) error {
	n.mu.RLock()
	conf := n.conf
	n.mu.RUnlock()
	if conf.BotToken == "" || conf.PaidChatID == "" {
		return fmt.Errorf("telegram: bot token and paid chat ID must be set")
	}
	png, err := invoice.Generate(invoice.Data{
		TransactionID: "pay_test_connection",
		Reference:     "test",
		StoreName:     storeName,
		Amount:        0,
		Provider:      "Test",
		PaidAt:        time.Now(),
	})
	if err != nil {
		return err
	}
	caption := "✅ *Connection test* — Telegram notifier works."
	return n.sendPhoto(ctx, conf, png, caption)
}

func (n *Notifier) sendPhoto(ctx context.Context, conf Config, png []byte, caption string) error {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	part, err := w.CreateFormFile("photo", "invoice.png")
	if err != nil {
		return err
	}
	if _, err := part.Write(png); err != nil {
		return err
	}
	w.WriteField("chat_id", conf.PaidChatID)
	w.WriteField("caption", caption)
	w.WriteField("parse_mode", "Markdown")
	w.Close()

	u := fmt.Sprintf("https://api.telegram.org/bot%s/sendPhoto", conf.BotToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	return n.do(req)
}

func (n *Notifier) sendText(ctx context.Context, conf Config, message string) error {
	form := url.Values{}
	form.Set("chat_id", conf.AdminChatID)
	form.Set("text", message)
	form.Set("parse_mode", "Markdown")

	u := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", conf.BotToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return n.do(req)
}

func (n *Notifier) do(req *http.Request) error {
	resp, err := n.cl.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram: %d — %s", resp.StatusCode, string(respBody))
	}
	var tgResp struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	json.Unmarshal(respBody, &tgResp)
	if !tgResp.OK {
		return fmt.Errorf("telegram: %s", tgResp.Description)
	}
	return nil
}

func formatRupiah(n int64) string {
	s := fmt.Sprintf("%d", n)
	out := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out += "."
		}
		out += string(c)
	}
	return "Rp " + out
}