import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Link } from "react-router-dom";
import { Button } from "../../components/ui/Button";
import { Card } from "../../components/ui/Card";
import { Input, Label, Select } from "../../components/ui/Form";
import { Modal } from "../../components/ui/Modal";
import { StatusBadge } from "../../components/ui/Badge";
import { Table } from "../../components/ui/Table";
import { formatTime } from "../../lib/format";
import { MANUAL_SUB_NAME, type Subscription, subscriptionsApi } from "../../services/api";
import { ManualNodesSection } from "./ManualNodesSection";
import { SubscriptionAddModal, type AddMode } from "./SubscriptionAddModal";

const intervals = [
  { sec: 3600, label: "1 hour" },
  { sec: 21600, label: "6 hours" },
  { sec: 86400, label: "24 hours" },
];

export function SubscriptionListPage() {
  const qc = useQueryClient();
  const { data = [], isLoading } = useQuery({
    queryKey: ["subscriptions"],
    queryFn: ({ signal }) => subscriptionsApi.list({ signal }),
  });
  const [addOpen, setAddOpen] = useState(false);
  const [addMode, setAddMode] = useState<AddMode>("subscription_url");
  const [edit, setEdit] = useState<Subscription | null>(null);

  const update = useMutation({
    mutationFn: (s: Subscription) => subscriptionsApi.update(s),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["subscriptions"] });
      setEdit(null);
    },
  });

  const remove = useMutation({
    mutationFn: (id: number) => subscriptionsApi.remove(id),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["subscriptions"] }),
  });

  const refresh = useMutation({
    mutationFn: (id: number) => subscriptionsApi.refresh(id),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ["subscriptions"] }),
  });

  function openAdd(mode: AddMode) {
    setAddMode(mode);
    setAddOpen(true);
  }

  if (isLoading) return <p className="text-[#94a3b8]">Loading…</p>;

  const userSubs = data.filter((s) => s.Name !== MANUAL_SUB_NAME);
  const manualSub = data.find((s) => s.Name === MANUAL_SUB_NAME) ?? null;

  return (
    <div className="space-y-10">
      <section className="space-y-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 className="text-lg font-semibold text-[#e2e8f0]">Subscriptions</h2>
            <p className="text-sm text-[#94a3b8]">
              Provider URLs that fetch VLESS nodes on a schedule.
            </p>
          </div>
          <Button variant="secondary" onClick={() => openAdd("subscription_url")}>
            Add subscription
          </Button>
        </div>

        {userSubs.length === 0 ? (
          <Card className="text-center text-sm text-[#94a3b8]">
            No subscriptions yet. Use “Add subscription” to register a provider URL.
          </Card>
        ) : (
          <Table headers={["Name", "URL", "Nodes", "Last fetched", "Status", "Actions"]}>
            {userSubs.map((s) => (
              <tr key={s.ID} className="border-b border-[#2a2a3f]">
                <td className="px-4 py-3">
                  <Link className="text-[#6366f1] hover:underline" to={`/subscriptions/${s.ID}`}>
                    {s.Name}
                  </Link>
                </td>
                <td className="max-w-[200px] truncate px-4 py-3 font-mono text-xs text-[#94a3b8]" title={s.URL}>
                  {s.URL}
                </td>
                <td className="px-4 py-3">{s.NodeCount}</td>
                <td className="px-4 py-3 text-[#94a3b8]">{formatTime(s.LastFetchedAt)}</td>
                <td className="px-4 py-3">
                  <StatusBadge status={s.Status} />
                </td>
                <td className="space-x-2 px-4 py-3">
                  <Button variant="ghost" className="!px-2 !py-1 text-xs" onClick={() => refresh.mutate(s.ID)}>
                    Refresh
                  </Button>
                  <Button variant="secondary" className="!px-2 !py-1 text-xs" onClick={() => setEdit(s)}>
                    Edit
                  </Button>
                  <Button
                    variant="danger"
                    className="!px-2 !py-1 text-xs"
                    onClick={() => {
                      if (confirm(`Delete subscription ${s.Name}?`)) remove.mutate(s.ID);
                    }}
                  >
                    Delete
                  </Button>
                </td>
              </tr>
            ))}
          </Table>
        )}
      </section>

      <ManualNodesSection
        manualSubId={manualSub?.ID ?? null}
        onAdd={() => openAdd("vless_uri")}
      />

      <SubscriptionAddModal
        open={addOpen}
        initialMode={addMode}
        onClose={() => setAddOpen(false)}
      />

      <Modal
        open={!!edit}
        title="Edit subscription"
        onClose={() => setEdit(null)}
      >
        {edit ? (
          <div className="space-y-3">
            <div>
              <Label>Name</Label>
              <Input
                value={edit.Name}
                onChange={(e) => setEdit({ ...edit, Name: e.target.value })}
              />
            </div>
            <div>
              <Label>URL</Label>
              <Input
                value={edit.URL}
                onChange={(e) => setEdit({ ...edit, URL: e.target.value })}
              />
            </div>
            <div>
              <Label>Refresh interval</Label>
              <Select
                value={String(edit.RefreshInterval)}
                onChange={(e) =>
                  setEdit({ ...edit, RefreshInterval: Number(e.target.value) })
                }
              >
                {intervals.map((x) => (
                  <option key={x.sec} value={String(x.sec)}>
                    {x.label}
                  </option>
                ))}
              </Select>
            </div>
            {update.error && (
              <p className="text-sm text-[#ef4444]">{(update.error as Error).message}</p>
            )}
            <div className="flex justify-end gap-2 pt-2">
              <Button variant="secondary" onClick={() => setEdit(null)}>
                Cancel
              </Button>
              <Button onClick={() => update.mutate(edit)}>Save</Button>
            </div>
          </div>
        ) : null}
      </Modal>
    </div>
  );
}
