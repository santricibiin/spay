import { useEffect, useState, useRef } from "react";
import {
  Zap,
  ZapOff,
  Smartphone,
  KeyRound,
  LogOut,
  Upload,
  QrCode,
  Check,
  Loader2,
} from "lucide-react";
import { api, getToken, type ProviderStatus } from "../api/client";
import { Card } from "../components/ui/Card";
import { Badge } from "../components/ui/Badge";
import { Button } from "../components/ui/Button";
import { Input } from "../components/ui/Input";

export default function Provider() {
  const [status, setStatus] = useState<ProviderStatus | null>(null);
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [requestId, setRequestId] = useState<string | null>(null);
  const [otp, setOtp] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const [qrString, setQrString] = useState("");
  const [uploading, setUploading] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    api.get<ProviderStatus>("/admin/provider/status").then(setStatus);
  }, []);

  async function requestOtp() {
    if (!phone) return;
    setError("");
    setLoading(true);
    try {
      const res = await api.post<{ request_id: string }>(
        "/admin/provider/otp",
        { phone, password },
      );
      setRequestId(res.request_id);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to send OTP");
    } finally {
      setLoading(false);
    }
  }

  async function verifyOtp() {
    if (!requestId || !otp) return;
    setError("");
    setLoading(true);
    try {
      await api.post("/admin/provider/verify", {
        request_id: requestId,
        otp,
      });
      setStatus({ authenticated: true });
      setRequestId(null);
      setOtp("");
      setPassword("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Verification failed");
    } finally {
      setLoading(false);
    }
  }

  async function uploadQr(file: File) {
    setUploading(true);
    setError("");
    try {
      const fd = new FormData();
      fd.append("qr", file);
      const res = await fetch("/api/admin/provider/qris", {
        method: "POST",
        headers: { Authorization: `Bearer ${getToken()}` },
        body: fd,
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error ?? "Upload failed");
      setQrString(data.qr_string);
    } catch (err) {
      setError(err instanceof Error ? err.message : "QR upload failed");
    } finally {
      setUploading(false);
    }
  }

  async function logout() {
    await api.post("/admin/provider/logout");
    setStatus({ authenticated: false });
    setQrString("");
  }

  return (
    <div className="space-y-5">
      <h1 className="text-lg font-semibold tracking-tight">Provider</h1>

      <Card className="p-5">
        <div className="flex items-center gap-3">
          <div
            className={`flex h-12 w-12 items-center justify-center rounded-2xl ${
              status?.authenticated
                ? "bg-mint-dim text-mint"
                : "bg-crimson-dim text-crimson"
            }`}
          >
            {status?.authenticated ? <Zap size={22} /> : <ZapOff size={22} />}
          </div>
          <div className="flex-1">
            <p className="text-sm font-semibold">Shopee</p>
            <div className="mt-1">
              <Badge tone={status?.authenticated ? "mint" : "crimson"}>
                {status?.authenticated ? "Connected" : "Disconnected"}
              </Badge>
            </div>
          </div>
          {status?.authenticated && (
            <Button variant="danger" onClick={logout}>
              <LogOut size={15} />
              Logout
            </Button>
          )}
        </div>

        {status !== null && !status.authenticated && (
          <div className="mt-5 border-t border-line pt-5">
            {!requestId ? (
              <div className="space-y-3">
                <label className="block text-xs font-medium text-mut">
                  Phone number
                </label>
                <div className="relative">
                  <Smartphone
                    size={16}
                    className="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 text-faint"
                  />
                  <Input
                    value={phone}
                    onChange={(e) => setPhone(e.target.value)}
                    placeholder="081234567890"
                    className="pl-10"
                    inputMode="tel"
                  />
                </div>

                <label className="block text-xs font-medium text-mut">
                  Password{" "}
                  <span className="text-faint">(opsional, jika akun di-password)</span>
                </label>
                <div className="relative">
                  <KeyRound
                    size={16}
                    className="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 text-faint"
                  />
                  <Input
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="Password akun Shopee"
                    className="pl-10"
                  />
                </div>

                <Button onClick={requestOtp} loading={loading} className="w-full">
                  Request OTP
                </Button>
              </div>
            ) : (
              <div className="space-y-3">
                <label className="block text-xs font-medium text-mut">
                  OTP dikirim ke {phone}
                </label>
                <div className="relative">
                  <KeyRound
                    size={16}
                    className="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 text-faint"
                  />
                  <Input
                    value={otp}
                    onChange={(e) => setOtp(e.target.value)}
                    placeholder="123456"
                    className="pl-10"
                    inputMode="numeric"
                  />
                </div>
                <Button onClick={verifyOtp} loading={loading} className="w-full">
                  Verify & connect
                </Button>
                <Button
                  variant="ghost"
                  onClick={() => setRequestId(null)}
                  className="w-full"
                >
                  Back
                </Button>
              </div>
            )}
            {error && <p className="mt-3 text-sm text-crimson">{error}</p>}
          </div>
        )}

        {status?.authenticated && (
          <div className="mt-5 border-t border-line pt-5">
            <p className="mb-3 flex items-center gap-2 text-xs font-medium text-mut">
              <QrCode size={15} />
              Static QRIS
            </p>

            {qrString ? (
              <div className="rounded-xl border border-mint/25 bg-mint-dim/40 p-3">
                <div className="flex items-center gap-2">
                  <Check size={16} className="text-mint" />
                  <span className="text-sm font-medium text-mint">
                    QRIS terpasang
                  </span>
                </div>
                <code className="mt-2 block break-all text-xs text-mut tnum">
                  {qrString}
                </code>
              </div>
            ) : (
              <p className="mb-3 text-xs text-mut">
                Upload QR code toko kamu — otomatis di-decode jadi string QRIS.
              </p>
            )}

            <input
              ref={fileRef}
              type="file"
              accept="image/*"
              className="hidden"
              onChange={(e) => {
                const f = e.target.files?.[0];
                if (f) uploadQr(f);
                e.target.value = "";
              }}
            />
            <Button
              variant="outline"
              onClick={() => fileRef.current?.click()}
              disabled={uploading}
              className="w-full"
            >
              {uploading ? (
                <>
                  <Loader2 size={16} className="animate-spin" />
                  Decoding QR...
                </>
              ) : (
                <>
                  <Upload size={16} />
                  {qrString ? "Ganti QR" : "Upload QR code"}
                </>
              )}
            </Button>
            {error && <p className="mt-3 text-sm text-crimson">{error}</p>}
          </div>
        )}
      </Card>
    </div>
  );
}