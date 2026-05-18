import { Input, Label, Select } from "../ui/Form";
import type {
  TransportGRPC,
  TransportHTTPUpgrade,
  TransportKCP,
  TransportRaw,
  TransportSpec,
  TransportWS,
  TransportXHTTP,
} from "../../types/vless";

// Each form takes its slice of TransportSpec and returns the next slice on edit. Keeping all
// six forms in one file is intentional: every form is < 20 LOC, and co-locating them makes
// it obvious which fields belong to which transport.

export function RawForm({ value, onChange }: { value: TransportRaw; onChange: (v: TransportRaw) => void }) {
  void value;
  void onChange;
  return (
    <p className="text-xs text-[#64748b]">
      Plain TCP — no transport-level parameters. Set security below as required by the upstream.
    </p>
  );
}

export function WSForm({ value, onChange }: { value: TransportWS; onChange: (v: TransportWS) => void }) {
  return (
    <div className="space-y-3">
      <div>
        <Label htmlFor="ws-path">Path</Label>
        <Input id="ws-path" value={value.path ?? ""} onChange={(e) => onChange({ ...value, path: e.target.value })} placeholder="/" />
      </div>
      <div>
        <Label htmlFor="ws-host">Host header</Label>
        <Input id="ws-host" value={value.host ?? ""} onChange={(e) => onChange({ ...value, host: e.target.value })} placeholder="cdn.example.com" />
      </div>
    </div>
  );
}

export function GRPCForm({ value, onChange }: { value: TransportGRPC; onChange: (v: TransportGRPC) => void }) {
  return (
    <div className="space-y-3">
      <div>
        <Label htmlFor="grpc-service">Service name</Label>
        <Input
          id="grpc-service"
          value={value.serviceName ?? ""}
          onChange={(e) => onChange({ ...value, serviceName: e.target.value })}
          placeholder="GunService"
        />
      </div>
      <div>
        <Label htmlFor="grpc-authority">Authority (Host)</Label>
        <Input
          id="grpc-authority"
          value={value.authority ?? ""}
          onChange={(e) => onChange({ ...value, authority: e.target.value })}
          placeholder="(optional)"
        />
      </div>
      <label className="flex items-center gap-2 text-sm text-[#cbd5e1]">
        <input
          type="checkbox"
          checked={value.multiMode ?? false}
          onChange={(e) => onChange({ ...value, multiMode: e.target.checked })}
        />
        Multi-mode (experimental)
      </label>
    </div>
  );
}

export function XHTTPForm({ value, onChange }: { value: TransportXHTTP; onChange: (v: TransportXHTTP) => void }) {
  return (
    <div className="space-y-3">
      <div>
        <Label htmlFor="xhttp-path">Path</Label>
        <Input id="xhttp-path" value={value.path ?? ""} onChange={(e) => onChange({ ...value, path: e.target.value })} placeholder="/" />
      </div>
      <div>
        <Label htmlFor="xhttp-host">Host header</Label>
        <Input id="xhttp-host" value={value.host ?? ""} onChange={(e) => onChange({ ...value, host: e.target.value })} placeholder="(optional)" />
      </div>
      <div>
        <Label htmlFor="xhttp-mode">Mode</Label>
        <Select
          id="xhttp-mode"
          value={value.mode ?? "auto"}
          onChange={(e) => onChange({ ...value, mode: e.target.value as TransportXHTTP["mode"] })}
        >
          <option value="auto">auto (default)</option>
          <option value="packet-up">packet-up</option>
          <option value="stream-up">stream-up</option>
          <option value="stream-one">stream-one</option>
        </Select>
      </div>
    </div>
  );
}

export function HTTPUpgradeForm({ value, onChange }: { value: TransportHTTPUpgrade; onChange: (v: TransportHTTPUpgrade) => void }) {
  return (
    <div className="space-y-3">
      <div>
        <Label htmlFor="hu-path">Path</Label>
        <Input id="hu-path" value={value.path ?? ""} onChange={(e) => onChange({ ...value, path: e.target.value })} placeholder="/" />
      </div>
      <div>
        <Label htmlFor="hu-host">Host header</Label>
        <Input id="hu-host" value={value.host ?? ""} onChange={(e) => onChange({ ...value, host: e.target.value })} placeholder="cdn.example.com" />
      </div>
    </div>
  );
}

export function KCPForm({ value, onChange }: { value: TransportKCP; onChange: (v: TransportKCP) => void }) {
  return (
    <div className="space-y-3">
      <div>
        <Label htmlFor="kcp-header">Header obfuscation</Label>
        <Select
          id="kcp-header"
          value={value.headerType ?? "none"}
          onChange={(e) => onChange({ ...value, headerType: e.target.value as TransportKCP["headerType"] })}
        >
          {["none", "srtp", "utp", "wechat-video", "dtls", "wireguard", "dns"].map((t) => (
            <option key={t} value={t}>
              {t}
            </option>
          ))}
        </Select>
      </div>
      <div>
        <Label htmlFor="kcp-seed">Seed</Label>
        <Input id="kcp-seed" value={value.seed ?? ""} onChange={(e) => onChange({ ...value, seed: e.target.value })} placeholder="(optional)" />
      </div>
    </div>
  );
}

export function TransportFormSwitch({ value, onChange }: { value: TransportSpec; onChange: (v: TransportSpec) => void }) {
  switch (value.type) {
    case "tcp":
      return <RawForm value={value} onChange={onChange} />;
    case "ws":
      return <WSForm value={value} onChange={onChange} />;
    case "grpc":
      return <GRPCForm value={value} onChange={onChange} />;
    case "xhttp":
      return <XHTTPForm value={value} onChange={onChange} />;
    case "httpupgrade":
      return <HTTPUpgradeForm value={value} onChange={onChange} />;
    case "kcp":
      return <KCPForm value={value} onChange={onChange} />;
  }
}
