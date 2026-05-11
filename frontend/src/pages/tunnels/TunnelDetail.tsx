import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { TrafficChart } from "../../components/charts/TrafficChart";
import { NodeHealthBadge } from "../../components/tunnels/NodeHealthBadge";
import { NodeSelector, type SelectedNode } from "../../components/tunnels/NodeSelector";
import { StatusBadge } from "../../components/ui/Badge";
import { Button } from "../../components/ui/Button";
import { Card } from "../../components/ui/Card";
import { PeerTable } from "../../components/peers/PeerTable";
import { useTunnelStatsStream } from "../../hooks/useTunnelStatsStream";
import { formatTime } from "../../lib/format";
import {
  peersApi,
  tunnelsApi,
  type NodeHealthInfo,
  type StatSnapshot,
} from "../../services/api";

type Win = "1h" | "6h" | "24h";
type Tab = "peers" | "nodes" | "statistics" | "info";

const TAB_LABELS: Record<Tab, string> = {
  peers: "Пиры",
  nodes: "Узлы",
  statistics: "Статистика",
  info: "Инфо",
};

export function TunnelDetailPage() {
  const { id } = useParams();
  const tid = Number(id);
  const qc = useQueryClient();
  const [tab, setTab] = useState<Tab>("peers");
  const [win, setWin] = useState<Win>("1h");

  const { data: t, isLoading } = useQuery({
    queryKey: ["tunnel", tid],
    queryFn: ({ signal }) => tunnelsApi.get(tid, { signal }),
    enabled: Number.isFinite(tid),
  });

  const { data: nodesData } = useQuery({
    queryKey: ["tunnel-nodes", tid],
    queryFn: ({ signal }) => tunnelsApi.getNodes(tid, { signal }),
    enabled: Number.isFinite(tid) && tab === "nodes",
    refetchInterval: tab === "nodes" && t?.Status === "running" ? 10_000 : false,
  });

  const { data: peers = [] } = useQuery({
    queryKey: ["peers", tid],
    queryFn: ({ signal }) => peersApi.list(tid, { signal }),
    enabled: Number.isFinite(tid),
  });

  const { data: stats = [] } = useQuery({
    queryKey: ["tunnel-stats", tid, win],
    queryFn: ({ signal }) => tunnelsApi.stats(tid, win, { signal }),
    enabled: Number.isFinite(tid) && tab === "statistics",
    refetchInterval: tab === "statistics" ? 5_000 : false,
  });

  const live = useTunnelStatsStream(tab === "statistics");

  const chartSamples = useMemo(() => {
    const rows = stats as StatSnapshot[];
    const pts = rows.map((r) => ({
      t: new Date(r.SampledAt).getTime() / 1000,
      rx: Number(r.RxRate),
      tx: Number(r.TxRate),
    }));
    const l = live[tid];
    if (l) pts.push({ t: Date.now() / 1000, rx: l.rx, tx: l.tx });
    return pts;
  }, [stats, live, tid]);

  const start = useMutation({
    mutationFn: () => tunnelsApi.start(tid),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["tunnel", tid] }),
  });
  const stop = useMutation({
    mutationFn: () => tunnelsApi.stop(tid),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["tunnel", tid] }),
  });
  const remove = useMutation({
    mutationFn: () => tunnelsApi.remove(tid),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["tunnels"] });
      window.location.hash = "#/tunnels";
    },
  });

  if (!Number.isFinite(tid)) return <p className="text-red-400">Неверный ID туннеля</p>;
  if (isLoading || !t) return <p className="text-slate-400">Загрузка…</p>;

  // Build view data for node tab
  const viewNodes: SelectedNode[] = (nodesData?.nodes ?? []).map((n) => ({
    id: n.id,
    display_name: n.display_name,
    address: n.address,
    port: n.port,
  }));
  const viewHealth: Record<number, NodeHealthInfo> = {};
  (nodesData?.nodes ?? []).forEach((n, i) => { viewHealth[i] = n.health; });

  const allNodesDown =
    nodesData &&
    nodesData.nodes.length > 1 &&
    t.Status === "running" &&
    nodesData.nodes.every((n) => n.health !== null && n.health !== undefined && !n.health.alive);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 className="text-xl font-semibold text-slate-100">{t.Name}</h2>
          <div className="mt-2 flex flex-wrap items-center gap-3">
            <StatusBadge status={t.Status} />
            {t.UptimeStarted ? (
              <span className="text-sm text-slate-500">Запущен {formatTime(t.UptimeStarted)}</span>
            ) : null}
          </div>
          {t.Status === "error" && t.ErrorMessage ? (
            <p className="mt-2 text-sm text-red-400">{t.ErrorMessage}</p>
          ) : null}
          {allNodesDown && (
            <p className="mt-3 rounded-lg border border-red-500/40 bg-red-950/30 px-4 py-2 text-sm text-red-400">
              Все VLESS-узлы недоступны. Туннель запущен, но трафик не проходит.
            </p>
          )}
        </div>
        <div className="flex flex-wrap gap-2">
          {t.Status === "running" ? (
            <Button variant="secondary" onClick={() => stop.mutate()} disabled={stop.isPending}>
              Остановить
            </Button>
          ) : (
            <Button onClick={() => start.mutate()} disabled={start.isPending}>
              Запустить
            </Button>
          )}
          <Link to={`/tunnels/${tid}/edit`}>
            <Button variant="secondary">Изменить</Button>
          </Link>
          <Button
            variant="danger"
            onClick={() => {
              if (confirm(`Удалить туннель ${t.Name}?`)) remove.mutate();
            }}
          >
            Удалить
          </Button>
        </div>
      </div>

      {/* Tabs */}
      <div className="mb-4 flex gap-2 border-b border-slate-800 pb-2">
        {(["peers", "nodes", "statistics", "info"] as Tab[]).map((k) => (
          <button
            key={k}
            type="button"
            className={`rounded-lg px-4 py-2 text-sm font-medium ${
              tab === k ? "bg-slate-800 text-slate-100" : "text-slate-400 hover:bg-slate-900"
            }`}
            onClick={() => setTab(k)}
          >
            {TAB_LABELS[k]}
          </button>
        ))}
      </div>

      {/* Peers tab */}
      {tab === "peers" && (
        <div className="space-y-4">
          <Link to={`/tunnels/${tid}/peers/new`}>
            <Button>Добавить пир</Button>
          </Link>
          <PeerTable
            showTunnelColumn={false}
            rows={peers.map((p) => ({ peer: p, tunnelId: tid }))}
            onAfterDelete={(deletedTid) => {
              void qc.invalidateQueries({ queryKey: ["peers", deletedTid] });
              void qc.invalidateQueries({ queryKey: ["peers-global"] });
            }}
          />
        </div>
      )}

      {/* Nodes tab */}
      {tab === "nodes" && (
        <div className="space-y-4">
          {nodesData ? (
            <>
              <NodeSelector
                selectedNodes={viewNodes}
                availableNodes={[]}
                strategy={nodesData.strategy}
                health={viewHealth}
                readOnly
              />
              {viewNodes.length === 0 && (
                <p className="text-sm text-slate-500">Узлы не назначены.</p>
              )}
            </>
          ) : (
            <p className="text-sm text-slate-500">Загрузка…</p>
          )}
          <Link to={`/tunnels/${tid}/edit`}>
            <Button variant="secondary">Изменить узлы</Button>
          </Link>
          {nodesData && nodesData.nodes.length > 0 && t.Status !== "running" && (
            <p className="text-xs text-slate-600">
              Данные о доступности появятся после запуска туннеля со стратегией «Least Ping».
            </p>
          )}
        </div>
      )}

      {/* Statistics tab */}
      {tab === "statistics" && (
        <div className="space-y-4">
          <div className="flex gap-2">
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
          <Card>
            <TrafficChart samples={chartSamples} />
          </Card>
        </div>
      )}

      {/* Info tab */}
      {tab === "info" && (
        <Card className="space-y-3 font-mono text-sm">
          <div>
            <span className="text-slate-500">Публичный ключ сервера</span>
            <p className="break-all text-slate-200">{t.PublicKey}</p>
          </div>
          <div>
            <span className="text-slate-500">Адрес WG</span>
            <p className="text-slate-200">{t.WgAddress}</p>
          </div>
          <div>
            <span className="text-slate-500">Порт</span>
            <p className="text-slate-200">{t.ListenPort}</p>
          </div>
          <div>
            <span className="text-slate-500">Стратегия балансировки</span>
            <p className="text-slate-200">
              {t.BalancingStrategy === "least_ping" ? "Least Ping" : "Round Robin"}
            </p>
          </div>
          <div>
            <span className="text-slate-500">Узлов</span>
            <p className="text-slate-200">{nodesData?.nodes.length ?? "—"}</p>
          </div>
          {nodesData && nodesData.nodes.length > 0 && (
            <div>
              <span className="text-slate-500">Статус узлов</span>
              <div className="mt-1 flex flex-wrap gap-2">
                {nodesData.nodes.map((n) => (
                  <span key={n.id} className="flex items-center gap-1.5 text-xs">
                    <span className="text-slate-400">{n.display_name || n.address}</span>
                    <NodeHealthBadge health={n.health} />
                  </span>
                ))}
              </div>
            </div>
          )}
        </Card>
      )}

      <Link to="/tunnels" className="text-sm text-indigo-400 hover:underline">
        ← Все туннели
      </Link>
    </div>
  );
}
