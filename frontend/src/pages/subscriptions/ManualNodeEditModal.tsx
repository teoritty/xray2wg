import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { ManualNodeForm } from "../../components/nodes/ManualNodeForm";
import { Button } from "../../components/ui/Button";
import { Label, TextArea } from "../../components/ui/Form";
import { Modal } from "../../components/ui/Modal";
import { ApiException, subscriptionsApi, type VlessNode } from "../../services/api";
import { defaultSecurity, defaultTransport, type ManualNodeBody, type SecuritySpec, type TransportSpec } from "../../types/vless";

type Props = {
  open: boolean;
  node: VlessNode | null;
  manualSubId: number;
  onClose: () => void;
};

type Tab = "uri" | "form";

// nodeToManualBody projects an existing VlessNode into the structured form's value. The
// Transport / SecurityCfg fields from the API DTO carry the decoded specs; when they are
// missing (server failed to decode) we fall back to the transport / security defaults so
// the form is editable rather than empty.
function nodeToManualBody(n: VlessNode): ManualNodeBody {
  const tType = (n.Network as TransportSpec["type"]) || "tcp";
  const sName = (n.Security as SecuritySpec["name"]) || "none";
  const tSpec = ((): TransportSpec => {
    const decoded = (n as unknown as { Transport?: unknown }).Transport;
    if (decoded && typeof decoded === "object") {
      return { type: tType, ...(decoded as object) } as TransportSpec;
    }
    return defaultTransport(tType);
  })();
  const sSpec = ((): SecuritySpec => {
    const decoded = (n as unknown as { SecurityCfg?: unknown }).SecurityCfg;
    if (decoded && typeof decoded === "object") {
      return { name: sName, ...(decoded as object) } as SecuritySpec;
    }
    return defaultSecurity(sName);
  })();
  return {
    display_name: n.DisplayName,
    uuid: n.UUID,
    address: n.Address,
    port: n.Port,
    flow: n.Flow,
    encryption: n.Encryption || "none",
    packet_encoding: n.PacketEncoding,
    network: tType,
    transport: tSpec,
    security: sName,
    security_cfg: sSpec,
  };
}

export function ManualNodeEditModal({ open, node, manualSubId, onClose }: Props) {
  const qc = useQueryClient();
  const [tab, setTab] = useState<Tab>("uri");
  const [vlessUri, setVlessUri] = useState("");
  const [form, setForm] = useState<ManualNodeBody | null>(null);

  useEffect(() => {
    if (!open || !node) return;
    setTab("uri");
    setVlessUri(node.RawURI);
    setForm(nodeToManualBody(node));
  }, [open, node]);

  const update = useMutation({
    mutationFn: () =>
      tab === "uri"
        ? subscriptionsApi.updateManualNode(node!.ID, { vless_uri: vlessUri.trim() })
        : subscriptionsApi.updateManualNode(node!.ID, { manual: form! }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["nodes", manualSubId] });
      void qc.invalidateQueries({ queryKey: ["tunnels"] });
      onClose();
    },
  });

  const errMsg =
    update.error instanceof ApiException ? update.error.message : (update.error as Error)?.message;

  return (
    <Modal open={open} title="Edit manual VLESS node" onClose={onClose}>
      {node && form ? (
        <div className="space-y-3">
          <div className="flex gap-2" role="tablist">
            <Button
              variant={tab === "uri" ? "primary" : "secondary"}
              className="!px-3 !py-1.5 text-sm"
              onClick={() => setTab("uri")}
            >
              Paste URI
            </Button>
            <Button
              variant={tab === "form" ? "primary" : "secondary"}
              className="!px-3 !py-1.5 text-sm"
              onClick={() => setTab("form")}
            >
              Manual edit
            </Button>
          </div>

          {tab === "uri" ? (
            <div>
              <Label htmlFor="manual-edit-vless">VLESS URI</Label>
              <TextArea
                id="manual-edit-vless"
                rows={4}
                value={vlessUri}
                onChange={(e) => setVlessUri(e.target.value)}
                spellCheck={false}
                autoComplete="off"
              />
            </div>
          ) : (
            <ManualNodeForm value={form} onChange={setForm} />
          )}

          {errMsg ? <p className="text-sm text-[#ef4444]">{errMsg}</p> : null}
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="secondary" type="button" onClick={onClose}>
              Cancel
            </Button>
            <Button
              type="button"
              onClick={() => update.mutate()}
              disabled={
                update.isPending ||
                (tab === "uri" ? !vlessUri.trim() : !form.uuid.trim() || !form.address.trim() || form.port <= 0)
              }
            >
              {update.isPending ? "Saving…" : "Save"}
            </Button>
          </div>
        </div>
      ) : null}
    </Modal>
  );
}
