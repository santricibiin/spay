package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/hirotomasato/paygatemeapp/internal/config"
	"github.com/hirotomasato/paygatemeapp/internal/db"
	"github.com/hirotomasato/paygatemeapp/internal/domain"
	"github.com/hirotomasato/paygatemeapp/internal/gateway"
	"github.com/hirotomasato/paygatemeapp/internal/notify/telegram"
	"github.com/hirotomasato/paygatemeapp/internal/webhook"
)

type ctxKey int

const storeCtxKey ctxKey = iota

// Server wires together all handlers.
type Server struct {
	cfg     *config.Config
	repo    *db.Repo
	gw      *gateway.Orchestrator
	webhook *webhook.Dispatcher
	tg      *telegram.Notifier
}

func NewServer(cfg *config.Config, repo *db.Repo, gw *gateway.Orchestrator, wh *webhook.Dispatcher, tg *telegram.Notifier) *Server {
	return &Server{cfg: cfg, repo: repo, gw: gw, webhook: wh, tg: tg}
}

func storeFromCtx(ctx context.Context) *domain.Store {
	s, _ := ctx.Value(storeCtxKey).(*domain.Store)
	return s
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	return dec.Decode(v)
}