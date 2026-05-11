import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Button } from "../../components/ui/Button";
import { Card } from "../../components/ui/Card";
import { Table } from "../../components/ui/Table";
import { ApiException, subscriptionsApi, tunnelsApi, type VlessNode } from "../../services/api";
import { ManualNodeEditModal } from "./ManualNodeEditModal";

type Props = {
  manualSubId: number | null;
  onAdd: () => void;
};

export function ManualNodesSection({ manualSubId, onAdd }: Props) {
  const qc = useQueryClient();
  const [editNode, setEditNode] = useState<VlessNode | null>(null);

  const { data: nodes = [], isLoading } = useQuery({
    queryKey: ["nodes", manualSubId],
    queryFn: ({ signal }) => subscriptionsApi.nodes(manualSubId!, { signal }),
    enabled: manualSubId != null,
  });

  const { data: tunnels = [] } = useQuery({
    queryKey: ["tunnels"],
    queryFn: ({ signal }) => tunnelsApi.list({ signal }),
  });

  const remove = useMutation({
    mutationFn: (id: number) => subscriptionsApi.deleteManualNode(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["nodes", manualSubId] });
      void qc.invalidateQueries({ queryKey: ["subscriptions"] });
      void qc.invalidateQueries({ queryKey: ["tunnels"] });
    },
  });

  const using = (n: VlessNode) =>
    tunnels.find((t) => t.ActiveNodeID === n.ID)?.Name ?? "—";

  function onDeleteClick(n: VlessNode) {
    if (!confirm(`Remove manual node “${n.DisplayName || n.Address}”?`)) return;
    remove.mutate(n.ID, {
      onError: (e) => {
        const msg = e instanceof ApiException ? e.message : (e as Error).message;
        window.alert(msg);
      },
    });
  }

  return (
    <section className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-lg font-semibold text-[#e2e8f0]">Manual VLESS nodes</h2>
          <p className="text-sm text-[#94a3b8]">
            Single <span className="font-mono text-xs text-[#cbd5e1]">vless://</span> links saved for ad-hoc
            tunnels. Not refreshed on a schedule.
          </p>
        </div>
        <Button variant="secondary" onClick={onAdd}>
          Add VLESS link
        </Button>
      </div>

      {manualSubId == null || isLoading ? null : nodes.length === 0 ? (
        <Card className="text-center text-sm text-[#94a3b8]">
          No manual nodes yet. Use “Add VLESS link” to paste a single vless:// URI.
        </Card>
      ) : (
        <Table
          headers={["Display name", "Address", "Port", "Security", "Flow", "Used by", "Actions"]}
        >
          {nodes.map((n) => (
            <tr key={n.ID} className="border-b border-[#2a2a3f]">
              <td className="px-4 py-3">{n.DisplayName || "—"}</td>
              <td className="px-4 py-3 font-mono text-xs">{n.Address}</td>
              <td className="px-4 py-3">{n.Port}</td>
              <td className="px-4 py-3">{n.Security}</td>
              <td className="px-4 py-3">{n.Flow}</td>
              <td className="px-4 py-3">{using(n)}</td>
              <td className="space-x-2 px-4 py-3">
                <Button variant="secondary" className="!px-2 !py-1 text-xs" onClick={() => setEditNode(n)}>
                  Edit
                </Button>
                <Button
                  variant="danger"
                  className="!px-2 !py-1 text-xs"
                  onClick={() => onDeleteClick(n)}
                  disabled={remove.isPending}
                >
                  Delete
                </Button>
              </td>
            </tr>
          ))}
        </Table>
      )}

      {manualSubId != null ? (
        <ManualNodeEditModal
          open={editNode != null}
          node={editNode}
          manualSubId={manualSubId}
          onClose={() => setEditNode(null)}
        />
      ) : null}
    </section>
  );
}
