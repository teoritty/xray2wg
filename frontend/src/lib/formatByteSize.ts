const UNITS = ["B", "KB", "MB", "GB", "TB"] as const;

/** Formats byte totals using 1024-based steps (labels KB/MB/GB as common UI convention). */
export function formatByteSize(n: number): string {
  if (!Number.isFinite(n) || n === 0) return "0 B";
  let i = 0;
  let v = Math.abs(n);
  while (v >= 1024 && i < UNITS.length - 1) {
    v /= 1024;
    i++;
  }
  const sign = n < 0 ? "-" : "";
  const rounded = v < 10 && i > 0 ? v.toFixed(1) : String(Math.round(v));
  return `${sign}${rounded} ${UNITS[i]}`;
}

/** Same scaling as {@link formatByteSize}, with `/s` for throughput (e.g. `1.2 MB/s`). */
export function formatByteRate(n: number): string {
  return `${formatByteSize(n)}/s`;
}
