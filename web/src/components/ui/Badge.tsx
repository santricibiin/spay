type Tone = "mint" | "amber" | "crimson" | "sky" | "muted";

const tones: Record<Tone, string> = {
  mint: "bg-mint-dim text-mint border-mint/25",
  amber: "bg-amber-dim text-amber border-amber/25",
  crimson: "bg-crimson-dim text-crimson border-crimson/25",
  sky: "bg-sky-dim text-sky border-sky/25",
  muted: "bg-raised text-mut border-line",
};

export function Badge({
  tone = "muted",
  children,
}: {
  tone?: Tone;
  children: React.ReactNode;
}) {
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full border px-2.5 py-0.5 text-xs font-medium ${tones[tone]}`}
    >
      {children}
    </span>
  );
}