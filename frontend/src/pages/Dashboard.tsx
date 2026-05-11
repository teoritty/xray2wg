import { useQuery } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { SparkLine } from "../components/charts/SparkLine";
import { Card } from "../components/ui/Card";
import { StatusBadge } from "../components/ui/Badge";
import { Table } from "../components/ui/Table";
import { useTunnelStatsStream } from "../hooks/useTunnelStatsStream";
import { formatBytes, formatBytesRate, formatTime } from "../lib/format";
import { summaryApi, tunnelsApi } from "../services/api";

const MAX_SPARK = 40;

export function Dashboard() {
  const { data: summary } = useQuery({
    queryKey: ["summary"],
    queryFn: ({ signal }) => summaryApi.get({ signal }),
    refetchInterval: 5000,
  });
  const { data: tunnels } = useQuery({
    queryKey: ["tunnels"],
    queryFn: ({ signal }) => tunnelsApi.list({ signal }),
    refetchInterval: 5000,
  });

  const wsRates = useTunnelStatsStream(true);

  const [bufs, setBufs] = useState<Record<number, { rx: number[]; tx: number[] }>>({});

  useEffect(() => {
    setBufs((prev) => {
      const next = { ...prev };
      for (const tid of Object.keys(wsRates)) {
        const id = Number(tid);
        const { rx, tx } = wsRates[id] ?? { rx: 0, tx: 0 };
        const b = next[id] ?? { rx: [], tx: [] };
        b.rx = [...b.rx, rx].slice(-MAX_SPARK);
        b.tx = [...b.tx, tx].slice(-MAX_SPARK);
        next[id] = b;
      }
      return next;
    });
  }, [wsRates]);

  const activeTunnels = useMemo(
    () => (tunnels ?? []).filter((t) => t.Status === "running"),
    [tunnels]
  );

  const events = summary?.events ?? [];

  return (
    <div className="space-y-8">
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <p className="text-xs uppercase text-[#94a3b8]">Active tunnels</p>
          <p className="mt-2 text-2xl font-semibold text-[#e2e8f0]">{summary?.active_tunnels ?? "—"}</p>
        </Card>
        <Card>
          <p className="text-xs uppercase text-[#94a3b8]">Total peers</p>
          <p className="mt-2 text-2xl font-semibold text-[#e2e8f0]">{summary?.total_peers ?? "—"}</p>
        </Card>
        <Card>
          <p className="text-xs uppercase text-[#94a3b8]">Total RX</p>
          <p className="mt-2 text-2xl font-semibold text-[#e2e8f0]">{formatBytes(summary?.total_rx ?? 0)}</p>
        </Card>
        <Card>
          <p className="text-xs uppercase text-[#94a3b8]">Total TX</p>
          <p className="mt-2 text-2xl font-semibold text-[#e2e8f0]">{formatBytes(summary?.total_tx ?? 0)}</p>
        </Card>
      </div>

      <div>
        <h2 className="mb-4 text-sm font-semibold uppercase tracking-wide text-[#94a3b8]">Active tunnels</h2>
        {activeTunnels.length === 0 ? (
          <Card>
            <p className="text-[#94a3b8]">No running tunnels. Start one from the Tunnels page.</p>
          </Card>
        ) : (
          <Table
            headers={["Name", "Status", "Rates", "Trend"]}
          >
            {activeTunnels.map((t) => {
              const tr = summary?.tunnel_rates?.[String(t.ID)];
              const b = bufs[t.ID];
              return (
                <tr key={t.ID} className="border-b border-[#2a2a3f] hover:bg-[#1e1e2e]/50">
                  <td className="px-4 py-3">
                    <Link className="text-[#6366f1] hover:underline" to={`/tunnels/${t.ID}`}>
                      {t.Name}
                    </Link>
                  </td>
                  <td className="px-4 py-3">
                    <StatusBadge status={t.Status} />
                  </td>
                  <td className="px-4 py-3 text-sm text-[#94a3b8]">
                    {tr ? (
                      <>
                        {formatBytesRate(tr[0])} / {formatBytesRate(tr[1])}
                      </>
                    ) : (
                      "—"
                    )}
                  </td>
                  <td className="px-4 py-3 w-40">
                    <SparkLine rx={b?.rx ?? []} tx={b?.tx ?? []} height={36} />
                  </td>
                </tr>
              );
            })}
          </Table>
        )}
      </div>

      <div>
        <h2 className="mb-4 text-sm font-semibold uppercase tracking-wide text-[#94a3b8]">Recent events</h2>
        <Card>
          {events.length === 0 ? (
            <p className="text-[#94a3b8]">No events yet.</p>
          ) : (
            <ul className="space-y-2 text-sm">
              {events.slice(0, 20).map((e, i) => (
                <li key={i} className="flex gap-4 border-b border-[#2a2a3f] py-2 last:border-0">
                  <span className="font-mono text-xs text-[#64748b]">{formatTime(e.ts)}</span>
                  <span className="text-[#94a3b8]">{e.level}</span>
                  <span className="text-[#e2e8f0]">{e.message}</span>
                </li>
              ))}
            </ul>
          )}
        </Card>
      </div>
    </div>
  );
}
