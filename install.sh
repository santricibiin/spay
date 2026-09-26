#!/usr/bin/env bash
set -euo pipefail

# ============================================================
# PayGateMe App — Auto Installer (Ubuntu/Debian, localhost)
# Usage: chmod +x install.sh && sudo ./install.sh
# ============================================================

APP_DIR="$(cd "$(dirname "$0")" && pwd)"
GO_VERSION="1.23.4"
NODE_VERSION="22"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[+]${NC} $1"; }
warn() { echo -e "${YELLOW}[!]${NC} $1"; }
die()  { echo -e "${RED}[x]${NC} $1"; exit 1; }

# --- Root check ---
[[ $EUID -ne 0 ]] && die "Jalankan dengan sudo: sudo ./install.sh"

# --- Detect OS ---
if ! command -v apt-get &>/dev/null; then
    die "Script ini hanya untuk Ubuntu/Debian (apt-get tidak ditemukan)"
fi

log "Update package list..."
apt-get update -qq

# =========================
# 1. Install Go
# =========================
if command -v go &>/dev/null; then
    CURRENT_GO=$(go version | awk '{print $3}' | sed 's/go//')
    log "Go sudah terinstall: v${CURRENT_GO}"
else
    log "Install Go ${GO_VERSION}..."
    ARCH=$(dpkg --print-architecture)
    case "$ARCH" in
        amd64) GO_ARCH="amd64" ;;
        arm64) GO_ARCH="arm64" ;;
        *) die "Arsitektur tidak didukung: $ARCH" ;;
    esac
    GO_TAR="go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    wget -q "https://go.dev/dl/${GO_TAR}" -O "/tmp/${GO_TAR}"
    rm -rf /usr/local/go
    tar -C /usr/local -xzf "/tmp/${GO_TAR}"
    rm "/tmp/${GO_TAR}"

    # Set PATH for current session & future logins
    export PATH="/usr/local/go/bin:$PATH"
    if ! grep -q '/usr/local/go/bin' /etc/profile.d/go.sh 2>/dev/null; then
        echo 'export PATH="/usr/local/go/bin:$PATH"' > /etc/profile.d/go.sh
    fi
    log "Go ${GO_VERSION} terinstall"
fi

# =========================
# 2. Install Node.js
# =========================
if command -v node &>/dev/null; then
    CURRENT_NODE=$(node -v)
    log "Node.js sudah terinstall: ${CURRENT_NODE}"
else
    log "Install Node.js ${NODE_VERSION}..."
    apt-get install -y -qq ca-certificates curl gnupg
    mkdir -p /etc/apt/keyrings
    curl -fsSL "https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key" | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg 2>/dev/null
    echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_${NODE_VERSION}.x nodistro main" > /etc/apt/sources.list.d/nodesource.list
    apt-get update -qq
    apt-get install -y -qq nodejs
    log "Node.js $(node -v) terinstall"
fi

# =========================
# 3. Install Docker
# =========================
if command -v docker &>/dev/null; then
    log "Docker sudah terinstall"
else
    log "Install Docker..."
    apt-get install -y -qq ca-certificates curl
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
    chmod a+r /etc/apt/keyrings/docker.asc
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" > /etc/apt/sources.list.d/docker.list
    apt-get update -qq
    apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-compose-plugin
    systemctl enable --now docker
    log "Docker terinstall"
fi

# =========================
# 4. Install misc tools
# =========================
log "Install dependencies (apache2-utils untuk htpasswd)..."
apt-get install -y -qq apache2-utils openssl wget git

# =========================
# 5. Start PostgreSQL + Redis via Docker
# =========================
log "Start PostgreSQL & Redis..."
cd "$APP_DIR"
docker compose up -d
log "Menunggu PostgreSQL ready..."
until docker compose exec -T postgres pg_isready -U paygateme &>/dev/null; do
    sleep 1
done
log "PostgreSQL ready"

# =========================
# 6. Generate .env
# =========================
if [[ -f "$APP_DIR/.env" ]]; then
    warn ".env sudah ada, skip generate (backup: .env.bak)"
    cp "$APP_DIR/.env" "$APP_DIR/.env.bak"
else
    log "Generate .env..."

    SESSION_KEY=$(openssl rand -hex 32)
    JWT_SECRET=$(openssl rand -hex 16)

    # Default admin password: "admin123" — GANTI SETELAH INSTALL
    ADMIN_HASH=$(htpasswd -bnBC 10 "" "admin123" | tr -d ':\n')

    cat > "$APP_DIR/.env" <<EOF
# Server
PORT=8080
DATABASE_URL=postgres://paygateme:paygateme@localhost:5432/paygatemeapp?sslmode=disable
REDIS_URL=redis://localhost:6379/0

# Admin auth
ADMIN_PASSWORD=${ADMIN_HASH}
ADMIN_JWT_SECRET=${JWT_SECRET}

# Session encryption (auto-generated)
SESSION_ENCRYPT_KEY=${SESSION_KEY}

# Static QRIS (set via panel atau isi manual)
STATIC_QRIS=

# Telegram (opsional, set via panel)
TELEGRAM_BOT_TOKEN=
TELEGRAM_CHAT_ID=
EOF

    log ".env generated (admin password: admin123 — SEGERA GANTI!)"
fi

# =========================
# 7. Build Frontend
# =========================
log "Build frontend..."
cd "$APP_DIR/web"
npm install --silent
npm run build
log "Frontend built"

# =========================
# 8. Build & Run Backend
# =========================
log "Build backend..."
cd "$APP_DIR"
export PATH="/usr/local/go/bin:$PATH"
export GOPATH="${GOPATH:-$HOME/go}"
export PATH="$GOPATH/bin:$PATH"

# Ensure replace directive is removed (use remote module)
sed -i '/^replace /d' go.mod

go mod tidy
go build -o paygatemeapp ./cmd/server
log "Backend built: ${APP_DIR}/paygatemeapp"

# =========================
# 9. Create systemd service
# =========================
log "Setup systemd service..."
cat > /etc/systemd/system/paygatemeapp.service <<EOF
[Unit]
Description=PayGateMe App
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
User=root
WorkingDirectory=${APP_DIR}
EnvironmentFile=${APP_DIR}/.env
ExecStart=${APP_DIR}/paygatemeapp
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable paygatemeapp
systemctl restart paygatemeapp

log "Service started"

# =========================
# Done
# =========================
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN} PayGateMe App berhasil diinstall!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "  URL:       http://localhost:8080"
echo -e "  Admin:     password = ${YELLOW}admin123${NC} (GANTI SEGERA!)"
echo -e "  Service:   systemctl status paygatemeapp"
echo -e "  Logs:      journalctl -u paygatemeapp -f"
echo ""
echo -e "  ${YELLOW}Langkah selanjutnya:${NC}"
echo -e "  1. Buka http://localhost:8080"
echo -e "  2. Login dengan password admin123"
echo -e "  3. Setup provider Shopee (OTP login)"
echo -e "  4. Upload QRIS toko"
echo -e "  5. Buat store & dapatkan API key"
echo ""
