import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { Button } from "../../components/ui/Button";
import { Card } from "../../components/ui/Card";
import { Input, Label } from "../../components/ui/Form";
import { peersApi } from "../../services/api";

export function PeerCreatePage() {
  const { tid } = useParams();
  const tunnelId = Number(tid);
  const nav = useNavigate();
  const qc = useQueryClient();
  const [name, setName] = useState("");
  const [pub, setPub] = useState("");
  const [clientIp, setClientIp] = useState("");

  const create = useMutation({
    mutationFn: () =>
      peersApi.create(tunnelId, {
        name,
        public_key: pub.trim(),
        client_address: clientIp.trim(),
      }),
    onSuccess: (p) => {
      void qc.invalidateQueries({ queryKey: ["peers", tunnelId] });
      void qc.invalidateQueries({ queryKey: ["peers-global"] });
      nav(`/tunnels/${tunnelId}/peers/${p.ID}/config`);
    },
  });

  if (!Number.isFinite(tunnelId)) return <p className="text-[#ef4444]">Invalid tunnel</p>;

  return (
    <Card className="max-w-lg space-y-4">
      <div>
        <Label>Name</Label>
        <Input value={name} onChange={(e) => setName(e.target.value)} />
      </div>
      <div>
        <Label>Public key (optional — generated if empty)</Label>
        <Input value={pub} onChange={(e) => setPub(e.target.value)} className="font-mono text-xs" />
      </div>
      <div>
        <Label>Client address (optional — auto if empty)</Label>
        <Input value={clientIp} onChange={(e) => setClientIp(e.target.value)} placeholder="10.100.x.y/32" />
      </div>
      {(create.error as Error | undefined) && (
        <p className="text-sm text-[#ef4444]">{(create.error as Error).message}</p>
      )}
      <div className="flex gap-2">
        <Button onClick={() => create.mutate()} disabled={!name.trim()}>
          Create
        </Button>
        <Link to={`/tunnels/${tunnelId}`}>
          <Button variant="secondary">Cancel</Button>
        </Link>
      </div>
    </Card>
  );
}
