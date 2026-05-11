import type { InputHTMLAttributes, ReactNode, SelectHTMLAttributes, TextareaHTMLAttributes } from "react";

export function Label({ children, htmlFor }: { children: ReactNode; htmlFor?: string }) {
  return (
    <label htmlFor={htmlFor} className="mb-1 block text-xs font-semibold uppercase tracking-wide text-[#94a3b8]">
      {children}
    </label>
  );
}

export function Input(props: InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      {...props}
      className={`w-full rounded-lg border border-[#2a2a3f] bg-[#0d0d12] px-3 py-2 text-[#e2e8f0] outline-none focus:border-[#6366f1] ${props.className ?? ""}`}
    />
  );
}

export function TextArea(props: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return (
    <textarea
      {...props}
      className={`w-full rounded-lg border border-[#2a2a3f] bg-[#0d0d12] px-3 py-2 font-mono text-sm text-[#e2e8f0] outline-none focus:border-[#6366f1] ${props.className ?? ""}`}
    />
  );
}

export function Select(props: SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select
      {...props}
      className={`w-full rounded-lg border border-[#2a2a3f] bg-[#0d0d12] px-3 py-2 text-[#e2e8f0] outline-none focus:border-[#6366f1] ${props.className ?? ""}`}
    />
  );
}
