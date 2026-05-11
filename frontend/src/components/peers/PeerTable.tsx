import { useState } from "react";
import { Link } from "react-router-dom";
import { formatBytes, formatTime } from "../../lib/format";
import { peersApi, type WgPeer } from "../../services/api";
import { Button } from "../ui/Button";
import { Table } from "../ui/Table";

export type PeerTableRowModel = {
  peer: WgPeer;
  tunnelId: number;
  tunnelName?: string;
};

type PeerTableProps = {
  rows: PeerTableRowModel[];
  showTunnelColumn: boolean;
  onAfterDelete: (tunnelId: number) => void;
};

const actionBtn = "!px-2 !py-1 text-xs";

export function PeerTable({ rows, showTunnelColumn, onAfterDelete }: PeerTableProps) {
  const [deletingKey, setDeletingKey] = useState<string | null>(null);

  const headers = showTunnelColumn
    ? ["Tunnel", "Name", "Client IP", "Last handshake", "RX", "TX", "Actions"]
    : ["Name", "Client IP", "Last handshake", "RX", "TX", "Actions"];

  return (
    <Table headers={headers}>
      {rows.map(({ peer: p, tunnelId, tunnelName }) => {
        const rowKey = `${tunnelId}-${p.ID}`;
        return (
        <tr key={rowKey} className="border-b border-[#2a2a3f]">
          {showTunnelColumn ? (
            <td className="px-4 py-3">
              <Link className="text-[#6366f1] hover:underline" to={`/tunnels/${tunnelId}`}>
                {tunnelName ?? `Tunnel #${tunnelId}`}
              </Link>
            </td>
          ) : null}
          <td className="px-4 py-3">{p.Name}</td>
          <td className="px-4 py-3 font-mono text-xs">{p.ClientAddress}</td>
          <td className="px-4 py-3 text-[#94a3b8]">{formatTime(p.LastHandshake)}</td>
          <td className="px-4 py-3 font-mono text-xs">{formatBytes(p.RxBytes)}</td>
          <td className="px-4 py-3 font-mono text-xs">{formatBytes(p.TxBytes)}</td>
          <td className="px-4 py-3">
            <div className="flex flex-wrap items-center gap-2">
              <Link to={`/tunnels/${tunnelId}/peers/${p.ID}/config`}>
                <Button variant="secondary" className={actionBtn}>
                  Config
                </Button>
              </Link>
              <Button
                variant="danger"
                className={actionBtn}
                disabled={deletingKey === rowKey}
                onClick={() => {
                  if (!confirm(`Remove peer ${p.Name}?`)) return;
                  setDeletingKey(rowKey);
                  void peersApi
                    .remove(tunnelId, p.ID)
                    .then(() => onAfterDelete(tunnelId))
                    .finally(() => setDeletingKey(null));
                }}
              >
                Delete
              </Button>
            </div>
          </td>
        </tr>
        );
      })}
    </Table>
  );
}
