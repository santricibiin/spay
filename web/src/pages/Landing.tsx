import { useEffect, useRef, useState } from "react";
import {
  Hexagon,
  QrCode,
  Webhook,
  KeyRound,
  Clock,
  ShieldCheck,
  Zap,
  ArrowDown,
  Store,
  PlugZap,
  Check,
} from "lucide-react";

// ---- scroll-reveal hook ----

function useReveal() {
  const ref = useRef<HTMLDivElement>(null);
  const [visible, setVisible] = useState(false);
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const obs = new IntersectionObserver(
      ([entry]) => { if (entry.isIntersecting) setVisible(true); },
      { threshold: 0.15, rootMargin: "0px 0px -40px 0px" },
    );
    obs.observe(el);
    return () => obs.disconnect();
  }, []);
  return { ref, visible };
}

function Reveal({ children, className }: { children: React.ReactNode; className?: string }) {
  const { ref, visible } = useReveal();
  return (
    <div
      ref={ref}
      className={`transition-all duration-700 ease-out ${
        visible ? "translate-y-0 opacity-100" : "translate-y-8 opacity-0"
      } ${className ?? ""}`}
    >
      {children}
    </div>
  );
}

// ---- page ----

export default function Landing() {
  return (
    <div className="min-h-screen bg-bg text-fg">
      <LandingNav />
      <Hero />
      <Features />
      <HowItWorks />
      <Providers />
      <Footer />
    </div>
  );
}

function LandingNav() {
  return (
    <header className="sticky top-0 z-40 border-b border-line-soft bg-bg/80 backdrop-blur-xl">
      <div className="mx-auto flex h-16 max-w-5xl items-center justify-between px-5">
        <a href="#top" className="flex items-center gap-2.5">
          <Hexagon size={22} className="text-mint" />
          <span className="text-base font-semibold tracking-tight">
            PayGateMe
          </span>
        </a>
        <a
          href="#cara-kerja"
          className="rounded-xl bg-mint px-4 py-2 text-sm font-semibold text-[#04211a] transition-colors hover:brightness-110"
        >
          Cara kerja
        </a>
      </div>
    </header>
  );
}

function Hero() {
  const { ref, visible } = useReveal();
  return (
    <section ref={ref} id="top" className="relative overflow-hidden">
      <div
        className={`pointer-events-none absolute left-1/2 top-[-200px] h-80 w-80 -translate-x-1/2 rounded-full opacity-15 blur-3xl transition-opacity duration-1000 ${visible ? "opacity-20" : "opacity-0"}`}
        style={{ background: "var(--color-mint)" }}
      />

      <div className="relative mx-auto grid max-w-5xl gap-10 px-5 pb-16 pt-16 md:grid-cols-2 md:items-center md:pb-20 md:pt-24">
        <div>
          <h1 className="text-4xl font-bold leading-tight tracking-tight md:text-5xl">
            Terima pembayaran{" "}
            <span className="text-mint">QRIS</span>{" "}
            lewat satu API.
          </h1>
          <p className="mt-5 max-w-md text-base leading-relaxed text-mut">
            Hubungkan akun Shopee kamu, dapat QR dinamis per transaksi, dan
            terima notifikasi realtime — tanpa bikin infrastruktur sendiri.
          </p>
          <div className="mt-7">
            <a
              href="#cara-kerja"
              className="inline-flex items-center gap-2 rounded-xl bg-mint px-6 py-3 text-sm font-semibold text-[#04211a] transition-colors hover:brightness-110"
            >
              Mulai
              <ArrowDown size={16} />
            </a>
          </div>
        </div>

        <div className="rounded-2xl border border-line bg-surface p-4 shadow-2xl shadow-black/30">
          <div className="rounded-lg border border-line-soft bg-raised p-4">
            <pre className="overflow-x-auto font-mono text-[13px] leading-relaxed">
              <code>
                <span className="text-sky">curl</span>{" "}
                <span className="text-amber">https://paygateme.com/api/v1/</span>
                <span className="text-mint">transactions</span>
                {"\n"}  <span className="faint">-H</span>{" "}
                <span className="text-mint">"Authorization: Bearer pg_live_..."</span>
                {"\n"}  <span className="text-faint">-H</span>{" "}
                <span className="text-mint">"Idempotency-Key: order-001"</span>
                {"\n"}  <span className="text-faint">-d</span>{" "}
                <span className="text-mint">{`{"amount": 50000, "reference": "order-001"}`}</span>
                {"\n\n"}
                <span className="text-mut">{`// → 201 { "status": "pending",`}</span>
                {"\n"}
                <span className="text-mut">{`//   "unique_amount": 50001,`}</span>
                {"\n"}
                <span className="text-mut">{`//   "qr_string": "00020101..." }`}</span>
              </code>
            </pre>
          </div>
        </div>
      </div>
    </section>
  );
}

// ---- features ----

