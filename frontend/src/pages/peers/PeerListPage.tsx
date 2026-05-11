import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { PeerTable } from "../../components/peers/PeerTable";
import { Button } from "../../components/ui/Button";
import { peersApi } from "../../services/api";

export function PeerListPage() {
  const qc = useQueryClient();
  const { data: items = [], isLoading } = useQuery({
    queryKey: ["peers-global"],
    queryFn: ({ signal }) => peersApi.listAll({ signal }),
  });

  if (isLoading) return <p className="text-[#94a3b8]">Loading…</p>;

  const rows = items.map((p) => ({
    peer: p,
    tunnelId: p.InterfaceID,
    tunnelName: p.TunnelName,
  }));

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold text-[#e2e8f0]">Peers</h2>
        <p className="mt-2 max-w-2xl text-sm text-[#94a3b8]">
          All WireGuard peers across tunnels. Open a tunnel to add a peer or use{" "}
          <span className="text-[#e2e8f0]">Add peer</span> from that tunnel&apos;s detail page.
        </p>
      </div>
      <div className="flex flex-wrap gap-2">
        <Link to="/tunnels">
          <Button variant="secondary">Tunnels</Button>
        </Link>
      </div>
      {rows.length === 0 ? (
        <p className="text-sm text-[#64748b]">No peers yet. Create a tunnel and add peers from its detail view.</p>
      ) : (
        <PeerTable
          showTunnelColumn
          rows={rows}
          onAfterDelete={(tunnelId) => {
            void qc.invalidateQueries({ queryKey: ["peers-global"] });
            void qc.invalidateQueries({ queryKey: ["peers", tunnelId] });
          }}
        />
      )}
    </div>
  );
}
