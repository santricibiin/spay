import type { InputHTMLAttributes } from "react";

export function Input({
  className = "",
  ...rest
}: InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      className={`w-full rounded-xl border border-line bg-raised px-3.5 py-2.5 text-sm text-fg placeholder:text-faint outline-none transition-colors focus:border-mint/50 focus:ring-2 focus:ring-mint/15 ${className}`}
      {...rest}
    />
  );
}