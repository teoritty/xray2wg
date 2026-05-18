import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { ManualNodeForm, emptyManualNodeBody } from "../../components/nodes/ManualNodeForm";
import { Button } from "../../components/ui/Button";
import { Input, Label, Select, TextArea } from "../../components/ui/Form";
import { Modal } from "../../components/ui/Modal";
import { ApiException, MANUAL_SUB_NAME, subscriptionsApi } from "../../services/api";
import type { ManualNodeBody } from "../../types/vless";

export type AddMode = "subscription_url" | "vless_uri";

type VlessSubMode = "uri" | "form";

const intervals = [
  { sec: 3600, label: "1 hour" },
  { sec: 21600, label: "6 hours" },
  { sec: 86400, label: "24 hours" },
];

type Props = {
  open: boolean;
  initialMode?: AddMode;
  onClose: () => void;
};

export function SubscriptionAddModal({ open, initialMode = "subscription_url", onClose }: Props) {
  const qc = useQueryClient();
  const [mode, setMode] = useState<AddMode>(initialMode);
  const [vlessSubMode, setVlessSubMode] = useState<VlessSubMode>("uri");
  const [name, setName] = useState("");
  const [url, setUrl] = useState("");
  const [ival, setIval] = useState(3600);
  const [vlessUri, setVlessUri] = useState("");
  const [manualForm, setManualForm] = useState<ManualNodeBody>(emptyManualNodeBody());
  const [savedOnce, setSavedOnce] = useState(false);

  useEffect(() => {
    if (!open) return;
    setMode(initialMode);
    setVlessSubMode("uri");
    setName("");
    setUrl("");
    setIval(3600);
    setVlessUri("");
    setManualForm(emptyManualNodeBody());
    setSavedOnce(false);
  }, [open, initialMode]);

  const create = useMutation({
    mutationFn: () =>
      subscriptionsApi.create({ name, url, refresh_interval: ival }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["subscriptions"] });
      onClose();
    },
  });

  const addManual = useMutation({
    mutationFn: () =>
      vlessSubMode === "uri"
        ? subscriptionsApi.addManualNode({ vless_uri: vlessUri.trim() })
        : subscriptionsApi.addManualNode({ manual: manualForm }),
    onSuccess: (node) => {
      void qc.invalidateQueries({ queryKey: ["subscriptions"] });
      void qc.invalidateQueries({ queryKey: ["nodes", node.SubscriptionID] });
      setVlessUri("");
      setManualForm(emptyManualNodeBody());
      setSavedOnce(true);
    },
  });

  const feedError =
    create.error instanceof ApiException ? create.error.message : (create.error as Error)?.message;
  const vlessError =
    addManual.error instanceof ApiException ? addManual.error.message : (addManual.error as Error)?.message;

  return (
    <Modal open={open} title="Add subscription or VLESS link" onClose={onClose}>
      <div className="space-y-4">
        <p className="text-sm text-[#94a3b8]">
          Choose how you want to add outbound nodes: a provider subscription URL (fetched on a schedule), or a
          single <span className="font-mono text-xs text-[#cbd5e1]">vless://</span> share link saved as a
          manual node for tunnels.
        </p>

        <div className="flex flex-wrap gap-2" role="tablist" aria-label="Add method">
          <Button
            type="button"
            variant={mode === "subscription_url" ? "primary" : "secondary"}
            className="!px-3 !py-1.5 text-sm"
            onClick={() => setMode("subscription_url")}
          >
            Subscription URL
          </Button>
          <Button
            type="button"
            variant={mode === "vless_uri" ? "primary" : "secondary"}
            className="!px-3 !py-1.5 text-sm"
            onClick={() => setMode("vless_uri")}
          >
            Single VLESS link
          </Button>
        </div>

        {mode === "subscription_url" ? (
          <div className="space-y-3">
            <div>
              <Label htmlFor="sub-name">Name</Label>
              <Input
                id="sub-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                autoComplete="off"
                placeholder={`Anything except "${MANUAL_SUB_NAME}"`}
              />
            </div>
            <div>
              <Label htmlFor="sub-url">Subscription URL</Label>
              <Input
                id="sub-url"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                placeholder="https://…"
                autoComplete="off"
              />
            </div>
            <div>
              <Label htmlFor="sub-interval">Refresh interval</Label>
              <Select id="sub-interval" value={String(ival)} onChange={(e) => setIval(Number(e.target.value))}>
                {intervals.map((x) => (
                  <option key={x.sec} value={String(x.sec)}>
                    {x.label}
                  </option>
                ))}
              </Select>
            </div>
            {feedError ? <p className="text-sm text-[#ef4444]">{feedError}</p> : null}
            <div className="flex justify-end gap-2 pt-2">
              <Button variant="secondary" type="button" onClick={onClose}>
                Cancel
              </Button>
              <Button
                type="button"
                onClick={() => create.mutate()}
                disabled={!name.trim() || !url.trim() || create.isPending}
              >
                {create.isPending ? "Saving…" : "Add subscription"}
              </Button>
            </div>
          </div>
        ) : (
          <div className="space-y-3">
            <div className="flex gap-2" role="tablist">
              <Button
                type="button"
                variant={vlessSubMode === "uri" ? "primary" : "secondary"}
                className="!px-3 !py-1.5 text-xs"
                onClick={() => setVlessSubMode("uri")}
              >
                Paste URI
              </Button>
              <Button
                type="button"
                variant={vlessSubMode === "form" ? "primary" : "secondary"}
                className="!px-3 !py-1.5 text-xs"
                onClick={() => setVlessSubMode("form")}
              >
                Manual edit
              </Button>
            </div>

            {vlessSubMode === "uri" ? (
              <div>
                <Label htmlFor="vless-paste">VLESS URI</Label>
                <TextArea
                  id="vless-paste"
                  rows={4}
                  value={vlessUri}
                  onChange={(e) => setVlessUri(e.target.value)}
                  placeholder="vless://uuid@host:443?…#label"
                  spellCheck={false}
                  autoComplete="off"
                />
                <p className="mt-2 text-xs text-[#64748b]">
                  Paste one complete link. It is stored as a manual node (same pool used when creating a tunnel
                  with “Manual VLESS URI”). Duplicate links are rejected.
                </p>
              </div>
            ) : (
              <ManualNodeForm value={manualForm} onChange={setManualForm} />
            )}

            {vlessError ? <p className="text-sm text-[#ef4444]">{vlessError}</p> : null}
            {savedOnce && !addManual.isPending && !vlessError ? (
              <p className="text-sm text-[#22c55e]">
                Saved. The new node now appears in the “Manual VLESS nodes” section.
              </p>
            ) : null}
            <div className="flex justify-end gap-2 pt-2">
              <Button variant="secondary" type="button" onClick={onClose}>
                {savedOnce ? "Close" : "Cancel"}
              </Button>
              <Button
                type="button"
                onClick={() => addManual.mutate()}
                disabled={
                  addManual.isPending ||
                  (vlessSubMode === "uri"
                    ? !vlessUri.trim()
                    : !manualForm.uuid.trim() || !manualForm.address.trim() || manualForm.port <= 0)
                }
              >
                {addManual.isPending ? "Saving…" : "Save VLESS node"}
              </Button>
            </div>
          </div>
        )}
      </div>
    </Modal>
  );
}
