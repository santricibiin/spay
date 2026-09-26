import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Plus, Store, Globe, Copy, Check, Trash2 } from "lucide-react";
import { api, type Store as StoreType } from "../api/client";
import { Card } from "../components/ui/Card";
import { Badge } from "../components/ui/Badge";
import { Button } from "../components/ui/Button";
import { Input } from "../components/ui/Input";
import { Modal } from "../components/ui/Modal";

export default function Stores() {
  const [stores, setStores] = useState<StoreType[]>([]);
  const [showAdd, setShowAdd] = useState(false);
  const [newKey, setNewKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);
  const navigate = useNavigate();

  useEffect(() => {
    api.get<StoreType[]>("/admin/stores").then(setStores);
  }, []);

  async function copyKey(k: string) {
    await navigator.clipboard.writeText(k);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  async function doDelete() {
    if (!deleteId) return;
    setDeleting(true);
    try {
      await api.delete(`/admin/stores/${deleteId}`);
      setStores((s) => s.filter((st) => st.id !== deleteId));
      setDeleteId(null);
    } finally {
      setDeleting(false);
    }
  }

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-semibold tracking-tight">Stores</h1>
        <Button onClick={() => setShowAdd(true)}>
          <Plus size={16} />
          Add Store
        </Button>
      </div>

      <div className="grid gap-3 sm:grid-cols-2">
        {stores.map((st) => (
          <div key={st.id} className="relative">
            <Card
              className="flex cursor-pointer items-start gap-3 p-4 transition-colors hover:border-mint/30"
              onClick={() => navigate(`/panel/stores/${st.id}`)}
            >
              <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-raised">
                <Store size={18} className="text-mut" />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-semibold truncate">{st.name}</p>
                <div className="mt-1 flex items-center gap-1.5 text-xs text-mut">
                  <Globe size={12} />
                  <span className="truncate">
                    {st.webhook_url
                      ? new URL(st.webhook_url).hostname
                      : "no webhook"}
                  </span>
                </div>
                <div className="mt-2">
                  <Badge tone={st.active ? "mint" : "muted"}>
                    {st.active ? "active" : "inactive"}
                  </Badge>
                </div>
              </div>
            </Card>
            <button
              onClick={(e) => {
                e.stopPropagation();
                setDeleteId(st.id);
              }}
              className="absolute right-3 top-3 rounded-lg p-1.5 text-faint hover:bg-crimson-dim hover:text-crimson"
            >
              <Trash2 size={15} />
            </button>
          </div>
        ))}
        {stores.length === 0 && (
          <div className="col-span-full py-12 text-center text-sm text-mut">
            <Store size={32} className="mx-auto mb-2 text-faint" />
            No stores yet. Add your first store to start accepting payments.
          </div>
        )}
      </div>

      {/* Delete confirm */}
      <Modal
        open={!!deleteId}
        onClose={() => setDeleteId(null)}
        title="Delete store"
      >
        <p className="mb-4 text-sm text-mut">
          This permanently deletes the store and all its transactions &amp;
          webhook logs. This cannot be undone.
        </p>
        <Button
          variant="danger"
          onClick={doDelete}
          loading={deleting}
          className="w-full"
        >
          Delete permanently
        </Button>
      </Modal>

      {/* Add Store Modal */}
      <Modal
        open={showAdd}
        onClose={() => {
          setShowAdd(false);
          setNewKey(null);
        }}
        title="Add Store"
      >
        <AddStoreForm
          onCreated={(key) => {
            setNewKey(key);
            api.get<StoreType[]>("/admin/stores").then(setStores);
          }}
        />
      </Modal>

      {/* Key reveal modal */}
      {newKey && (
        <Modal
          open={!!newKey}
          onClose={() => setNewKey(null)}
          title="Store created — save this key"
        >
          <div className="space-y-4">
            <p className="text-sm text-mut">
              This server key is shown only once. Copy it now and store it
              securely.
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
              I've saved it
            </Button>
          </div>
        </Modal>
      )}
    </div>
  );
}

function AddStoreForm({
  onCreated,
}: {
  onCreated: (key: string) => void;
}) {
  const [name, setName] = useState("");
  const [webhook, setWebhook] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name) return;
    setError("");
    setLoading(true);
    try {
      const res = await api.post<{
        store: StoreType;
        server_key: string;
      }>("/admin/stores", { name, webhook_url: webhook });
      onCreated(res.server_key);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={onSubmit} className="space-y-4">
      <div>
        <label className="mb-1 block text-xs font-medium text-mut">
          Store name
        </label>
        <Input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="My Store"
          required
        />
      </div>
      <div>
        <label className="mb-1 block text-xs font-medium text-mut">
          Webhook URL{" "}
          <span className="text-faint">(opsional, bisa diisi nanti)</span>
        </label>
        <Input
          value={webhook}
          onChange={(e) => setWebhook(e.target.value)}
          placeholder="https://store.com/webhook"
        />
      </div>
      {error && <p className="text-sm text-crimson">{error}</p>}
      <Button type="submit" loading={loading} className="w-full">
        Create store
      </Button>
    </form>
  );
}