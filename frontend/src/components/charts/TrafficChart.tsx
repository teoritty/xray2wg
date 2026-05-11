import uPlot from "uplot";
import "uplot/dist/uPlot.min.css";
import { useEffect, useRef } from "react";
import { formatByteRate } from "../../lib/formatByteSize";

type Props = {
  samples: { t: number; rx: number; tx: number }[];
  height?: number;
};

/** uPlot default Y axis width; too narrow for {@link formatByteRate} tick labels. */
const UplotDefaultYAxisSize = 50;

/**
 * Reserves enough horizontal space for the current tick labels. uPlot's default
 * is a fixed 50px, which clips strings like `1024 MB/s`.
 */
function byteRateAxisSize(
  u: uPlot,
  values: (string | number)[] | null,
  axisIdx: number,
): number {
  const axis = u.axes[axisIdx];
  const tickShown = axis.ticks?.show !== false;
  const tickSize = tickShown ? (axis.ticks?.size ?? 10) : 0;
  const gap = axis.gap ?? 5;
  const pad = 6;

  if (!values?.length) {
    return UplotDefaultYAxisSize;
  }

  const font = Array.isArray(axis.font) ? axis.font[0] : axis.font;
  u.ctx.save();
  u.ctx.font = font;
  let maxDevicePx = 0;
  for (const v of values) {
    if (v == null || v === "") continue;
    maxDevicePx = Math.max(maxDevicePx, u.ctx.measureText(String(v)).width);
  }
  u.ctx.restore();

  const maxCssPx = maxDevicePx / uPlot.pxRatio;
  return Math.max(
    UplotDefaultYAxisSize,
    Math.ceil(maxCssPx + tickSize + gap + pad),
  );
}

export function TrafficChart({ samples, height = 220 }: Props) {
  const root = useRef<HTMLDivElement>(null);
  const plot = useRef<uPlot | null>(null);

  useEffect(() => {
    if (!root.current) return;

    const src =
      samples.length >= 2
        ? samples
        : [
            { t: 0, rx: 0, tx: 0 },
            { t: 1, rx: 0, tx: 0 },
          ];
    const xs = src.map((s) => s.t);
    const rx = src.map((s) => s.rx);
    const tx = src.map((s) => s.tx);

    const opts: uPlot.Options = {
      width: root.current.clientWidth || 400,
      height,
      series: [
        {},
        {
          label: "RX",
          stroke: "#22c55e",
          fill: "rgba(34,197,94,0.12)",
          scale: "bps",
          value: (_u, raw) => formatByteRate(raw),
        },
        {
          label: "TX",
          stroke: "#6366f1",
          fill: "rgba(99,102,241,0.12)",
          scale: "bps",
          value: (_u, raw) => formatByteRate(raw),
        },
      ],
      axes: [
        { stroke: "#64748b", grid: { stroke: "#2a2a3f" } },
        {
          scale: "bps",
          stroke: "#64748b",
          grid: { stroke: "#2a2a3f" },
          size: byteRateAxisSize,
          values: (_self, splits) => splits.map((v) => (v == null ? "" : formatByteRate(Number(v)))),
        },
      ],
      scales: {
        // X values are Unix epoch seconds (see Statistics / TunnelDetail samples).
        x: { time: true },
        bps: {},
      },
      legend: { show: true },
    };

    const data: uPlot.AlignedData = [xs, rx, tx];

    if (!plot.current) {
      plot.current = new uPlot(opts, data, root.current);
    } else {
      plot.current.setSize({ width: root.current.clientWidth, height });
      plot.current.setData(data);
    }

    const ro = new ResizeObserver(() => {
      if (!root.current || !plot.current) return;
      plot.current.setSize({ width: root.current.clientWidth, height });
    });
    ro.observe(root.current);

    return () => {
      ro.disconnect();
      plot.current?.destroy();
      plot.current = null;
    };
  }, [samples, height]);

  return <div ref={root} className="w-full min-h-[220px]" />;
}
