import type { ReactNode } from "react";

export function Card({ children, className = "" }: { children: ReactNode; className?: string }) {
  return (
    <div className={`rounded-xl border border-[#2a2a3f] bg-[#161620] p-5 ${className}`}>{children}</div>
  );
}
