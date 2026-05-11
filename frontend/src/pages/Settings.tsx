import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Input, Label } from "../components/ui/Form";
import { settingsApi } from "../services/api";

export function SettingsPage() {
  const qc = useQueryClient();
  const fileRef = useRef<HTMLInputElement>(null);

  const { data: s } = useQuery({
    queryKey: ["settings"],
    queryFn: ({ signal }) => settingsApi.get({ signal }),
  });

  const [host, setHost] = useState("");
  const [oldPw, setOldPw] = useState("");
  const [newPw, setNewPw] = useState("");
  const [confirmPw, setConfirmPw] = useState("");

  useEffect(() => {
    setHost(s?.server_host ?? "");
  }, [s?.server_host]);

  const saveHost = useMutation({
    mutationFn: () => settingsApi.put(host.trim()),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["settings"] }),
  });

  const savePw = useMutation({
    mutationFn: () => settingsApi.password(oldPw, newPw),
    onSuccess: () => {
      setOldPw("");
      setNewPw("");
      setConfirmPw("");
    },
  });

  const importMut = useMutation({
    mutationFn: (txt: string) => settingsApi.importData(txt),
    onSuccess: () => void qc.invalidateQueries(),
  });

  return (
    <div className="mx-auto max-w-2xl space-y-8">
      <Card className="space-y-4">
        <h2 className="text-lg font-semibold text-[#e2e8f0]">Authentication</h2>
        <div>
          <Label>Current password</Label>
          <Input type="password" value={oldPw} autoComplete="current-password" onChange={(e) => setOldPw(e.target.value)} />
        </div>
        <div>
          <Label>New password (min 8)</Label>
          <Input type="password" value={newPw} autoComplete="new-password" onChange={(e) => setNewPw(e.target.value)} />
        </div>
        <div>
          <Label>Confirm new password</Label>
          <Input type="password" value={confirmPw} autoComplete="new-password" onChange={(e) => setConfirmPw(e.target.value)} />
        </div>
        {(savePw.error as Error | undefined) && (
          <p className="text-sm text-[#ef4444]">{(savePw.error as Error).message}</p>
        )}
        <Button
          onClick={() => {
            if (newPw !== confirmPw) return;
            savePw.mutate();
          }}
          disabled={!oldPw || newPw.length < 8 || newPw !== confirmPw || savePw.isPending}
        >
          Change password
        </Button>
      </Card>

      <Card className="space-y-4">
        <h2 className="text-lg font-semibold text-[#e2e8f0]">Server</h2>
        <p className="text-sm text-[#94a3b8]">
          Hostname or IP substituted into MikroTik export scripts as the WG endpoint address.
        </p>
        <div>
          <Label>Server host</Label>
          <Input value={host} onChange={(e) => setHost(e.target.value)} placeholder="203.0.113.10" />
        </div>
        {(saveHost.error as Error | undefined) && (
          <p className="text-sm text-[#ef4444]">{(saveHost.error as Error).message}</p>
        )}
        <Button onClick={() => saveHost.mutate()} disabled={saveHost.isPending}>
          Save
        </Button>
      </Card>

      <Card className="space-y-4">
        <h2 className="text-lg font-semibold text-[#e2e8f0]">Danger zone</h2>
        <p className="text-sm text-[#94a3b8]">
          Export or import minimal configuration (subscriptions). Review JSON before importing on a live system.
        </p>
        <div className="flex flex-wrap gap-2">
          <Button variant="secondary" onClick={() => void settingsApi.exportDownload()}>
            Export
          </Button>
          <Button
            variant="secondary"
            onClick={() => fileRef.current?.click()}
          >
            Import
          </Button>
          <input
            ref={fileRef}
            type="file"
            accept="application/json,.json"
            className="hidden"
            onChange={(e) => {
              const f = e.target.files?.[0];
              if (!f) return;
              const reader = new FileReader();
              reader.onload = () => {
                const txt = String(reader.result ?? "");
                importMut.mutate(txt);
                e.target.value = "";
              };
              reader.readAsText(f);
            }}
          />
        </div>
        {(importMut.error as Error | undefined) && (
          <p className="text-sm text-[#ef4444]">{(importMut.error as Error).message}</p>
        )}
      </Card>
    </div>
  );
}
