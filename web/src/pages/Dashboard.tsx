import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import {
  ArrowLeftRight,
  Clock,
  CheckCircle,
  XCircle,
  Zap,
  ZapOff,
  ShoppingCart,
  Users,
} from "lucide-react";
import { api, type Transaction, type ProviderStatus } from "../api/client";
import { Card } from "../components/ui/Card";
import { Badge } from "../components/ui/Badge";
import { rupiah, timeAgo } from "../lib/format";

export default function Dashboard() {
  const [provider, setProvider] = useState<ProviderStatus | null>(null);
  const [txs, setTxs] = useState<Transaction[]>([]);
  const [stores, setStores] = useState<number>(0);

  useEffect(() => {
    api.get<ProviderStatus>("/admin/provider/status").then(setProvider);
    api.get<Transaction[]>("/admin/transactions").then(setTxs);
    api.get<any[]>("/admin/stores").then((s) => setStores(s.length));
  }, []);

  const today = txs.filter(
    (t) => new Date(t.created_at).toDateString() === new Date().toDateString(),
  );
  const pending = txs.filter((t) => t.status === "pending");
  const settled = txs.filter((t) => t.status === "settlement");
  const expired = txs.filter((t) => t.status === "expired");
  const settledVolume = settled.reduce((sum, t) => sum + t.amount, 0);

  const statusTone = {
    pending: "amber" as const,
    settlement: "mint" as const,
    expired: "crimson" as const,
    cancelled: "muted" as const,
  };

  return (
    <div className="space-y-5">
      <h1 className="text-lg font-semibold tracking-tight">Dashboard</h1>

      {/* Provider status */}
      <Card className="p-4">
        <div className="flex items-center gap-3">
          <div
            className={`flex h-10 w-10 items-center justify-center rounded-xl ${
              provider?.authenticated
                ? "bg-mint-dim text-mint"
                : "bg-crimson-dim text-crimson"
            }`}
          >
            {provider?.authenticated ? (
              <Zap size={20} />
            ) : (
              <ZapOff size={20} />
            )}
          </div>
          <div>
            <p className="text-sm font-medium">Shopee Provider</p>
            <p className="text-xs text-mut">
              {provider?.authenticated ? "Connected" : "Disconnected — login"}
            </p>
          </div>
          <Link
            to="/provider"
            className="ml-auto text-sm text-mint hover:underline"
          >
            Manage
          </Link>
        </div>
      </Card>

      {/* Stats row */}
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <StatCard
          icon={ShoppingCart}
          label="Today"
          value={String(today.length)}
          tone="sky"
        />
        <StatCard
          icon={Clock}
          label="Pending"
          value={String(pending.length)}
          tone="amber"
        />
        <StatCard
          icon={CheckCircle}
          label="Settled"
          value={String(settled.length)}
          tone="mint"
        />
        <StatCard
          icon={XCircle}
          label="Expired"
          value={String(expired.length)}
          tone="crimson"
        />
      </div>

      {/* Revenue */}
      <Card className="flex items-center gap-3 p-4">
        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-mint-dim text-mint">
          <ArrowLeftRight size={18} />
        </div>
        <div>
          <p className="text-xs text-mut">Total volume settled</p>
          <p className="text-xl font-bold tnum">{rupiah(settledVolume)}</p>
        </div>
        <div className="ml-auto text-right">
          <p className="text-xs text-mut">{stores} stores</p>
          <div className="mt-1 flex items-center justify-end gap-1.5">
            <Users size={14} className="text-mut" />
          </div>
        </div>
      </Card>

      {/* Recent transactions */}
      <div>
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-semibold text-mut">Recent</h2>
          <Link to="/transactions" className="text-xs text-mint hover:underline">
            View all
          </Link>
        </div>
        <div className="space-y-2">
          {txs.slice(0, 8).map((tx) => (
            <Card key={tx.transaction_id} className="flex items-center gap-3 p-3">
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-raised">
                <ArrowLeftRight size={14} className="text-mut" />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium truncate">
                  {tx.reference || tx.transaction_id.slice(0, 8)}
                </p>
                <p className="text-xs text-mut">{timeAgo(tx.created_at)}</p>
              </div>
              <div className="text-right">
                <p className="text-sm font-semibold tnum">
                  {rupiah(tx.amount)}
                </p>
                <Badge tone={statusTone[tx.status]}>{tx.status}</Badge>
              </div>
            </Card>
          ))}
          {txs.length === 0 && (
            <p className="py-8 text-center text-sm text-mut">
              No transactions yet. Create a store and start accepting payments.
            </p>
          )}
        </div>
      </div>
    </div>
  );
}

function StatCard({
  icon: Icon,
  label,
  value,
  tone,
}: {
  icon: React.ElementType;
  label: string;
  value: string;
  tone: "sky" | "amber" | "mint" | "crimson";
}) {
  const bg = {
    sky: "bg-sky-dim text-sky",
    amber: "bg-amber-dim text-amber",
    mint: "bg-mint-dim text-mint",
    crimson: "bg-crimson-dim text-crimson",
  };
  return (
    <Card className="p-3.5">
      <div className={`mb-2 inline-flex rounded-lg p-1.5 ${bg[tone]}`}>
        <Icon size={16} />
      </div>
      <p className="text-2xl font-bold tnum">{value}</p>
      <p className="text-xs text-mut">{label}</p>
    </Card>
  );
}