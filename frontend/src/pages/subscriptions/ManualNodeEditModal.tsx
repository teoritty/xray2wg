import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { Button } from "../../components/ui/Button";
import { Label, TextArea } from "../../components/ui/Form";
import { Modal } from "../../components/ui/Modal";
import { ApiException, subscriptionsApi, type VlessNode } from "../../services/api";

type Props = {
  open: boolean;
  node: VlessNode | null;
  manualSubId: number;
  onClose: () => void;
};

export function ManualNodeEditModal({ open, node, manualSubId, onClose }: Props) {
  const qc = useQueryClient();
  const [vlessUri, setVlessUri] = useState("");

  useEffect(() => {
    if (!open || !node) return;
    setVlessUri(node.RawURI);
  }, [open, node]);

  const update = useMutation({
    mutationFn: () => subscriptionsApi.updateManualNode(node!.ID, { vless_uri: vlessUri.trim() }),
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
      {node ? (
        <div className="space-y-3">
          <p className="text-sm text-[#94a3b8]">
            Replace this node with a new <span className="font-mono text-xs text-[#cbd5e1]">vless://</span> link.
            Fields are parsed from the URI; duplicate links are rejected.
          </p>
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
          {errMsg ? <p className="text-sm text-[#ef4444]">{errMsg}</p> : null}
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="secondary" type="button" onClick={onClose}>
              Cancel
            </Button>
            <Button type="button" onClick={() => update.mutate()} disabled={!vlessUri.trim() || update.isPending}>
              {update.isPending ? "Saving…" : "Save"}
            </Button>
          </div>
        </div>
      ) : null}
    </Modal>
  );
}
