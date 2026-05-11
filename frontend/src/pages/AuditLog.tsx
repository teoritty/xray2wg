import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { auditApi, type AuditLogEntry } from "../services/api";

const LEVELS = [
  { value: "", label: "All" },
  { value: "warn", label: "Warn+" },
  { value: "error", label: "Error" },
];

const PAGE_SIZE = 50;

function LevelBadge({ level }: { level: string }) {
  const cfg: Record<string, string> = {
    error: "bg-[#ef4444]/15 text-[#ef4444] border border-[#ef4444]/30",
    warn: "bg-[#f59e0b]/15 text-[#f59e0b] border border-[#f59e0b]/30",
    info: "bg-[#6366f1]/15 text-[#6366f1] border border-[#6366f1]/30",
  };
  const cls = cfg[level] ?? "bg-[#334155]/40 text-[#94a3b8] border border-[#334155]";
  return (
    <span className={`inline-flex items-center rounded px-2 py-0.5 text-xs font-semibold uppercase tracking-wider ${cls}`}>
      {level}
    </span>
  );
}

function formatTs(ts: string) {
  try {
    const d = new Date(ts);
    return d.toLocaleString("ru-RU", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  } catch {
    return ts;
  }
}

export function AuditLogPage() {
  const [level, setLevel] = useState("");
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [page, setPage] = useState(0);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(0);
    }, 300);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [search]);

  useEffect(() => {
    setPage(0);
  }, [level]);

  const { data, isLoading, error } = useQuery({
    queryKey: ["audit", level, debouncedSearch, page],
    queryFn: ({ signal }) =>
      auditApi.list({ level, search: debouncedSearch, limit: PAGE_SIZE, offset: page * PAGE_SIZE }, { signal }),
    refetchInterval: 30_000,
    placeholderData: (prev) => prev,
  });

  const items: AuditLogEntry[] = data?.items ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <div className="flex flex-col gap-6">
      {/* Header */}
      <div>
        <h1 className="text-xl font-semibold text-[#e2e8f0]">Audit Log</h1>
        <p className="mt-1 text-sm text-[#64748b]">
          System warnings and errors. Auto-refreshes every 30 seconds.
        </p>
      </div>

      {/* Filters row */}
      <div className="flex flex-wrap items-center gap-3">
        {/* Level pills */}
        <div className="flex gap-1 rounded-lg border border-[#2a2a3f] bg-[#1a1a2e] p-1">
          {LEVELS.map((l) => (
            <button
              key={l.value}
              type="button"
              onClick={() => setLevel(l.value)}
              className={`rounded px-3 py-1 text-sm font-medium transition ${
                level === l.value
                  ? "bg-[#6366f1] text-white"
                  : "text-[#64748b] hover:text-[#e2e8f0]"
              }`}
            >
              {l.label}
            </button>
          ))}
        </div>

        {/* Search */}
        <div className="relative flex-1 min-w-[200px] max-w-[400px]">
          <svg
            className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[#475569]"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.5"
          >
            <path d="M21 21l-4.35-4.35M17 11A6 6 0 115 11a6 6 0 0112 0z" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
          <input
            type="text"
            placeholder="Search messages…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-lg border border-[#2a2a3f] bg-[#1a1a2e] py-2 pl-9 pr-3 text-sm text-[#e2e8f0] placeholder-[#475569] focus:border-[#6366f1] focus:outline-none"
          />
        </div>

        {/* Total count */}
        {!isLoading && (
          <span className="ml-auto text-sm text-[#475569]">
            {total} {total === 1 ? "entry" : "entries"}
          </span>
        )}
      </div>

      {/* Table */}
      <div className="overflow-hidden rounded-xl border border-[#2a2a3f] bg-[#161620]">
        {isLoading && items.length === 0 ? (
          <div className="flex items-center justify-center py-16 text-[#475569]">Loading…</div>
        ) : error ? (
          <div className="flex items-center justify-center py-16 text-[#ef4444]">
            Failed to load audit log
          </div>
        ) : items.length === 0 ? (
          <EmptyState />
        ) : (
          <table className="w-full table-fixed text-sm">
            <thead>
              <tr className="border-b border-[#2a2a3f] text-left text-xs font-medium uppercase tracking-wider text-[#475569]">
                <th className="w-44 px-4 py-3">Time</th>
                <th className="w-20 px-4 py-3">Level</th>
                <th className="w-28 px-4 py-3">Source</th>
                <th className="px-4 py-3">Message</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#1e1e2e]">
              {items.map((entry) => (
                <tr key={entry.ID} className="hover:bg-[#1a1a2e] transition-colors">
                  <td className="px-4 py-3 font-mono text-xs text-[#64748b] whitespace-nowrap">
                    {formatTs(entry.CreatedAt)}
                  </td>
                  <td className="px-4 py-3">
                    <LevelBadge level={entry.Level} />
                  </td>
                  <td className="px-4 py-3 text-xs text-[#64748b] truncate">{entry.Source}</td>
                  <td className="px-4 py-3 text-[#cbd5e1] break-all">{entry.Message}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between text-sm text-[#64748b]">
          <span>
            Page {page + 1} of {totalPages}
          </span>
          <div className="flex gap-2">
            <button
              type="button"
              disabled={page === 0}
              onClick={() => setPage((p) => p - 1)}
              className="rounded-lg border border-[#2a2a3f] bg-[#1a1a2e] px-3 py-1.5 text-sm text-[#94a3b8] hover:text-[#e2e8f0] disabled:opacity-40 disabled:cursor-not-allowed transition"
            >
              ← Prev
            </button>
            <button
              type="button"
              disabled={page >= totalPages - 1}
              onClick={() => setPage((p) => p + 1)}
              className="rounded-lg border border-[#2a2a3f] bg-[#1a1a2e] px-3 py-1.5 text-sm text-[#94a3b8] hover:text-[#e2e8f0] disabled:opacity-40 disabled:cursor-not-allowed transition"
            >
              Next →
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

function EmptyState() {
  return (
    <div className="flex flex-col items-center justify-center gap-4 py-20">
      <svg
        className="h-14 w-14 text-[#2a2a3f]"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.2"
      >
        <path d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
      <div className="text-center">
        <p className="text-[#475569]">No log entries found</p>
        <p className="mt-1 text-sm text-[#334155]">Problems will appear here as they occur</p>
      </div>
    </div>
  );
}
