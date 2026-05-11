import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { QRCodeSVG } from "qrcode.react";
import { Button } from "../../components/ui/Button";
import { Card } from "../../components/ui/Card";
import { CodeFence } from "../../components/ui/CodeFence";
import { CopyButton } from "../../components/ui/CopyButton";
import { peersApi } from "../../services/api";

export function PeerConfigPage() {
  const { tid, pid } = useParams();
  const tunnelId = Number(tid);
  const peerId = Number(pid);
  const [tab, setTab] = useState<"wg" | "mikrotik">("wg");

  const wg = useQuery({
    queryKey: ["peer-config", tunnelId, peerId],
    queryFn: ({ signal }) => peersApi.config(tunnelId, peerId, { signal }),
    enabled: Number.isFinite(tunnelId) && Number.isFinite(peerId),
  });

  const mk = useQuery({
    queryKey: ["peer-mikrotik", tunnelId, peerId],
    queryFn: ({ signal }) => peersApi.mikrotik(tunnelId, peerId, { signal }),
    enabled: tab === "mikrotik" && Number.isFinite(tunnelId) && Number.isFinite(peerId),
  });

  if (!Number.isFinite(tunnelId) || !Number.isFinite(peerId)) {
    return <p className="text-[#ef4444]">Invalid route</p>;
  }

  const wgText = wg.data ?? "";
  const mikrotikText = mk.data ?? "";

  function downloadConf() {
    const blob = new Blob([wgText], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `peer-${peerId}.conf`;
    a.click();
    URL.revokeObjectURL(url);
  }

  return (
    <div className="space-y-8">
      <header className="space-y-1 border-b border-[#2a2a3f] pb-4">
        <h1 className="text-lg font-semibold tracking-tight text-[#f1f5f9]">Peer configuration</h1>
        <p className="max-w-2xl text-sm leading-relaxed text-[#94a3b8]">
          Use the WireGuard tab to scan or copy the client config. Switch to MikroTik for RouterOS import
          commands. QR codes always encode the raw <code className="rounded bg-[#1e1e2e] px-1 py-0.5 font-mono text-[#cbd5e1]">.conf</code> text only.
        </p>
      </header>

      <div className="flex gap-2 border-b border-[#2a2a3f] pb-2">
        <button
          type="button"
          className={`rounded-lg px-4 py-2 text-sm font-medium transition ${tab === "wg" ? "bg-[#1e1e2e] text-[#e2e8f0]" : "text-[#94a3b8] hover:text-[#cbd5e1]"}`}
          onClick={() => setTab("wg")}
        >
          WireGuard
        </button>
        <button
          type="button"
          className={`rounded-lg px-4 py-2 text-sm font-medium transition ${tab === "mikrotik" ? "bg-[#1e1e2e] text-[#e2e8f0]" : "text-[#94a3b8] hover:text-[#cbd5e1]"}`}
          onClick={() => setTab("mikrotik")}
        >
          MikroTik
        </button>
      </div>

      {tab === "wg" ? (
        <div className="space-y-8">
          <section className="space-y-3">
            <h2 className="text-sm font-semibold uppercase tracking-wide text-[#64748b]">Mobile setup</h2>
            <p className="text-sm text-[#94a3b8]">
              Scan this QR in the official WireGuard app (or any client that supports QR import). The encoded
              payload is identical to the file below — plain WireGuard config, not markdown.
            </p>
            <Card className="flex max-w-md flex-col items-center gap-4">
              {wg.isLoading ? (
                <p className="text-[#94a3b8]">Loading…</p>
              ) : wg.error ? (
                <p className="text-[#ef4444]">{(wg.error as Error).message}</p>
              ) : (
                <>
                  <QRCodeSVG
                  value={wgText}
                  size={280}
                  level="L"
                  bgColor="#ffffff"
                  fgColor="#000000"
                  includeMargin
                />
                  <p className="text-center text-xs text-[#64748b]">Scan with mobile WG client</p>
                </>
              )}
            </Card>
          </section>

          <section className="space-y-3">
            <h2 className="text-sm font-semibold uppercase tracking-wide text-[#64748b]">Configuration file</h2>
            <p className="text-sm text-[#94a3b8]">
              Paste into a <code className="rounded bg-[#1e1e2e] px-1 py-0.5 font-mono text-[#cbd5e1]">.conf</code>{" "}
              file or use copy / download. Keys and addresses stay exactly as returned by the server.
            </p>
            {wg.isLoading ? (
              <p className="text-[#94a3b8]">Loading…</p>
            ) : wg.error ? (
              <p className="text-[#ef4444]">{(wg.error as Error).message}</p>
            ) : (
              <CodeFence
                language="ini"
                code={wgText}
                actions={
                  <>
                    <CopyButton text={wgText} />
                    <Button variant="secondary" onClick={downloadConf}>
                      Download .conf
                    </Button>
                  </>
                }
              />
            )}
          </section>
        </div>
      ) : (
        <section className="space-y-4">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-[#64748b]">RouterOS CLI</h2>
          <p className="text-sm text-[#94a3b8]">
            Run these commands in a MikroTik terminal (or paste into a script). Adjust firewall and routing on
            the router to match your network.
          </p>
          <div className="rounded-lg border border-[#f59e0b]/40 bg-[#f59e0b]/10 px-4 py-3 text-sm leading-relaxed text-[#fbbf24]">
            <strong className="font-semibold text-[#f59e0b]">Note:</strong> routing rules are not included —
            configure manually on the router as needed.
          </div>
          {mk.isLoading ? (
            <p className="text-[#94a3b8]">Loading…</p>
          ) : mk.error ? (
            <p className="text-[#ef4444]">{(mk.error as Error).message}</p>
          ) : (
            <CodeFence
              language="routeros"
              code={mikrotikText}
              actions={<CopyButton text={mikrotikText} label="Copy all" />}
            />
          )}
        </section>
      )}

      <Link to={`/tunnels/${tunnelId}`} className="inline-block text-sm text-[#6366f1] hover:underline">
        ← Back to tunnel
      </Link>
    </div>
  );
}
