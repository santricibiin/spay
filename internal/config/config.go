package config

import "os"

// Config is the runtime configuration, loaded from environment variables.
type Config struct {
	Port             string
	DatabaseURL      string
	RedisURL         string
	AdminPassword    string // bcrypt hash of the admin password
	AdminJWTSecret   string
	SessionEncryptKey string // 32-byte key (hex) for encrypting provider sessions at rest
	StaticQris        string // static QRIS payload bound to the Shopee store
	TelegramBotToken  string
	TelegramPaidChatID  string // channel for paid invoices
	TelegramAdminChatID string // DM for session-expiry alerts
}

func Load() *Config {
	return &Config{
		Port:              envOr("PORT", "8080"),
		DatabaseURL:       envOr("DATABASE_URL", "postgres://paygateme:paygateme@localhost:5432/paygatemeapp?sslmode=disable"),
		RedisURL:          envOr("REDIS_URL", "redis://localhost:6379/0"),
		AdminPassword:     os.Getenv("ADMIN_PASSWORD"),
		AdminJWTSecret:    envOr("ADMIN_JWT_SECRET", "dev-only-change-me"),
		SessionEncryptKey: os.Getenv("SESSION_ENCRYPT_KEY"),
		StaticQris:        os.Getenv("STATIC_QRIS"),
		TelegramBotToken:  os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramPaidChatID:  os.Getenv("TELEGRAM_PAID_CHAT_ID"),
		TelegramAdminChatID: os.Getenv("TELEGRAM_ADMIN_CHAT_ID"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
