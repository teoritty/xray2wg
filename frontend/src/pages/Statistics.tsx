import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { TrafficChart } from "../components/charts/TrafficChart";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Table } from "../components/ui/Table";
import { useTunnelStatsStream } from "../hooks/useTunnelStatsStream";
import { formatBytes, formatTime } from "../lib/format";
import { peersApi, tunnelsApi, type StatSnapshot } from "../services/api";

type Win = "1h" | "6h" | "24h";

// Backend writes a fresh snapshot every 2s. Refetching every 5s keeps the
// chart live without hammering the server, while the WS stream fills in the
// trailing point so the right edge moves smoothly between historical fetches.
const STATS_REFETCH_MS = 5_000;

export function StatisticsPage() {
  const [win, setWin] = useState<Win>("1h");
  const { data: tunnels = [] } = useQuery({
    queryKey: ["tunnels"],
    queryFn: ({ signal }) => tunnelsApi.list({ signal }),
    refetchInterval: STATS_REFETCH_MS,
  });
  const liveRates = useTunnelStatsStream(true);

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-sm text-[#94a3b8]">Time range</span>
        {(["1h", "6h", "24h"] as Win[]).map((w) => (
          <Button
            key={w}
            variant={win === w ? "primary" : "secondary"}
            className="!px-3 !py-1 text-xs"
            onClick={() => setWin(w)}
          >
            {w}
          </Button>
        ))}
      </div>

      {tunnels.map((t) => (
        <TunnelSection
          key={t.ID}
          tunnelId={t.ID}
          name={t.Name}
          win={win}
          live={liveRates[t.ID]}
        />
      ))}
    </div>
  );
}

function TunnelSection({
  tunnelId,
  name,
  win,
  live,
}: {
  tunnelId: number;
  name: string;
  win: Win;
  live?: { rx: number; tx: number };
}) {
  const [open, setOpen] = useState(true);
  const { data: stats = [] } = useQuery({
    queryKey: ["tunnel-stats", tunnelId, win],
    queryFn: ({ signal }) => tunnelsApi.stats(tunnelId, win, { signal }),
    refetchInterval: STATS_REFETCH_MS,
  });
  const { data: peers = [] } = useQuery({
    queryKey: ["peers", tunnelId],
    queryFn: ({ signal }) => peersApi.list(tunnelId, { signal }),
    refetchInterval: STATS_REFETCH_MS,
  });

  const samples = useMemo(() => {
    const pts = (stats as StatSnapshot[]).map((r) => ({
      t: new Date(r.SampledAt).getTime() / 1000,
      rx: Number(r.RxRate),
      tx: Number(r.TxRate),
    }));
    if (live) {
      pts.push({ t: Date.now() / 1000, rx: live.rx, tx: live.tx });
    }
    return pts;
  }, [stats, live]);

  return (
    <Card>
      <button
        type="button"
        className="mb-4 flex w-full items-center justify-between text-left font-semibold text-[#e2e8f0]"
        onClick={() => setOpen((v) => !v)}
      >
        {name}
        <span className="text-sm text-[#64748b]">{open ? "Hide" : "Show"}</span>
      </button>
      {open ? (
        <div className="space-y-6">
          <TrafficChart samples={samples} height={240} />
          <div>
            <h3 className="mb-2 text-sm uppercase text-[#94a3b8]">Peer activity</h3>
            <Table headers={["Name", "Handshake", "RX", "TX"]}>
              {peers.map((p) => (
                <tr key={p.ID} className="border-b border-[#2a2a3f]">
                  <td className="px-4 py-2">{p.Name}</td>
                  <td className="px-4 py-2 text-[#94a3b8]">{formatTime(p.LastHandshake)}</td>
                  <td className="px-4 py-2 font-mono text-xs">{formatBytes(p.RxBytes)}</td>
                  <td className="px-4 py-2 font-mono text-xs">{formatBytes(p.TxBytes)}</td>
                </tr>
              ))}
            </Table>
          </div>
        </div>
      ) : null}
    </Card>
  );
}
