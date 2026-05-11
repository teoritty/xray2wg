import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, Navigate, useParams } from "react-router-dom";
import { Button } from "../../components/ui/Button";
import { Card } from "../../components/ui/Card";
import { StatusBadge } from "../../components/ui/Badge";
import { Table } from "../../components/ui/Table";
import { formatTime } from "../../lib/format";
import {
  MANUAL_SUB_NAME,
  subscriptionsApi,
  tunnelsApi,
  type VlessNode,
} from "../../services/api";

export function SubscriptionDetailPage() {
  const { id } = useParams();
  const sid = Number(id);
  const qc = useQueryClient();

  const { data: sub, isLoading } = useQuery({
    queryKey: ["subscription", sid],
    queryFn: ({ signal }) => subscriptionsApi.get(sid, { signal }),
    enabled: Number.isFinite(sid),
  });

  const { data: nodes = [] } = useQuery({
    queryKey: ["nodes", sid],
    queryFn: ({ signal }) => subscriptionsApi.nodes(sid, { signal }),
    enabled: Number.isFinite(sid) && sub?.Name !== MANUAL_SUB_NAME,
  });

  const { data: tunnels = [] } = useQuery({
    queryKey: ["tunnels"],
    queryFn: ({ signal }) => tunnelsApi.list({ signal }),
  });

  const refresh = useMutation({
    mutationFn: () => subscriptionsApi.refresh(sid),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["subscription", sid] });
      void qc.invalidateQueries({ queryKey: ["nodes", sid] });
    },
  });

  if (!Number.isFinite(sid)) return <p className="text-[#ef4444]">Invalid id</p>;
  if (isLoading || !sub) return <p className="text-[#94a3b8]">Loading…</p>;

  // Manual nodes are presented inline on the subscriptions list page; never as a
  // standalone subscription detail. Redirect to keep the internal `__manual__`
  // record out of the user-visible navigation.
  if (sub.Name === MANUAL_SUB_NAME) {
    return <Navigate to="/subscriptions" replace />;
  }

  const using = (n: VlessNode) =>
    tunnels.find((t) => t.ActiveNodeID === n.ID)?.Name ?? "—";

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h2 className="text-xl font-semibold text-[#e2e8f0]">{sub.Name}</h2>
          <p className="mt-1 text-sm text-[#94a3b8]">
            <StatusBadge status={sub.Status} /> · Last fetch: {formatTime(sub.LastFetchedAt)}
          </p>
          <p className="mt-2 max-w-2xl truncate font-mono text-xs text-[#64748b]" title={sub.URL}>
            {sub.URL}
          </p>
        </div>
        <Button onClick={() => refresh.mutate()} disabled={refresh.isPending}>
          Refresh now
        </Button>
      </div>

      {sub.ErrorMessage ? (
        <Card className="border-[#ef4444]/40">
          <p className="text-sm text-[#ef4444]">{sub.ErrorMessage}</p>
        </Card>
      ) : null}

      <Table
        headers={["Display name", "Address", "Port", "Security", "Flow", "Used by"]}
      >
        {nodes.map((n) => (
          <tr key={n.ID} className="border-b border-[#2a2a3f]">
            <td className="px-4 py-3">{n.DisplayName || "—"}</td>
            <td className="px-4 py-3 font-mono text-xs">{n.Address}</td>
            <td className="px-4 py-3">{n.Port}</td>
            <td className="px-4 py-3">{n.Security}</td>
            <td className="px-4 py-3">{n.Flow}</td>
            <td className="px-4 py-3">{using(n)}</td>
          </tr>
        ))}
      </Table>
      {refresh.error && (
        <p className="text-sm text-[#ef4444]">{(refresh.error as Error).message}</p>
      )}

      <Link to="/subscriptions" className="text-sm text-[#6366f1] hover:underline">
        Back to list
      </Link>
    </div>
  );
}
