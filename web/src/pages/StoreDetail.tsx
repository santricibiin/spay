import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  ArrowLeft,
  Globe,
  Copy,
  Check,
  RefreshCw,
  ChevronRight,
  History,
} from "lucide-react";
import {
  api,
  type Store as StoreType,
  type WebhookDelivery,
} from "../api/client";
import { Card } from "../components/ui/Card";
import { Badge } from "../components/ui/Badge";
import { Button } from "../components/ui/Button";
import { Input } from "../components/ui/Input";
import { Modal } from "../components/ui/Modal";
import { timeAgo } from "../lib/format";

export default function StoreDetail() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [store, setStore] = useState<StoreType | null>(null);
  const [logs, setLogs] = useState<WebhookDelivery[]>([]);
  const [showRotate, setShowRotate] = useState(false);
  const [showDelete, setShowDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [newKey, setNewKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [editing, setEditing] = useState(false);
  const [editName, setEditName] = useState("");
  const [editWebhook, setEditWebhook] = useState("");

  useEffect(() => {
    if (!id) return;
    api.get<StoreType[]>("/admin/stores").then((stores) => {
      const s = stores.find((st) => st.id === id);
      if (s) {
        setStore(s);
        setEditName(s.name);
        setEditWebhook(s.webhook_url);
      }
    });
    api.get<WebhookDelivery[]>(`/admin/stores/${id}/webhook-logs`).then(setLogs);
  }, [id]);

  async function rotateKey() {
    if (!id) return;
    const res = await api.post<{ server_key: string }>(
      `/admin/stores/${id}/rotate-key`,
    );
    setNewKey(res.server_key);
    setShowRotate(false);
  }

  async function deleteStore() {
    if (!id) return;
    setDeleting(true);
    try {
      await api.delete(`/admin/stores/${id}`);
      navigate("/panel/stores");
    } catch (err) {
      setDeleting(false);
    }
  }

  async function saveStore() {
    if (!id) return;
    await api.put(`/admin/stores/${id}`, {
      name: editName,
      webhook_url: editWebhook,
    });
    setStore((s) => (s ? { ...s, name: editName, webhook_url: editWebhook } : s));
    setEditing(false);
  }

  async function copyKey(k: string) {
    await navigator.clipboard.writeText(k);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  if (!store) {
    return <p className="py-8 text-center text-sm text-mut">Loading...</p>;
  }

  const toneForDelivery = (s: string) =>
    s === "success" ? "mint" : s === "failed" ? "crimson" : "amber";

  return (
    <div className="space-y-5">
      <button
        onClick={() => navigate("/panel/stores")}
        className="flex items-center gap-1.5 text-sm text-mut hover:text-fg"
      >
        <ArrowLeft size={16} />
        Back to stores
      </button>

      {/* Store card */}
      <Card className="p-4">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-lg font-semibold">{store.name}</h1>
            <div className="mt-1 flex items-center gap-1.5 text-sm text-mut">
              <Globe size={13} />
              <span className="truncate">{store.webhook_url}</span>
            </div>
          </div>
          <Badge tone={store.active ? "mint" : "muted"}>
            {store.active ? "active" : "inactive"}
          </Badge>
        </div>
        <div className="mt-4 flex flex-wrap gap-2">
          <Button variant="outline" onClick={() => setEditing(true)}>
            Edit
          </Button>
          <Button variant="outline" onClick={() => setShowRotate(true)}>
            <RefreshCw size={14} />
            Rotate key
          </Button>
          <Button variant="danger" onClick={() => setShowDelete(true)}>
            Delete
          </Button>
        </div>
      </Card>

      {/* Webhook logs */}
      <div>
        <h2 className="mb-3 flex items-center gap-2 text-sm font-semibold text-mut">
          <History size={15} />
          Webhook deliveries
        </h2>
        <div className="space-y-2">
          {logs.map((d) => (
            <Card key={d.id} className="flex items-center gap-3 p-3">
              <ChevronRight size={14} className="text-faint" />
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium">{d.event}</p>
                <p className="text-xs text-mut">
                  attempt {d.attempt} · {timeAgo(d.created_at)}
                </p>
              </div>
              <Badge tone={toneForDelivery(d.status)}>{d.status}</Badge>
            </Card>
          ))}
          {logs.length === 0 && (
            <p className="py-6 text-center text-xs text-mut">
              No webhook deliveries yet.
            </p>
          )}
        </div>
      </div>

      {/* Edit modal */}
      <Modal
        open={editing}
        onClose={() => setEditing(false)}
        title="Edit Store"
      >
        <div className="space-y-4">
          <div>
            <label className="mb-1 block text-xs font-medium text-mut">
              Name
            </label>
            <Input
              value={editName}
              onChange={(e) => setEditName(e.target.value)}
            />
          </div>
          <div>
            <label className="mb-1 block text-xs font-medium text-mut">
              Webhook URL
            </label>
            <Input
              value={editWebhook}
              onChange={(e) => setEditWebhook(e.target.value)}
            />
          </div>
          <Button onClick={saveStore} className="w-full">
            Save
          </Button>
        </div>
      </Modal>

      {/* Rotate key confirm */}
      <Modal
        open={showRotate}
        onClose={() => setShowRotate(false)}
        title="Rotate server key"
      >
        <p className="mb-4 text-sm text-mut">
          The old key will be invalidated immediately. All store integrations
          must update to the new key.
        </p>
        <Button variant="danger" onClick={rotateKey} className="w-full">
          Confirm rotate
        </Button>
      </Modal>

      {/* Delete confirm */}
      <Modal
        open={showDelete}
        onClose={() => setShowDelete(false)}
        title="Delete store"
      >
        <p className="mb-4 text-sm text-mut">
          This permanently deletes the store, its transactions, and webhook
          logs. This cannot be undone.
        </p>
        <Button
          variant="danger"
          onClick={deleteStore}
          loading={deleting}
          className="w-full"
        >
          Delete permanently
        </Button>
      </Modal>

      {/* New key reveal */}
      {newKey && (
        <Modal
          open={!!newKey}
          onClose={() => setNewKey(null)}
          title="New server key"
        >
          <div className="space-y-4">
            <p className="text-sm text-mut">
              Copy this key now. It will not be shown again.
            </p>
            <div className="flex items-center gap-2 rounded-xl border border-line bg-raised p-3">
              <code className="flex-1 break-all text-xs text-mint tnum">
                {newKey}
              </code>
              <button
                onClick={() => copyKey(newKey)}
                className="shrink-0 rounded-lg p-1.5 text-mut hover:bg-line hover:text-fg"
              >
                {copied ? <Check size={16} className="text-mint" /> : <Copy size={16} />}
              </button>
            </div>
            <Button
              variant="ghost"
              onClick={() => setNewKey(null)}
              className="w-full"
            >
              Done
            </Button>
          </div>
        </Modal>
      )}
    </div>
  );
}