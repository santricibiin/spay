import { useEffect, useState } from "react";
import { ArrowLeftRight, Search, QrCode, History } from "lucide-react";
import { api, type Transaction, type WebhookDelivery } from "../api/client";
import { Card } from "../components/ui/Card";
import { Badge } from "../components/ui/Badge";
import { Input } from "../components/ui/Input";
import { Modal } from "../components/ui/Modal";
import { rupiah, timeAgo } from "../lib/format";

const statusTone: Record<string, "mint" | "amber" | "crimson" | "muted"> = {
  pending: "amber",
  settlement: "mint",
  expired: "crimson",
  cancelled: "muted",
};

export default function Transactions() {
  const [txs, setTxs] = useState<Transaction[]>([]);
  const [filter, setFilter] = useState("");
  const [selected, setSelected] = useState<Transaction | null>(null);
  const [logs, setLogs] = useState<WebhookDelivery[]>([]);

  useEffect(() => {
    api.get<Transaction[]>("/admin/transactions").then(setTxs);
  }, []);

  async function openDetail(tx: Transaction) {
    setSelected(tx);
    // The webhook-logs endpoint resolves by transaction_id (store path ignored).
    const res = await api.get<WebhookDelivery[]>(
      `/admin/stores/_/webhook-logs?transaction_id=${tx.transaction_id}`,
    );
    setLogs(res);
  }

  const filtered = filter
    ? txs.filter(
        (t) =>
          t.reference.toLowerCase().includes(filter.toLowerCase()) ||
          t.transaction_id.includes(filter),
      )
    : txs;

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-semibold tracking-tight">Transactions</h1>
        <span className="text-sm text-mut">{txs.length} total</span>
      </div>

      <div className="relative">
        <Search
          size={16}
          className="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 text-faint"
        />
        <Input
          placeholder="Filter by reference or ID..."
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          className="pl-10"
        />
      </div>

      <div className="space-y-2">
        {filtered.map((tx) => (
          <Card
            key={tx.transaction_id}
            className="flex cursor-pointer items-center gap-3 p-3.5 transition-colors hover:border-mint/30"
            onClick={() => openDetail(tx)}
          >
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-raised">
              <ArrowLeftRight size={16} className="text-mut" />
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium truncate">
                {tx.reference || tx.transaction_id.slice(0, 10)}
              </p>
              <p className="text-xs text-mut">
                {tx.transaction_id.slice(0, 12)} · {timeAgo(tx.created_at)}
              </p>
            </div>
            <div className="text-right">
              <p className="text-sm font-semibold tnum">
                {rupiah(tx.unique_amount)}
              </p>
              <Badge tone={statusTone[tx.status]}>{tx.status}</Badge>
            </div>
          </Card>
        ))}
        {filtered.length === 0 && (
          <p className="py-8 text-center text-sm text-mut">
            {filter ? "No matching transactions." : "No transactions yet."}
          </p>
        )}
      </div>

      {/* Transaction detail modal */}
      <Modal
        open={!!selected}
        onClose={() => setSelected(null)}
        title="Transaction detail"
      >
        {selected && (
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-3 text-sm">
              <div>
                <p className="text-xs text-mut">Amount</p>
                <p className="font-semibold tnum">{rupiah(selected.amount)}</p>
              </div>
              <div>
                <p className="text-xs text-mut">Unique amount</p>
                <p className="font-semibold tnum">
                  {rupiah(selected.unique_amount)}
                </p>
              </div>
              <div>
                <p className="text-xs text-mut">Status</p>
                <Badge tone={statusTone[selected.status]}>
                  {selected.status}
                </Badge>
              </div>
              <div>
                <p className="text-xs text-mut">Reference</p>
                <p className="font-medium truncate">
                  {selected.reference || "—"}
                </p>
              </div>
              <div className="col-span-2">
                <p className="text-xs text-mut">Transaction ID</p>
                <p className="font-mono text-xs break-all">
                  {selected.transaction_id}
                </p>
              </div>
            </div>

            {selected.qr_string && (
              <div>
                <p className="mb-1 flex items-center gap-1.5 text-xs text-mut">
                  <QrCode size={13} />
                  QR string
                </p>
                <code className="block break-all rounded-lg border border-line bg-raised p-2.5 text-xs text-mint tnum">
                  {selected.qr_string}
                </code>
              </div>
            )}

            <div>
              <p className="mb-2 flex items-center gap-1.5 text-xs font-medium text-mut">
                <History size={13} />
                Webhook deliveries
              </p>
              {logs.length === 0 ? (
                <p className="text-xs text-mut">No deliveries yet.</p>
              ) : (
                <div className="space-y-1.5">
                  {logs.map((d) => (
                    <div
                      key={d.id}
                      className="flex items-center justify-between rounded-lg border border-line bg-raised px-3 py-2"
                    >
                      <div>
                        <p className="text-xs font-medium">{d.event}</p>
                        <p className="text-xs text-mut">
                          attempt {d.attempt} · {timeAgo(d.created_at)}
                        </p>
                      </div>
                      <Badge
                        tone={
                          d.status === "success"
                            ? "mint"
                            : d.status === "failed"
                              ? "crimson"
                              : "amber"
                        }
                      >
                        {d.status}
                      </Badge>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}