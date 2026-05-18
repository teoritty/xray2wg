import { Input, Label } from "../ui/Form";
import type { SecurityNone, SecurityReality, SecuritySpec, SecurityTLS } from "../../types/vless";

export function NoneForm({ value, onChange }: { value: SecurityNone; onChange: (v: SecurityNone) => void }) {
  void value;
  void onChange;
  return (
    <p className="text-xs text-[#64748b]">
      No transport encryption. Only use when the underlying transport (xhttp, ws over h2) already encrypts at the
      CDN edge.
    </p>
  );
}

export function TLSForm({ value, onChange }: { value: SecurityTLS; onChange: (v: SecurityTLS) => void }) {
  const alpn = (value.alpn ?? []).join(", ");
  return (
    <div className="space-y-3">
      <div>
        <Label htmlFor="tls-sni">SNI</Label>
        <Input
          id="tls-sni"
          value={value.serverName ?? ""}
          onChange={(e) => onChange({ ...value, serverName: e.target.value })}
          placeholder="vpn.example.com"
        />
      </div>
      <div>
        <Label htmlFor="tls-fp">Fingerprint</Label>
        <Input
          id="tls-fp"
          value={value.fingerprint ?? ""}
          onChange={(e) => onChange({ ...value, fingerprint: e.target.value })}
          placeholder="chrome (default)"
        />
      </div>
      <div>
        <Label htmlFor="tls-alpn">ALPN</Label>
        <Input
          id="tls-alpn"
          value={alpn}
          onChange={(e) =>
            onChange({
              ...value,
              alpn: e.target.value
                .split(",")
                .map((s) => s.trim())
                .filter(Boolean),
            })
          }
          placeholder="h2, http/1.1"
        />
      </div>
      <label className="flex items-center gap-2 text-sm text-[#cbd5e1]">
        <input
          type="checkbox"
          checked={value.allowInsecure ?? false}
          onChange={(e) => onChange({ ...value, allowInsecure: e.target.checked })}
        />
        Allow insecure certificate (self-signed / corporate CA only)
      </label>
    </div>
  );
}

export function RealityForm({ value, onChange }: { value: SecurityReality; onChange: (v: SecurityReality) => void }) {
  return (
    <div className="space-y-3">
      <div>
        <Label htmlFor="reality-sni">SNI</Label>
        <Input
          id="reality-sni"
          value={value.serverName ?? ""}
          onChange={(e) => onChange({ ...value, serverName: e.target.value })}
          placeholder="vpn.example.com"
        />
      </div>
      <div>
        <Label htmlFor="reality-fp">Fingerprint</Label>
        <Input
          id="reality-fp"
          value={value.fingerprint ?? ""}
          onChange={(e) => onChange({ ...value, fingerprint: e.target.value })}
          placeholder="chrome (default)"
        />
      </div>
      <div>
        <Label htmlFor="reality-pbk">Public key (pbk)</Label>
        <Input
          id="reality-pbk"
          value={value.publicKey}
          onChange={(e) => onChange({ ...value, publicKey: e.target.value })}
          placeholder="required"
        />
      </div>
      <div>
        <Label htmlFor="reality-sid">Short ID (sid)</Label>
        <Input
          id="reality-sid"
          value={value.shortId ?? ""}
          onChange={(e) => onChange({ ...value, shortId: e.target.value })}
          placeholder="(optional)"
        />
      </div>
      <div>
        <Label htmlFor="reality-spx">SpiderX</Label>
        <Input
          id="reality-spx"
          value={value.spiderX ?? ""}
          onChange={(e) => onChange({ ...value, spiderX: e.target.value })}
          placeholder="/ (default)"
        />
      </div>
    </div>
  );
}

export function SecurityFormSwitch({ value, onChange }: { value: SecuritySpec; onChange: (v: SecuritySpec) => void }) {
  switch (value.name) {
    case "none":
      return <NoneForm value={value} onChange={onChange} />;
    case "tls":
      return <TLSForm value={value} onChange={onChange} />;
    case "reality":
      return <RealityForm value={value} onChange={onChange} />;
  }
}
