import { useEffect, useRef, useState } from "react";

export type TunnelRates = Record<number, { rx: number; tx: number }>;

type WsMessage = {
  type: string;
  tunnels?: { id: number; rx_rate: number; tx_rate: number }[];
  ts?: number;
};

/** Live tunnel rates via WebSocket; uses HttpOnly session cookies (no query token). */
export function useTunnelStatsStream(enabled: boolean): TunnelRates {
  const [rates, setRates] = useState<TunnelRates>({});
  const backoff = useRef(800);

  useEffect(() => {
    if (!enabled) return;

    let ws: WebSocket | null = null;
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout>;

    const connect = () => {
      if (cancelled) return;
      const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
      const url = `${proto}//${window.location.host}/api/v1/ws/stats`;
      ws = new WebSocket(url);

      ws.onopen = () => {
        backoff.current = 800;
      };

      ws.onmessage = (ev) => {
        try {
          const msg = JSON.parse(ev.data as string) as WsMessage;
          if (msg.type !== "stats_update" || !msg.tunnels) return;
          const next: TunnelRates = {};
          for (const t of msg.tunnels) {
            next[t.id] = { rx: t.rx_rate, tx: t.tx_rate };
          }
          setRates(next);
        } catch {
          /* ignore */
        }
      };

      ws.onclose = () => {
        ws = null;
        if (cancelled) return;
        timer = setTimeout(connect, backoff.current);
        backoff.current = Math.min(backoff.current * 1.5, 30_000);
      };

      ws.onerror = () => {
        ws?.close();
      };
    };

    connect();
    return () => {
      cancelled = true;
      clearTimeout(timer);
      ws?.close();
    };
  }, [enabled]);

  return rates;
}
