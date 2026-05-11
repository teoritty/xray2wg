import type { ReactNode } from "react";

type CodeFenceProps = {
  /** Shown like a markdown fence label (e.g. `ini`, `routeros`). */
  language?: string;
  code: string;
  /** Toolbar on the right (copy, download, etc.). */
  actions?: ReactNode;
  className?: string;
};

/**
 * Fenced-code-block style container for monospace config or CLI output.
 * Passes through `code` as plain text only (no markdown parsing).
 */
export function CodeFence({ language, code, actions, className = "" }: CodeFenceProps) {
  return (
    <div
      className={`overflow-hidden rounded-lg border border-[#2a2a3f] bg-[#0d0d12] shadow-inner ${className}`}
    >
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-[#2a2a3f] bg-[#12121a] px-3 py-2">
        {language ? (
          <span className="font-mono text-xs tracking-wide text-[#64748b]">{language}</span>
        ) : (
          <span />
        )}
        {actions ? <div className="flex flex-wrap items-center gap-2">{actions}</div> : null}
      </div>
      <pre className="max-h-[min(60vh,520px)] overflow-auto p-4 font-mono text-xs leading-relaxed text-[#e2e8f0]">
        <code>{code}</code>
      </pre>
    </div>
  );
}