const features = [
  {
    icon: QrCode,
    title: "QRIS dinamis per transaksi",
    desc: "Setiap pembayaran dapat QR unik dengan nominal presisi. Auto-expire 5 menit.",
  },
  {
    icon: Webhook,
    title: "Webhook realtime",
    desc: "Notifikasi settlement & expired langsung ke endpoint kamu — signature HMAC, retry otomatis.",
  },
  {
    icon: KeyRound,
    title: "Satu key per store",
    desc: "Generate, rotate, dan revoke server key untuk setiap store secara independen.",
  },
  {
    icon: Clock,
    title: "Idempotensi built-in",
    desc: "Header Idempotency-Key mencegah transaksi ganda saat network retry.",
  },
  {
    icon: ShieldCheck,
    title: "Verifikasi signature",
    desc: "Setiap webhook ditandatangani HMAC-SHA256 — verifikasi sisi store dengan server key kamu.",
  },
  {
    icon: Zap,
    title: "Matching otomatis",
    desc: "Transaksi masuk dicocokkan nominal unik — langsung settle begitu pembeli bayar.",
  },
];

function Features() {
  return (
    <section className="border-t border-line-soft">
      <div className="mx-auto max-w-5xl px-5 py-16 md:py-20">
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {features.map((f, i) => (
            <StaggerCard key={f.title} delay={i}>
              <div className="mb-3 inline-flex rounded-xl bg-mint/10 p-2 text-mint">
                <f.icon size={18} />
              </div>
              <h3 className="mb-1.5 text-sm font-semibold">{f.title}</h3>
              <p className="text-sm leading-relaxed text-mut">{f.desc}</p>
            </StaggerCard>
          ))}
        </div>
      </div>
    </section>
  );
}

function StaggerCard({ children, delay }: { children: React.ReactNode; delay: number }) {
  const { ref, visible } = useReveal();
  return (
    <div
      ref={ref}
      className="rounded-2xl border border-line-soft bg-surface p-5 transition-all duration-700 ease-out hover:border-mint/25"
      style={{
        opacity: visible ? 1 : 0,
        transform: visible ? "translateY(0)" : "translateY(16px)",
        transitionDelay: `${delay * 80}ms`,
      }}
    >
      {children}
    </div>
  );
}

// ---- how it works ----

const steps = [
  {
    n: "1",
    icon: Store,
    title: "Daftarkan store",
    desc: "Dapat server key dan atur webhook URL. Sekali setup.",
  },
  {
    n: "2",
    icon: QrCode,
    title: "Buat transaksi",
    desc: "Panggil API dengan nominal & referensi. QR langsung terbit.",
  },
  {
    n: "3",
    icon: Webhook,
    title: "Terima notifikasi",
    desc: "Pembeli bayar → status settlement terkirim ke webhook kamu.",
  },
];

function HowItWorks() {
  return (
    <section id="cara-kerja" className="border-t border-line-soft bg-surface/30">
      <div className="mx-auto max-w-5xl px-5 py-16 md:py-20">
        <Reveal>
          <h2 className="mb-10 text-2xl font-bold tracking-tight">
            Tiga langkah.
          </h2>
        </Reveal>

        <div className="grid gap-4 md:grid-cols-3">
          {steps.map((s, i) => (
            <StaggerCard key={s.n} delay={i}>
              <div className="mb-4 flex items-center justify-between">
                <div className="rounded-lg bg-raised p-2 text-mint">
                  <s.icon size={18} />
                </div>
                <span className="font-mono text-lg font-bold text-line">
                  {s.n}
                </span>
              </div>
              <h3 className="mb-1.5 text-sm font-semibold">{s.title}</h3>
              <p className="text-sm leading-relaxed text-mut">{s.desc}</p>
            </StaggerCard>
          ))}
        </div>
      </div>
    </section>
  );
}

// ---- providers ----

function Providers() {
  return (
    <section className="border-t border-line-soft">
      <div className="mx-auto max-w-5xl px-5 py-16 md:py-20">
        <Reveal>
          <h2 className="mb-8 text-2xl font-bold tracking-tight">
            Provider
          </h2>
        </Reveal>

        <div className="grid gap-4 md:grid-cols-2">
          <StaggerCard delay={0}>
            <div className="mb-3 flex items-center gap-3">
              <div className="rounded-xl bg-mint/15 p-2 text-mint">
                <PlugZap size={18} />
              </div>
              <h3 className="text-base font-semibold">Shopee</h3>
              <span className="ml-auto inline-flex items-center gap-1 rounded-full bg-mint/15 px-2 py-0.5 text-xs font-medium text-mint">
                <Check size={11} />
                Ready
              </span>
            </div>
            <p className="text-sm leading-relaxed text-mut">
              Login OTP ke akun merchant, QRIS dinamis dari QR toko, feed
              transaksi realtime.
            </p>
          </StaggerCard>

          <StaggerCard delay={1}>
            <div className="mb-3 flex items-center gap-3">
              <div className="rounded-xl bg-raised p-2 text-mut">
                <PlugZap size={18} />
              </div>
              <h3 className="text-base font-semibold">GoPay</h3>
              <span className="ml-auto rounded-full bg-raised px-2 py-0.5 text-xs font-medium text-mut">
                Soon
              </span>
            </div>
            <p className="text-sm leading-relaxed text-mut">
              OAuth2 + GoID, outlet profile, feed offset-based. Dalam pengembangan.
            </p>
          </StaggerCard>
        </div>
      </div>
    </section>
  );
}

function Footer() {
  return (
    <footer className="border-t border-line-soft">
      <div className="mx-auto flex max-w-5xl flex-col items-center justify-between gap-3 px-5 py-8 text-sm text-faint sm:flex-row">
        <div className="flex items-center gap-2">
          <Hexagon size={16} className="text-mint" />
          <span className="font-medium text-mut">PayGateMe</span>
        </div>
        <p>paygateme.com</p>
      </div>
    </footer>
  );
}