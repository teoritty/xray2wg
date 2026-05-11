import { useQueries, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Button } from "../../components/ui/Button";
import { StatusBadge } from "../../components/ui/Badge";
import { EmptyState } from "../../components/ui/EmptyState";
import { Table } from "../../components/ui/Table";
import { useTunnelStatsStream } from "../../hooks/useTunnelStatsStream";
import { formatBytesRate } from "../../lib/format";
import { peersApi, summaryApi, tunnelsApi, type WgInterface } from "../../services/api";

function CreateTunnelFab() {
  return (
    <Link to="/tunnels/new">
      <button
        type="button"
        aria-label="Create tunnel"
        className="fixed bottom-10 right-10 z-10 flex h-14 w-14 items-center justify-center rounded-full bg-[#6366f1] text-2xl text-white shadow-lg hover:bg-[#5254cc]"
      >
        +
      </button>
    </Link>
  );
}

const DELETE_CONFIRM =
  "Associated peers will be removed; ensure it is stopped if needed (server will attempt stop before delete).";

function TunnelRowActions({
  tunnel,
  removePending,
  onDelete,
}: {
  tunnel: WgInterface;
  removePending: boolean;
  onDelete: (id: number) => void;
}) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <Link to={`/tunnels/${tunnel.ID}`}>
        <Button variant="secondary" className="!px-3 !py-1.5 text-xs">
          Open
        </Button>
      </Link>
      <Link to={`/tunnels/${tunnel.ID}/edit`}>
        <Button variant="secondary" className="!px-3 !py-1.5 text-xs">
          Edit
        </Button>
      </Link>
      <Button
        variant="danger"
        className="!px-3 !py-1.5 text-xs"
        disabled={removePending}
        onClick={() => {
          if (confirm(`Delete tunnel "${tunnel.Name}"? ${DELETE_CONFIRM}`)) {
            onDelete(tunnel.ID);
          }
        }}
      >
        Delete
      </Button>
    </div>
  );
}

function TunnelTableRow({
  tunnel,
  peerCount,
  rx,
  tx,
  onToggleRun,
  onDelete,
  removePending,
}: {
  tunnel: WgInterface;
  peerCount: number;
  rx: number;
  tx: number;
  onToggleRun: (tunnel: WgInterface, running: boolean) => void;
  onDelete: (id: number) => void;
  removePending: boolean;
}) {
  return (
    <tr className="border-b border-[#2a2a3f]">
      <td className="px-4 py-3 align-top">
        <Link className="font-medium text-[#6366f1] hover:underline" to={`/tunnels/${tunnel.ID}`}>
          {tunnel.Name}
        </Link>
        <div className="mt-1 md:hidden">
          <StatusBadge status={tunnel.Status} />
        </div>
      </td>
      <td className="hidden px-4 py-3 align-middle md:table-cell">
        <StatusBadge status={tunnel.Status} />
      </td>
      <td className="whitespace-nowrap px-4 py-3 align-middle">
        <label className="flex cursor-pointer items-center gap-2 text-sm text-[#94a3b8]">
          <span className="hidden sm:inline">Run</span>
          <input
            type="checkbox"
            className="accent-[#6366f1]"
            checked={tunnel.Status === "running"}
            onChange={(e) => onToggleRun(tunnel, e.target.checked)}
          />
        </label>
      </td>
      <td className="hidden px-4 py-3 font-mono text-xs text-[#e2e8f0] sm:table-cell">{tunnel.ListenPort}</td>
      <td className="hidden px-4 py-3 font-mono text-xs text-[#e2e8f0] md:table-cell">{peerCount}</td>
      <td className="min-w-[9rem] px-4 py-3 font-mono text-xs text-[#e2e8f0]">
        <span className="whitespace-nowrap">
          {formatBytesRate(rx)} · {formatBytesRate(tx)}
        </span>
      </td>
      <td className="px-4 py-3 align-middle">
        <TunnelRowActions tunnel={tunnel} removePending={removePending} onDelete={onDelete} />
      </td>
    </tr>
  );
}

export function TunnelListPage() {
  const qc = useQueryClient();
  const { data: tunnels = [], isLoading } = useQuery({
    queryKey: ["tunnels"],
    queryFn: ({ signal }) => tunnelsApi.list({ signal }),
  });
  const { data: summary } = useQuery({
    queryKey: ["summary"],
    queryFn: ({ signal }) => summaryApi.get({ signal }),
    refetchInterval: 5000,
  });
  const rates = useTunnelStatsStream(true);

  const peerCounts = useQueries({
    queries: tunnels.map((t) => ({
      queryKey: ["peers", t.ID],
      queryFn: ({ signal }) => peersApi.list(t.ID, { signal }),
      enabled: tunnels.length > 0,
    })),
    combine: (results) =>
      Object.fromEntries(
        tunnels.map((t, i) => [t.ID, results[i]?.data?.length ?? 0] as const)
      ) as Record<number, number>,
  });

  const start = useMutation({
    mutationFn: (id: number) => tunnelsApi.start(id),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["tunnels"] }),
  });
  const stop = useMutation({
    mutationFn: (id: number) => tunnelsApi.stop(id),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["tunnels"] }),
  });
  const remove = useMutation({
    mutationFn: (id: number) => tunnelsApi.remove(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["tunnels"] });
      void qc.invalidateQueries({ queryKey: ["summary"] });
    },
  });

  const toggleRun = (t: WgInterface, running: boolean) => {
    if (running) start.mutate(t.ID);
    else stop.mutate(t.ID);
  };

  if (isLoading) return <p className="text-[#94a3b8]">Loading…</p>;

  if (tunnels.length === 0) {
    return (
      <>
        <EmptyState
          title="No tunnels yet"
          description="Tunnels connect your WireGuard side to Xray. Create one to configure listen ports, peers, and traffic — then start or stop it anytime from this list."
          action={
            <Link to="/tunnels/new">
              <Button>Create tunnel</Button>
            </Link>
          }
        />
        <CreateTunnelFab />
      </>
    );
  }

  const headers = ["Tunnel", "Status", "Run", "UDP", "Peers", "RX / TX", "Actions"];
  const headerClassNames = [
    undefined,
    "hidden md:table-cell",
    undefined,
    "hidden sm:table-cell",
    "hidden md:table-cell",
    undefined,
    undefined,
  ];

  const mutationError = (start.error ?? stop.error ?? remove.error) as Error | undefined;

  return (
    <div className="space-y-4">
      {mutationError ? <p className="text-sm text-[#ef4444]">{mutationError.message}</p> : null}

      <Table headers={headers} headerClassNames={headerClassNames}>
        {tunnels.map((t) => {
          const tr = summary?.tunnel_rates?.[String(t.ID)];
          const live = rates[t.ID];
          const rx = live?.rx ?? tr?.[0] ?? 0;
          const tx = live?.tx ?? tr?.[1] ?? 0;
          return (
            <TunnelTableRow
              key={t.ID}
              tunnel={t}
              peerCount={peerCounts[t.ID] ?? 0}
              rx={rx}
              tx={tx}
              onToggleRun={toggleRun}
              onDelete={(id) => remove.mutate(id)}
              removePending={remove.isPending}
            />
          );
        })}
      </Table>

      <CreateTunnelFab />
    </div>
  );
}
