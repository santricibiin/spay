import { useEffect, useState } from "react";
import { Send, Save, PlugZap, CheckCircle2, AlertTriangle } from "lucide-react";
import { api } from "../api/client";
import { Card } from "../components/ui/Card";
import { Badge } from "../components/ui/Badge";
import { Button } from "../components/ui/Button";
import { Input } from "../components/ui/Input";

interface TelegramCfg {
  bot_token: string;
  paid_chat_id: string;
  admin_chat_id: string;
}

export default function Settings() {
  const [cfg, setCfg] = useState<TelegramCfg>({
    bot_token: "",
    paid_chat_id: "",
    admin_chat_id: "",
  });
  const [loaded, setLoaded] = useState(false);
  const [msg, setMsg] = useState<{ ok: boolean; text: string } | null>(null);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);

  useEffect(() => {
    api.get<TelegramCfg>("/admin/telegram").then((c) => {
      setCfg(c);
      setLoaded(true);
    });
  }, []);

  const enabled = cfg.bot_token !== "" && cfg.paid_chat_id !== "";

  async function save() {
    setSaving(true);
    setMsg(null);
    try {
      await api.put("/admin/telegram", cfg);
      setMsg({ ok: true, text: "Saved — notifier updated instantly." });
    } catch (err) {
      setMsg({ ok: false, text: err instanceof Error ? err.message : "Failed" });
    } finally {
      setSaving(false);
    }
  }

  async function test() {
    setTesting(true);
    setMsg(null);
    try {
      await api.post("/admin/telegram/test");
      setMsg({ ok: true, text: "Test invoice sent to channel. Check Telegram." });
    } catch (err) {
      setMsg({ ok: false, text: err instanceof Error ? err.message : "Test failed" });
    } finally {
      setTesting(false);
    }
  }

  return (
    <div className="space-y-5">
      <h1 className="text-lg font-semibold tracking-tight">Settings</h1>

      {/* Telegram */}
      <Card className="p-5">
        <div className="mb-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Send size={18} className="text-sky" />
            <h2 className="text-sm font-semibold">Telegram notifications</h2>
          </div>
          <Badge tone={enabled ? "mint" : "muted"}>
            {enabled ? "active" : "inactive"}
          </Badge>
        </div>

        {loaded && (
          <div className="space-y-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-mut">
                Bot token{" "}
                <span className="text-faint">(dari @BotFather)</span>
              </label>
              <Input
                value={cfg.bot_token}
                onChange={(e) => setCfg((c) => ({ ...c, bot_token: e.target.value }))}
                placeholder="1234567890:AAH..."
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-mut">
                Paid chat ID{" "}
                <span className="text-faint">(channel — invoice settled)</span>
              </label>
              <Input
                value={cfg.paid_chat_id}
                onChange={(e) => setCfg((c) => ({ ...c, paid_chat_id: e.target.value }))}
placeholder="-100xxxxxxxx"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-mut">
                Admin chat ID{" "}
                <span className="text-faint">(untuk alert session expired)</span>
              </label>
              <Input
                value={cfg.admin_chat_id}
                onChange={(e) => setCfg((c) => ({ ...c, admin_chat_id: e.target.value }))}
                placeholder="xxxxxxxxx"
              />
            </div>

            <div className="flex gap-2">
              <Button onClick={save} loading={saving} className="flex-1">
                <Save size={15} />
                Save
              </Button>
              <Button
                variant="outline"
                onClick={test}
                loading={testing}
                disabled={!enabled}
              >
                <PlugZap size={15} />
                Test connection
              </Button>
            </div>

            {msg && (
              <div
                className={`flex items-start gap-2 rounded-xl border p-3 text-xs ${
                  msg.ok
                    ? "border-mint/25 bg-mint-dim/40 text-mint"
                    : "border-crimson/25 bg-crimson-dim/40 text-crimson"
                }`}
              >
                {msg.ok ? (
                  <CheckCircle2 size={15} className="mt-0.5 shrink-0" />
                ) : (
                  <AlertTriangle size={15} className="mt-0.5 shrink-0" />
                )}
                <span>{msg.text}</span>
              </div>
            )}
          </div>
        )}
      </Card>
    </div>
  );
}