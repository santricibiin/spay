# PayGateMe App

Payment gateway panel untuk **Shopee merchant payments** — QRIS dinamis, webhook realtime, dan invoice Telegram. Dibangun di atas SDK [`paygateme`](https://github.com/hirotomasato/paygateme).

> ⚠️ **Unofficial** — menggunakan SDK reverse-engineered. Tidak berafiliasi dengan Shopee/GoPay. Gunakan dengan risiko sendiri.

---

## Fitur

| Fitur | Deskripsi |
|-------|-----------|
| **Login Shopee OTP** | Login akun merchant Shopee via phone + password (opsional) + OTP |
| **QRIS dinamis** | Upload QR code toko → auto-decode → QR unik per transaksi (auto-expire 5 menit) |
| **Multi-store** | Buat/ubah/hapus store, tiap store punya server key sendiri |
| **Webhook** | Notifikasi `settlement` / `expired` dengan HMAC-SHA256 signature + retry otomatis |
| **Telegram invoice** | Invoice PNG (dark theme) dikirim ke channel saat pembayaran masuk |
| **Session auto-refresh** | Session Shopee di-renew otomatis, alert Telegram jika expired |
| **Captcha login** | Proteksi login admin (text captcha) |
| **Landing page** | Landing publik + panel admin terpisah |
| **Docs panel** | Dokumentasi API built-in di dashboard |

---

## Tech stack

| Layer | Teknologi |
|-------|-----------|
| Backend | Go 1.26, `net/http` (ServeMux) |
| Database | PostgreSQL 16 |
| Provider | [`paygateme`](https://github.com/hirotomasato/paygateme) SDK (Shopee) |
| Frontend | Vite 7 + React 19 + TypeScript + Tailwind CSS 4 |
| Icons | lucide-react |
| Captcha | `dchest/captcha` (self-contained) |
| QR decode | `gozxing` |
| Reverse proxy | Caddy (HTTPS auto) |

---

## Struktur project

```
paygatemeapp/
├── cmd/server/main.go          entry point, wiring, graceful shutdown
├── internal/
│   ├── api/                    HTTP handlers (admin + store API, middleware)
│   ├── config/                 env config loader
│   ├── db/                     pgx pool, migrations, repositories
│   ├── domain/                 shared types
│   ├── gateway/                Shopee provider orchestration (login, session, polling)
│   ├── invoice/                invoice PNG generator
│   ├── key/                    server key generate/hash
│   ├── notify/telegram/        Telegram sender (invoice + alert)
│   ├── qr/                     QR image → string decoder
│   ├── secrets/                AES-256-GCM helpers
│   └── webhook/                HMAC signature + dispatcher worker
├── static/                     embedded frontend (go:embed)
├── web/                        frontend (Vite + React + Tailwind)
│   └── src/
│       ├── pages/              Landing, Login, Dashboard, Transactions,
│       │                       Stores, StoreDetail, Provider, Docs, Settings
│       ├── components/         Layout, UI primitives
│       ├── auth/               AuthContext (JWT di localStorage)
│       └── api/                typed fetch client
├── docker-compose.yml          PostgreSQL
└── .env.example                template konfigurasi
```

---

## Arsitektur

```
Browser ──▶ https://paygateme.com
              │
              ▼ Caddy (reverse proxy + HTTPS)
              │
              ▼ Go server (:8080)
              │   ├── /            → landing + SPA
              │   ├── /login       → panel login
              │   └── /api/*       → REST
              │
              ▼ PostgreSQL
```

### Alur pembayaran

```
Store (konsumen) ──POST /api/v1/transactions (Bearer key)──▶ Gateway
  → QRIS dinamis (unique amount) terbit
  → customer bayar via ShopeePay
  → Gateway poll feed → match unique amount → status settlement
  → webhook POST ke store + invoice Telegram ke channel
```

### Session lifecycle

```
Login OTP → session (AES-GCM encrypted) → PostgreSQL
  ├── health check tiap 2 menit
  ├── expired? → RefreshSession() (silent renewal)
  └── gagal? → disconnect + alert Telegram
```

---

## API

### Store API (untuk konsumen)

| Method | Endpoint | Deskripsi |
|--------|----------|-----------|
| POST | `/api/v1/transactions` | Buat pembayaran (header `Idempotency-Key` wajib) |
| GET | `/api/v1/transactions/{id}` | Cek status transaksi |
| POST | `/api/v1/transactions/{id}/cancel` | Batalkan transaksi pending |

Auth: `Authorization: Bearer pg_live_...` (server key store).

**Contoh buat transaksi:**

```bash
curl -X POST https://paygateme.com/api/v1/transactions \
  -H "Authorization: Bearer pg_live_..." \
  -H "Idempotency-Key: order-001" \
  -H "Content-Type: application/json" \
  -d '{"amount": 50000, "reference": "order-001"}'
```

```json
// Response 201
{
  "transaction_id": "pay_abc123",
  "amount": 50000,
  "unique_amount": 50001,
  "reference": "order-001",
  "status": "pending",
  "qr_string": "00020101021126610016ID.CO.SHOPEE...",
  "expires_at": "2026-09-10T15:12:00Z"
}
```

### Admin API (dashboard)

| Method | Endpoint | Deskripsi |
|--------|----------|-----------|
| GET | `/api/admin/captcha` | Captcha image (PNG) + `X-Captcha-Id` header |
| POST | `/api/admin/login` | Login admin (password + captcha) → JWT |
| GET/POST | `/api/admin/stores` | List / buat store |
| PUT | `/api/admin/stores/{id}` | Update store |
| DELETE | `/api/admin/stores/{id}` | Hapus store + semua datanya |
| POST | `/api/admin/stores/{id}/rotate-key` | Rotate server key |
| GET | `/api/admin/stores/{id}/webhook-logs` | Delivery log webhook |
| GET | `/api/admin/transactions` | List transaksi |
| GET | `/api/admin/provider/status` | Status session Shopee |
| POST | `/api/admin/provider/otp` | Request OTP (`{phone, password?}`) |
| POST | `/api/admin/provider/verify` | Verify OTP (`{request_id, otp}`) |
| POST | `/api/admin/provider/qris` | Upload QR image → decode → set |
| POST | `/api/admin/provider/logout` | Logout Shopee |
| GET/PUT | `/api/admin/telegram` | Get / save Telegram config |
| POST | `/api/admin/telegram/test` | Test koneksi Telegram |

---

## Webhook

### Payload

```json
{
  "event": "transaction.settlement",
  "transaction_id": "pay_abc123",
  "reference": "order-001",
  "amount": 50000,
  "unique_amount": 50001,
  "status": "settlement",
  "provider": "shopee",
  "paid_at": "2026-09-10T15:10:00Z"
}
```

### Signature

Header `X-Signature: sha256=<base64>` — HMAC-SHA256 dari raw body, key = server key store.

```js
// Node.js verifikasi
const hmac = crypto.createHmac("sha256", SERVER_KEY).update(rawBody).digest("base64");
const valid = crypto.timingSafeEqual(
  Buffer.from("sha256=" + hmac), Buffer.from(req.headers["x-signature"])
);
```

### Retry policy

30s → 2m → 10m → 1h → 6h → dead-letter.

---

## Setup

### Prasyarat

- Go 1.26+
- Node.js 22+
- Docker (untuk PostgreSQL)
- PostgreSQL 16

### 1. Database

```bash
docker compose up -d
```

### 2. Environment

Salin `.env.example` → `.env`, isi:

| Variable | Deskripsi |
|----------|-----------|
| `PORT` | Port HTTP (default 8080) |
| `DATABASE_URL` | Koneksi PostgreSQL |
| `ADMIN_PASSWORD` | bcrypt hash password admin |
| `ADMIN_JWT_SECRET` | Secret JWT (random 32 bytes) |
| `SESSION_ENCRYPT_KEY` | AES-256 key (64 hex chars) untuk enkripsi session |
| `STATIC_QRIS` | QRIS string statis (opsional, bisa di-set via panel) |
| `TELEGRAM_BOT_TOKEN` | Token bot Telegram |
| `TELEGRAM_PAID_CHAT_ID` | Chat ID channel untuk invoice |
| `TELEGRAM_ADMIN_CHAT_ID` | Chat ID untuk alert |

Generate key & password:

```bash
# Session encryption key (64 hex)
openssl rand -hex 32

# Admin password hash (bcrypt)
htpasswd -bnBC 10 "" "passwordkamu" | tr -d ':\n'
```

### 3. Build & run

```bash
# Frontend (output ke static/dist, di-embed oleh Go)
cd web && npm install && npm run build

# Backend (binary tunggal, frontend sudah di-embed)
cd ..
go run ./cmd/server
```

Buka `http://localhost:8080`.

---

## Deployment (VPS)

Lihat struktur sistemd service:

```ini
[Service]
Type=simple
WorkingDirectory=/opt/paygatemeapp
EnvironmentFile=-/opt/paygatemeapp/.env
ExecStart=/opt/paygatemeapp/paygatemeapp
Restart=always
```

Build static binary:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-s -w' -o paygatemeapp ./cmd/server
```

Reverse proxy (Caddy):

```
paygateme.com {
    reverse_proxy localhost:8080
}
```

---

## Keamanan

- **Server key** disimpan sebagai SHA-256 hash (auth masuk) + AES-GCM encrypted (sign webhook keluar). Plaintext ditampilkan sekali saat create/rotate.
- **Session Shopee** dienkripsi AES-256-GCM di database.
- **Webhook** ditandatangani HMAC-SHA256, wajib diverifikasi sisi store.
- **Idempotency-Key** mencegah transaksi ganda.
- **Captcha** di login admin.

---

## Lisensi

Lihat LICENSE.
