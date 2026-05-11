import type { ButtonHTMLAttributes } from "react";

const variants = {
  primary:
    "bg-[#6366f1] text-white hover:bg-[#5254cc] disabled:opacity-40",
  secondary:
    "bg-[#1e1e2e] text-[#e2e8f0] border border-[#2a2a3f] hover:bg-[#2a2a3f] disabled:opacity-40",
  danger: "bg-[#991b1b] text-white hover:bg-[#ef4444] disabled:opacity-40",
  ghost: "text-[#94a3b8] hover:bg-[#161620] disabled:opacity-40",
};

type Props = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: keyof typeof variants;
};

export function Button({ variant = "primary", className = "", ...rest }: Props) {
  return (
    <button
      type="button"
      className={`rounded-lg px-4 py-2 text-sm font-semibold transition ${variants[variant]} ${className}`}
      {...rest}
    />
  );
}
