package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hirotomasato/paygatemeapp/internal/api"
	"github.com/hirotomasato/paygatemeapp/internal/config"
	"github.com/hirotomasato/paygatemeapp/internal/db"
	"github.com/hirotomasato/paygatemeapp/internal/domain"
	"github.com/hirotomasato/paygatemeapp/internal/gateway"
	"github.com/hirotomasato/paygatemeapp/internal/invoice"
	"github.com/hirotomasato/paygatemeapp/internal/notify/telegram"
	"github.com/hirotomasato/paygatemeapp/internal/webhook"
)

func main() {
	cfg := config.Load()
	logger := log.New(os.Stdout, "paygatemeapp ", log.LstdFlags)

	if cfg.SessionEncryptKey == "" {
		logger.Fatal("SESSION_ENCRYPT_KEY must be set (64 hex chars)")
	}
	if cfg.AdminPassword == "" {
		logger.Fatal("ADMIN_PASSWORD must be set (bcrypt hash)")
	}

	ctx := context.Background()

	// Database.
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatalf("db: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		logger.Fatalf("migrate: %v", err)
	}
	logger.Print("migrations ok")

	repo := db.NewRepo(pool)
	pgStore := db.NewPgPaymentStore(pool)

	// Gateway orchestrator.
	gw := gateway.NewOrchestrator(repo, pgStore, cfg.SessionEncryptKey, cfg.StaticQris, logger)

	// Webhook dispatcher.
	wh := webhook.NewDispatcher(repo, cfg.SessionEncryptKey, logger)

	// Telegram notifier.
	tgConf := telegram.Config{
		BotToken:    cfg.TelegramBotToken,
		PaidChatID:  cfg.TelegramPaidChatID,
		AdminChatID: cfg.TelegramAdminChatID,
	}
	// Override from DB settings if present.
	if bot, err := repo.GetSetting(ctx, "telegram_bot_token"); err == nil && bot != "" {
		tgConf.BotToken = bot
	}
	if paid, err := repo.GetSetting(ctx, "telegram_paid_chat_id"); err == nil && paid != "" {
		tgConf.PaidChatID = paid
	}
	if admin, err := repo.GetSetting(ctx, "telegram_admin_chat_id"); err == nil && admin != "" {
		tgConf.AdminChatID = admin
	}
	tg := telegram.New(tgConf)

	// Wire event handler: webhook + Telegram notification on settlement.
	gw.SetEventHandler(func(ctx context.Context, tx domain.Transaction) error {
		store, err := repo.GetStoreByID(ctx, tx.StoreID)
		if err != nil || store == nil || !store.Active {
			return err
		}

		// Webhook (if configured).
		if store.WebhookURL != "" {
			event := eventForStatus(tx.Status)
			_ = wh.Enqueue(ctx, tx, event, store.WebhookURL)
		}

		// Telegram invoice (settlement only).
		if tx.Status == domain.StatusSettlement && tx.PaidAt != nil {
			go func() {
				if err := tg.SendInvoice(context.Background(), invoice.Data{
					TransactionID: tx.ID,
					Reference:     tx.Reference,
					StoreName:     store.Name,
					Amount:        tx.UniqueAmount,
					Provider:      tx.Provider,
					PaidAt:        *tx.PaidAt,
				}); err != nil {
					logger.Printf("telegram: %v", err)
				}
			}()
		}

		return nil
	})

	// Restore persisted provider session if any.
	if err := gw.Start(ctx); err != nil {
		logger.Printf("restore session: %v", err)
	}

	// Session-expiry alert → channel.
	gw.SetOnExpired(func() {
		if err := tg.SendAlert(context.Background(),
			"⚠️ *Shopee session expired* — silent renewal failed.\nLogin OTP lagi di dashboard."); err != nil {
			logger.Printf("telegram alert: %v", err)
		}
	})

	wh.Start()
	defer wh.Stop()

	// HTTP server.
	srv := api.NewServer(cfg, repo, gw, wh, tg)
	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: srv.Router(),
	}

	// Graceful shutdown.
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		logger.Print("shutting down")
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		httpServer.Shutdown(shutCtx)
	}()

	logger.Printf("listening on :%s", cfg.Port)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("serve: %v", err)
	}
}

func eventForStatus(s domain.TransactionStatus) string {
	switch s {
	case domain.StatusSettlement:
		return "transaction.settlement"
	case domain.StatusExpired:
		return "transaction.expired"
	case domain.StatusCancelled:
		return "transaction.cancelled"
	default:
		return "transaction." + string(s)
	}
}