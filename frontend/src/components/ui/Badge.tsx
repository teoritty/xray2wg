const map: Record<string, string> = {
  running: "text-[#22c55e]",
  stopped: "text-[#64748b]",
  error: "text-[#ef4444]",
  active: "text-[#22c55e]",
  inactive: "text-[#64748b]",
};

export function StatusBadge({ status }: { status?: string }) {
  const s = status ?? "unknown";
  const cls = map[s] ?? "text-[#94a3b8]";
  return (
    <span className={`inline-flex items-center gap-1 text-sm ${cls}`}>
      <span aria-hidden className="h-2 w-2 rounded-full bg-current" /> {s}
    </span>
  );
}
