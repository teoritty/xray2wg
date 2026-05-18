import { Input, Label, Select } from "../ui/Form";
import {
  SECURITY_NAMES,
  TRANSPORT_TYPES,
  defaultSecurity,
  defaultTransport,
  type ManualNodeBody,
  type SecuritySpec,
  type TransportSpec,
} from "../../types/vless";
import { TransportFormSwitch } from "./TransportForms";
import { SecurityFormSwitch } from "./SecurityForms";

type Props = {
  value: ManualNodeBody;
  onChange: (v: ManualNodeBody) => void;
};

// ManualNodeForm renders the full set of editable fields for a structured VLESS node. The
// transport and security sub-forms are looked up by their discriminator so each transport
// only sees the fields it actually carries; the rest of the form (UUID, address, flow, …)
// is shared.
export function ManualNodeForm({ value, onChange }: Props) {
  function setTransportType(t: TransportSpec["type"]) {
    onChange({ ...value, network: t, transport: defaultTransport(t) });
  }
  function setTransport(t: TransportSpec) {
    onChange({ ...value, transport: t });
  }
  function setSecurityName(n: SecuritySpec["name"]) {
    onChange({ ...value, security: n, security_cfg: defaultSecurity(n) });
  }
  function setSecurity(s: SecuritySpec) {
    onChange({ ...value, security_cfg: s });
  }
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-3">
        <div>
          <Label htmlFor="m-name">Display name</Label>
          <Input id="m-name" value={value.display_name} onChange={(e) => onChange({ ...value, display_name: e.target.value })} />
        </div>
        <div>
          <Label htmlFor="m-uuid">UUID</Label>
          <Input id="m-uuid" value={value.uuid} onChange={(e) => onChange({ ...value, uuid: e.target.value })} placeholder="550e8400-…" />
        </div>
        <div>
          <Label htmlFor="m-addr">Address</Label>
          <Input id="m-addr" value={value.address} onChange={(e) => onChange({ ...value, address: e.target.value })} placeholder="vpn.example.com" />
        </div>
        <div>
          <Label htmlFor="m-port">Port</Label>
          <Input
            id="m-port"
            type="number"
            value={value.port || ""}
            onChange={(e) => onChange({ ...value, port: Number(e.target.value) || 0 })}
          />
        </div>
        <div>
          <Label htmlFor="m-flow">Flow</Label>
          <Input id="m-flow" value={value.flow} onChange={(e) => onChange({ ...value, flow: e.target.value })} placeholder="xtls-rprx-vision (optional)" />
        </div>
        <div>
          <Label htmlFor="m-pe">Packet encoding</Label>
          <Select
            id="m-pe"
            value={value.packet_encoding}
            onChange={(e) => onChange({ ...value, packet_encoding: e.target.value })}
          >
            <option value="">(default)</option>
            <option value="xudp">xudp</option>
            <option value="none">none</option>
          </Select>
        </div>
      </div>

      <div className="rounded-lg border border-[#2a2a3f] p-3 space-y-3">
        <div>
          <Label htmlFor="m-network">Transport</Label>
          <Select
            id="m-network"
            value={value.network}
            onChange={(e) => setTransportType(e.target.value as TransportSpec["type"])}
          >
            {TRANSPORT_TYPES.map((t) => (
              <option key={t} value={t}>
                {t}
              </option>
            ))}
          </Select>
        </div>
        <TransportFormSwitch value={value.transport} onChange={setTransport} />
      </div>

      <div className="rounded-lg border border-[#2a2a3f] p-3 space-y-3">
        <div>
          <Label htmlFor="m-security">Security</Label>
          <Select
            id="m-security"
            value={value.security}
            onChange={(e) => setSecurityName(e.target.value as SecuritySpec["name"])}
          >
            {SECURITY_NAMES.map((n) => (
              <option key={n} value={n}>
                {n}
              </option>
            ))}
          </Select>
        </div>
        <SecurityFormSwitch value={value.security_cfg} onChange={setSecurity} />
      </div>
    </div>
  );
}

// eslint-disable-next-line react-refresh/only-export-components -- co-located factory, not a component
export function emptyManualNodeBody(): ManualNodeBody {
  return {
    display_name: "",
    uuid: "",
    address: "",
    port: 443,
    flow: "",
    encryption: "none",
    packet_encoding: "",
    network: "tcp",
    transport: defaultTransport("tcp"),
    security: "reality",
    security_cfg: defaultSecurity("reality"),
  };
}
