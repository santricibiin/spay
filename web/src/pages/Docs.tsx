import { useState } from "react";
import {
  BookOpen,
  KeyRound,
  ArrowLeftRight,
  Webhook,
  ShieldCheck,
  Copy,
  Check,
} from "lucide-react";
import { Card } from "../components/ui/Card";
import { Badge } from "../components/ui/Badge";

const sections = [
  { id: "auth", label: "Authentication", icon: KeyRound },
  { id: "transactions", label: "Transactions", icon: ArrowLeftRight },
  { id: "webhook", label: "Webhook", icon: Webhook },
  { id: "signature", label: "Signature", icon: ShieldCheck },
];

function CodeBlock({ code }: { code: string }) {
  const [copied, setCopied] = useState(false);
  async function copy() {
    await navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }
  return (
    <div className="relative rounded-xl border border-line bg-raised">
      <button
        onClick={copy}
        className="absolute right-2 top-2 rounded-lg p-1.5 text-faint hover:bg-line hover:text-fg"
      >
        {copied ? <Check size={14} className="text-mint" /> : <Copy size={14} />}
      </button>
      <pre className="overflow-x-auto p-4 font-mono text-[13px] leading-relaxed text-fg">
        <code>{code}</code>
      </pre>
    </div>
  );
}

function Method({ m }: { m: "POST" | "GET" | "DELETE" | "PUT" }) {
  const tone = m === "POST" ? "mint" : m === "GET" ? "sky" : m === "DELETE" ? "crimson" : "amber";
  return <Badge tone={tone}>{m}</Badge>;
}

