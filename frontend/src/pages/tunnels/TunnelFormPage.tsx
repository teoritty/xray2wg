import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { NodeSelector, type SelectedNode } from "../../components/tunnels/NodeSelector";
import { Button } from "../../components/ui/Button";
import { Card } from "../../components/ui/Card";
import { Input, Label, Select } from "../../components/ui/Form";
import { MANUAL_SUB_NAME, subscriptionsApi, tunnelsApi, type NodeHealthInfo, type WgInterface } from "../../services/api";

export function TunnelFormPage() {
  const { id } = useParams();
  const editId = id ? Number(id) : NaN;
  const isEdit = Number.isFinite(editId);
  const nav = useNavigate();
  const qc = useQueryClient();

  const { data: subs = [] } = useQuery({
    queryKey: ["subscriptions"],
    queryFn: ({ signal }) => subscriptionsApi.list({ signal }),
  });

  const { data: existing } = useQuery({
    queryKey: ["tunnel", editId],
    queryFn: ({ signal }) => tunnelsApi.get(editId, { signal }),
    enabled: isEdit,
  });

  const { data: existingNodes } = useQuery({
    queryKey: ["tunnel-nodes", editId],
    queryFn: ({ signal }) => tunnelsApi.getNodes(editId, { signal }),
    enabled: isEdit,
  });

  const manual = useMemo(() => subs.find((s) => s.Name === MANUAL_SUB_NAME), [subs]);
  const selectableSubs = useMemo(() => subs.filter((s) => s.Name !== MANUAL_SUB_NAME), [subs]);

  const [name, setName] = useState("");
  const [subId, setSubId] = useState<number | "">("");
  const [selectedNodes, setSelectedNodes] = useState<SelectedNode[]>([]);
  const [strategy, setStrategy] = useState<"round_robin" | "least_ping">("round_robin");
  const [vlessUri, setVlessUri] = useState("");
  const [listenPort, setListenPort] = useState<number | "">("");
  const [wgAddress, setWgAddress] = useState("");
  const [dns, setDns] = useState("");
  const [mtu, setMtu] = useState(1420);
  const [adv, setAdv] = useState(false);

  // Initialise form from existing tunnel data
  useEffect(() => {
    if (!existing) return;
    setName(existing.Name);
    setSubId(existing.SubscriptionID ?? "");
    setListenPort(existing.ListenPort);
    setWgAddress(existing.WgAddress);
    setDns(existing.DNS);
    setMtu(existing.MTU);
  }, [existing]);

  // Initialise nodes and strategy from existing node list
  useEffect(() => {
    if (!existingNodes) return;
    setStrategy(existingNodes.strategy ?? "round_robin");
    setSelectedNodes(
      existingNodes.nodes.map((n) => ({
        id: n.id,
        display_name: n.display_name,
        address: n.address,
        port: n.port,
      }))
    );
  }, [existingNodes]);

  const manualMode = useMemo(() => {
    if (!manual) return false;
    return subId === manual.ID;
  }, [manual, subId]);

  const { data: subNodes = [] } = useQuery({
    queryKey: ["nodes", subId],
    queryFn: ({ signal }) => subscriptionsApi.nodes(Number(subId), { signal }),
    enabled: typeof subId === "number" && !manualMode,
  });

  // All nodes (from all subscriptions) for adding from other subs
  const { data: allNodes = [] } = useQuery({
    queryKey: ["nodes", "all"],
    queryFn: () => Promise.all(subs.filter((s) => s.Name !== MANUAL_SUB_NAME).map((s) => subscriptionsApi.nodes(s.ID)))
      .then((arr) => arr.flat()),
    enabled: subs.length > 0 && !manualMode,
  });

  const availableNodes = subId !== "" && !manualMode ? subNodes : allNodes;

  // Build per-position health from the latest background TCP probe of each VLESS node, looked up
  // by node id from the union of subscription + manual nodes. The probe runs independently of any
  // tunnel, so this works on both the create and edit forms (issue #6).
  const nodesHealth = useMemo(() => {
    if (selectedNodes.length === 0) return undefined;
    const byId = new Map(availableNodes.map((n) => [n.ID, n.Health] as const));
    const health: Record<number, NodeHealthInfo> = {};
    selectedNodes.forEach((sn, i) => {
      health[i] = byId.get(sn.id) ?? null;
    });
    return health;
  }, [selectedNodes, availableNodes]);

  function handleNodeChange(nodeIds: number[], newStrategy: "round_robin" | "least_ping") {
    setStrategy(newStrategy);
    // Rebuild selectedNodes keeping order and metadata
    const nodeMap = new Map(availableNodes.map((n) => [n.ID, n]));
    // Also keep previously selected nodes not in availableNodes
    const existingMap = new Map(selectedNodes.map((n) => [n.id, n]));
    const next: SelectedNode[] = nodeIds.map((nid) => {
      const av = nodeMap.get(nid);
      if (av) return { id: av.ID, display_name: av.DisplayName, address: av.Address, port: av.Port };
      return existingMap.get(nid) ?? { id: nid, display_name: `Node ${nid}`, address: "", port: 0 };
    });
    setSelectedNodes(next);
  }

  const create = useMutation({
    mutationFn: () => {
      if (manualMode) {
        return tunnelsApi.create({
          name,
          listen_port: listenPort === "" ? 0 : Number(listenPort),
          wg_address: wgAddress,
          dns,
          mtu,
          subscription_id: null,
          active_node_id: null,
          vless_uri: vlessUri.trim(),
          balancing_strategy: "round_robin",
        });
      }
      return tunnelsApi.create({
        name,
        listen_port: listenPort === "" ? 0 : Number(listenPort),
        wg_address: wgAddress,
        dns,
        mtu,
        subscription_id: subId === "" ? null : Number(subId),
        active_node_id: selectedNodes[0]?.id ?? null,
        node_ids: selectedNodes.map((n) => n.id),
        balancing_strategy: strategy,
        vless_uri: "",
      });
    },
    onSuccess: (t) => {
      void qc.invalidateQueries({ queryKey: ["tunnels"] });
      nav(`/tunnels/${t.ID}`);
    },
  });

  const update = useMutation({
    mutationFn: async (iface: WgInterface) => {
      await tunnelsApi.update(iface);
      // Update node assignments separately
      if (!manualMode && selectedNodes.length > 0) {
        await tunnelsApi.setNodes(iface.ID, {
          node_ids: selectedNodes.map((n) => n.id),
          strategy,
        });
      }
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["tunnels"] });
      void qc.invalidateQueries({ queryKey: ["tunnel", editId] });
      void qc.invalidateQueries({ queryKey: ["tunnel-nodes", editId] });
      nav(`/tunnels/${editId}`);
    },
  });

  function submit() {
    if (isEdit && existing) {
      const iface: WgInterface = {
        ...existing,
        Name: name,
        ListenPort: Number(listenPort),
        WgAddress: wgAddress,
        DNS: dns,
        MTU: mtu,
        SubscriptionID: subId === "" ? null : Number(subId),
        ActiveNodeID: selectedNodes[0]?.id ?? existing.ActiveNodeID ?? null,
        BalancingStrategy: strategy,
      };
      update.mutate(iface);
      return;
    }
    create.mutate();
  }

  const err = (create.error ?? update.error) as Error | undefined;

  const canSubmit = name.trim() !== "" && (
    manualMode ? vlessUri.trim() !== "" : selectedNodes.length > 0
  );

  return (
    <Card className="max-w-2xl space-y-5">
      <div>
        <Label htmlFor="tun-name">Name</Label>
        <Input id="tun-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="My Tunnel" />
      </div>

      <div>
        <Label>Subscription</Label>
        <Select
          value={subId === "" ? "" : String(subId)}
          onChange={(e) => {
            const v = e.target.value;
            setSubId(v ? Number(v) : "");
            setSelectedNodes([]);
            setVlessUri("");
          }}
        >
          <option value="">Select a subscription…</option>
          {selectableSubs.map((s) => (
            <option key={s.ID} value={String(s.ID)}>
              {s.Name}
            </option>
          ))}
          {manual ? <option value={String(manual.ID)}>Manual VLESS URI</option> : null}
        </Select>
        <p className="mt-1 text-xs text-slate-500">
          Select a subscription to see available VLESS nodes.
        </p>
      </div>

      {manualMode ? (
        <div>
          <Label>VLESS URI</Label>
          <Input value={vlessUri} onChange={(e) => setVlessUri(e.target.value)} placeholder="vless://…" />
          <p className="mt-1 text-xs text-slate-500">Paste the vless:// string from your ISP.</p>
        </div>
      ) : (
        <div>
          <Label>
            VLESS-nodes{" "}
            {selectedNodes.length > 0 && (
              <span className="font-normal text-slate-500">({selectedNodes.length} selected)</span>
            )}
          </Label>
          <p className="mb-2 text-xs text-slate-500">
            Add one or more nodes. With multiple nodes, traffic is balanced between them.
          </p>
          <NodeSelector
            selectedNodes={selectedNodes}
            availableNodes={availableNodes}
            strategy={strategy}
            health={nodesHealth}
            onChange={handleNodeChange}
          />
        </div>
      )}

      <div>
        <Label htmlFor="listen-port">WireGuard Port (UDP)</Label>
        <Input
          id="listen-port"
          type="number"
          value={listenPort}
          onChange={(e) => setListenPort(e.target.value === "" ? "" : Number(e.target.value))}
          placeholder="auto (51820+)"
        />
      </div>

      <div>
        <Label htmlFor="wg-addr">WG Address (CIDR)</Label>
        <Input
          id="wg-addr"
          value={wgAddress}
          onChange={(e) => setWgAddress(e.target.value)}
          placeholder="auto (10.100.N.1/24)"
        />
      </div>

      <button type="button" className="text-sm text-indigo-400 hover:underline" onClick={() => setAdv((v) => !v)}>
        {adv ? "Hide" : "Advanced"}
      </button>
      {adv && (
        <div className="grid gap-3 sm:grid-cols-2">
          <div>
            <Label htmlFor="dns">DNS</Label>
            <Input
              id="dns"
              value={dns}
              onChange={(e) => setDns(e.target.value)}
              placeholder="empty = tunnel gateway (recommended)"
            />
          </div>
          <div>
            <Label htmlFor="mtu">MTU</Label>
            <Input id="mtu" type="number" value={mtu} onChange={(e) => setMtu(Number(e.target.value))} />
          </div>
        </div>
      )}

      {err && <p className="rounded bg-red-950/40 px-3 py-2 text-sm text-red-400">{err.message}</p>}

      <div className="flex gap-2">
        <Button onClick={submit} disabled={!canSubmit || create.isPending || update.isPending}>
          {isEdit ? "Save" : "Create"}
        </Button>
        <Link to={isEdit ? `/tunnels/${editId}` : "/tunnels"}>
          <Button variant="secondary">Cancel</Button>
        </Link>
      </div>
    </Card>
  );
}
