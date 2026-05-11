import uPlot from "uplot";
import "uplot/dist/uPlot.min.css";
import { useEffect, useRef } from "react";

type Props = {
  rx: number[];
  tx: number[];
  height?: number;
};

export function SparkLine({ rx, tx, height = 40 }: Props) {
  const root = useRef<HTMLDivElement>(null);
  const plot = useRef<uPlot | null>(null);

  useEffect(() => {
    if (!root.current) return;
    const cap = 48;
    const r = rx.slice(-cap);
    const t = tx.slice(-cap);
    const n = Math.max(r.length, t.length, 2);
    const rP = r.length < n ? [...Array(n - r.length).fill(0), ...r] : r.slice(-n);
    const tP = t.length < n ? [...Array(n - t.length).fill(0), ...t] : t.slice(-n);
    const xs = Array.from({ length: n }, (_, i) => i);

    const w = root.current.clientWidth || 120;
    const opts: uPlot.Options = {
      width: w,
      height,
      series: [{}, { stroke: "#22c55e", width: 1 }, { stroke: "#6366f1", width: 1 }],
      axes: [{ show: false }, { show: false }],
      scales: { x: { time: false } },
      legend: { show: false },
      cursor: { show: false },
    };

    const data: uPlot.AlignedData = [xs, rP, tP];

    if (!plot.current) {
      plot.current = new uPlot(opts, data, root.current);
    } else {
      plot.current.setSize({ width: w, height });
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
  }, [rx, tx, height]);

  return <div ref={root} className="w-full min-w-[100px]" style={{ height }} />;
}