export default function Docs() {
  const [active, setActive] = useState("auth");

  return (
    <div className="space-y-5">
      <div className="flex items-center gap-2">
        <BookOpen size={20} className="text-mint" />
        <h1 className="text-lg font-semibold tracking-tight">Documentation</h1>
      </div>

      {/* Section nav */}
      <div className="flex flex-wrap gap-2">
        {sections.map((s) => (
          <button
            key={s.id}
            onClick={() => setActive(s.id)}
            className={`inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-medium transition-colors ${
              active === s.id
                ? "border-mint/40 bg-mint-dim text-mint"
                : "border-line text-mut hover:text-fg"
            }`}
          >
            <s.icon size={13} />
            {s.label}
          </button>
        ))}
      </div>

      {active === "auth" && (
        <div className="space-y-4">
          <Card className="p-5">
            <h2 className="mb-3 text-sm font-semibold">Server Key</h2>
            <p className="mb-4 text-sm leading-relaxed text-mut">
              Setiap store punya satu server key. Dipakai untuk semua request
              ke Store API. Kirim sebagai Bearer token di header Authorization.
            </p>
            <CodeBlock
              code={`Authorization: Bearer pg_live_xxxxxxxxxxxxxxxxxxxxxxxxxxxx`}
            />
            <div className="mt-4 rounded-xl border border-amber/25 bg-amber-dim/40 p-3 text-xs text-amber">
              Key hanya ditampilkan sekali saat store dibuat atau di-rotate.
              Simpan di tempat aman.
            </div>
          </Card>

          <Card className="p-5">
            <h2 className="mb-3 text-sm font-semibold">Base URL</h2>
            <CodeBlock code={`https://paygateme.com/api/v1`} />
          </Card>
        </div>
      )}

      {active === "transactions" && (
        <div className="space-y-4">
          <Card className="p-5">
            <div className="mb-3 flex items-center gap-2">
              <Method m="POST" />
              <span className="font-mono text-sm">/transactions</span>
            </div>
            <p className="mb-4 text-sm text-mut">
              Buat pembayaran baru. Header <code>Idempotency-Key</code> wajib
              untuk mencegah transaksi ganda saat retry.
            </p>
            <CodeBlock
              code={`curl -X POST https://paygateme.com/api/v1/transactions \\
  -H "Authorization: Bearer pg_live_..." \\
  -H "Idempotency-Key: order-001" \\
  -H "Content-Type: application/json" \\
  -d '{"amount": 50000, "reference": "order-001"}'

// Response 201
{
  "transaction_id": "tx_abc123",
  "amount": 50000,
  "unique_amount": 50001,
  "reference": "order-001",
  "status": "pending",
  "qr_string": "00020101021126610016ID.CO.SHOPEE...",
  "expires_at": "2026-09-10T15:12:00Z"
}`}
            />
          </Card>

          <Card className="p-5">
            <div className="mb-3 flex items-center gap-2">
              <Method m="GET" />
              <span className="font-mono text-sm">/transactions/:id</span>
            </div>
            <p className="mb-4 text-sm text-mut">Cek status transaksi.</p>
            <CodeBlock
              code={`curl https://paygateme.com/api/v1/transactions/tx_abc123 \\
  -H "Authorization: Bearer pg_live_..."

// Response 200
{
  "transaction_id": "tx_abc123",
  "status": "settlement",
  "amount": 50000,
  "unique_amount": 50001,
  "paid_at": "2026-09-10T15:10:00Z"
}`}
            />
          </Card>

          <Card className="p-5">
            <div className="mb-3 flex items-center gap-2">
              <Method m="POST" />
              <span className="font-mono text-sm">/transactions/:id/cancel</span>
            </div>
            <p className="mb-4 text-sm text-mut">
              Batalkan transaksi yang masih pending (belum dibayar).
            </p>
            <CodeBlock
              code={`curl -X POST https://paygateme.com/api/v1/transactions/tx_abc123/cancel \\
  -H "Authorization: Bearer pg_live_..."`}
            />
          </Card>
        </div>
      )}

      {active === "webhook" && (
        <div className="space-y-4">
          <Card className="p-5">
            <h2 className="mb-3 text-sm font-semibold">Webhook Notification</h2>
            <p className="mb-4 text-sm leading-relaxed text-mut">
              Saat pembayaran settle atau expire, gateway kirim POST ke URL
              webhook store kamu. Verifikasi signature sebelum memproses.
            </p>
            <CodeBlock
              code={`POST https://store-anda.com/webhook
Content-Type: application/json
X-Signature: sha256=xxxxxxxxxxxxxxxx
X-Idempotency-Key: tx_abc123_transaction.settlement

{
  "event": "transaction.settlement",
  "transaction_id": "tx_abc123",
  "reference": "order-001",
  "amount": 50000,
  "unique_amount": 50001,
  "status": "settlement",
  "provider": "shopee",
  "paid_at": "2026-09-10T15:10:00Z"
}`}
            />
          </Card>

          <Card className="p-5">
            <h2 className="mb-3 text-sm font-semibold">Event Types</h2>
            <div className="space-y-2 text-sm">
              <div className="flex items-center gap-2">
                <Badge tone="mint">transaction.settlement</Badge>
                <span className="text-mut">Pembayaran diterima &amp; lunas</span>
              </div>
              <div className="flex items-center gap-2">
                <Badge tone="crimson">transaction.expired</Badge>
                <span className="text-mut">Pembayaran kedaluwarsa (5 menit)</span>
              </div>
            </div>
          </Card>

          <Card className="p-5">
            <h2 className="mb-3 text-sm font-semibold">Retry Policy</h2>
            <p className="text-sm leading-relaxed text-mut">
              Store wajib return <code className="text-fg">HTTP 200</code>.
              Jika gagal, gateway retry dengan backoff: 30s → 2m → 10m → 1h → 6h,
              lalu dead-letter.
            </p>
          </Card>
        </div>
      )}

      {active === "signature" && (
        <div className="space-y-4">
          <Card className="p-5">
            <h2 className="mb-3 text-sm font-semibold">Verifikasi Signature</h2>
            <p className="mb-4 text-sm leading-relaxed text-mut">
              Signature dihitung dengan HMAC-SHA256 dari body mentah, key =
              server key store kamu. Bandingkan dengan header X-Signature.
            </p>
            <CodeBlock
              code={`// Node.js
const crypto = require("crypto");

function verify(req) {
  const expected = req.headers["x-signature"];
  const hmac = crypto
    .createHmac("sha256", SERVER_KEY)
    .update(req.rawBody)
    .digest("base64");
  const actual = "sha256=" + hmac;
  return crypto.timingSafeEqual(
    Buffer.from(actual),
    Buffer.from(expected)
  );
}`}
            />
            <div className="mt-4 rounded-xl border border-crimson/25 bg-crimson-dim/40 p-3 text-xs text-crimson">
              Selalu verifikasi signature. Jangan pernah memproses webhook
              tanpa verifikasi — rawan spoofing.
            </div>
          </Card>
        </div>
      )}
    </div>
  );
}