import { formatByteRate, formatByteSize } from "./formatByteSize";

export function formatBytes(n: number): string {
  return formatByteSize(n);
}

export function formatBytesRate(n: number): string {
  return formatByteRate(n);
}

export function formatTime(ts?: string | null): string {
  if (!ts) return "—";
  const d = new Date(ts);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleString();
}
