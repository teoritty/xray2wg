import type { NodeHealthInfo } from "../../services/api";

type Props = {
  health: NodeHealthInfo | null;
};

export function NodeHealthBadge({ health }: Props) {
  if (health === null || health === undefined) {
    return (
      <span className="inline-flex items-center gap-1.5 text-xs text-slate-500">
        <span className="h-2 w-2 rounded-full bg-slate-500" />
        —
      </span>
    );
  }
  if (!health.alive) {
    return (
      <span className="inline-flex items-center gap-1.5 text-xs text-red-400">
        <span className="h-2 w-2 rounded-full bg-red-400" />
        unavailable
      </span>
    );
  }
  const label = health.delay_ms > 0 ? `${health.delay_ms} ms` : "ok";
  return (
    <span className="inline-flex items-center gap-1.5 text-xs text-emerald-400">
      <span className="h-2 w-2 rounded-full bg-emerald-400" />
      {label}
    </span>
  );
}