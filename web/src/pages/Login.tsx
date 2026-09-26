import { useState, useEffect, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import {
  Hexagon, Lock, Eye, EyeOff, ArrowRight, RefreshCw, KeyRound,
} from "lucide-react";
import { useAuth } from "../auth/AuthContext";
import { Button } from "../components/ui/Button";
import { Input } from "../components/ui/Input";

async function loadCaptcha(): Promise<{ id: string; url: string }> {
  const res = await fetch("/api/admin/captcha");
  const id = res.headers.get("X-Captcha-Id") ?? "";
  const blob = await res.blob();
  return { id, url: URL.createObjectURL(blob) };
}

export default function Login() {
  const { login } = useAuth(); const navigate = useNavigate();
  const [password, setPassword] = useState(""); const [show, setShow] = useState(false);
  const [captchaId, setCaptchaId] = useState(""); const [captchaUrl, setCaptchaUrl] = useState("");
  const [captchaValue, setCaptchaValue] = useState("");
  const [error, setError] = useState(""); const [loading, setLoading] = useState(false);

  async function refreshCaptcha() {
    try {
      const c = await loadCaptcha();
      setCaptchaId(c.id); setCaptchaUrl(c.url); setCaptchaValue("");
    } catch { setError("Gagal memuat captcha"); }
  }
  useEffect(() => { refreshCaptcha(); }, []);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (!password || !captchaValue) return;
    setError(""); setLoading(true);
    try { await login(password, captchaId, captchaValue); navigate("/panel", { replace: true }); }
    catch (err) { setError(err instanceof Error ? err.message : "Login failed"); refreshCaptcha(); }
    finally { setLoading(false); }
  }

  return (
    <div className="relative flex h-full items-center justify-center overflow-hidden px-5">
      <div className="pointer-events-none absolute left-1/2 top-0 h-72 w-72 -translate-x-1/2 rounded-full opacity-25 blur-3xl" style={{ background: "var(--color-mint)" }} />
      <div className="relative w-full max-w-sm animate-rise">
        <div className="mb-8 flex flex-col items-center gap-3">
          <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-mint/10 text-mint"><Hexagon size={26} /></div>
          <div className="text-center"><h1 className="text-xl font-semibold tracking-tight">PayGateMe</h1><p className="mt-1 text-sm text-mut">Gateway panel</p></div>
        </div>
        <form onSubmit={onSubmit} className="rounded-2xl border border-line-soft bg-surface p-5">
          <label className="mb-1.5 block text-xs font-medium text-mut">Admin password</label>
          <div className="relative">
            <Lock size={16} className="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 text-faint" />
            <Input type={show ? "text" : "password"} value={password} onChange={(e) => setPassword(e.target.value)} placeholder="••••••••" className="pl-10 pr-10" autoFocus />
            <button type="button" onClick={() => setShow((v) => !v)} className="absolute right-3 top-1/2 -translate-y-1/2 text-faint hover:text-mut">{show ? <EyeOff size={16} /> : <Eye size={16} />}</button>
          </div>

          <label className="mb-1.5 mt-4 block text-xs font-medium text-mut">Captcha</label>
          <div className="flex gap-2">
            <div className="relative min-w-0 flex-1">
              <KeyRound size={16} className="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 text-faint" />
              <Input value={captchaValue} onChange={(e) => setCaptchaValue(e.target.value)} placeholder="Ketik kode" className="pl-10 uppercase" autoComplete="off" />
            </div>
            <button type="button" onClick={refreshCaptcha} title="Refresh captcha" className="flex h-11 w-28 shrink-0 items-center justify-center overflow-hidden rounded-xl border border-line bg-raised">
              {captchaUrl ? <img src={captchaUrl} alt="captcha" className="h-full w-full object-cover" /> : <RefreshCw size={16} className="animate-spin text-mut" />}
            </button>
          </div>

          {error && <p className="mt-3 text-sm text-crimson">{error}</p>}

          <Button type="submit" loading={loading} className="mt-4 w-full">Enter dashboard<ArrowRight size={16} /></Button>
        </form>
      </div>
    </div>
  );
}