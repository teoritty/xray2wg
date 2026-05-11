import type { ReactNode } from "react";

export function EmptyState({
  title,
  description,
  action,
}: {
  title: string;
  description?: ReactNode;
  action?: ReactNode;
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-4 py-16 text-center">
      <svg width="96" height="96" viewBox="0 0 96 96" fill="none" className="text-[#2a2a3f]">
        <rect x="8" y="20" width="80" height="56" rx="8" stroke="currentColor" strokeWidth="2" />
        <path d="M28 52h40M36 44h24" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
      </svg>
      <div className="flex max-w-md flex-col gap-2">
        <p className="text-lg font-medium text-[#e2e8f0]">{title}</p>
        {description ? <p className="text-sm leading-relaxed text-[#64748b]">{description}</p> : null}
      </div>
      {action}
    </div>
  );
}
