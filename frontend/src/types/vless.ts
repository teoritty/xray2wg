// Discriminated unions describing the per-transport and per-security parameter shapes the
// backend accepts in the structured manual-node API. The `type` field in TransportSpec /
// `name` in SecuritySpec is the discriminator the union switches on.
//
// These shapes match the Go Spec structs in backend/internal/vless/transport and
// backend/internal/vless/security. Keep them in sync — the JSON shape is the contract.

export type TransportRaw = { type: "tcp" };
export type TransportWS = {
  type: "ws";
  path?: string;
  host?: string;
};
export type TransportGRPC = {
  type: "grpc";
  serviceName?: string;
  multiMode?: boolean;
  authority?: string;
};
export type TransportXHTTP = {
  type: "xhttp";
  path?: string;
  host?: string;
  mode?: "auto" | "packet-up" | "stream-up" | "stream-one";
  extra?: unknown;
};
export type TransportHTTPUpgrade = {
  type: "httpupgrade";
  path?: string;
  host?: string;
};
export type TransportKCP = {
  type: "kcp";
  headerType?: "none" | "srtp" | "utp" | "wechat-video" | "dtls" | "wireguard" | "dns";
  seed?: string;
};

export type TransportSpec =
  | TransportRaw
  | TransportWS
  | TransportGRPC
  | TransportXHTTP
  | TransportHTTPUpgrade
  | TransportKCP;

export const TRANSPORT_TYPES: TransportSpec["type"][] = [
  "tcp",
  "ws",
  "grpc",
  "xhttp",
  "httpupgrade",
  "kcp",
];

export type SecurityNone = { name: "none" };
export type SecurityTLS = {
  name: "tls";
  serverName?: string;
  alpn?: string[];
  fingerprint?: string;
  allowInsecure?: boolean;
};
export type SecurityReality = {
  name: "reality";
  serverName?: string;
  fingerprint?: string;
  publicKey: string;
  shortId?: string;
  spiderX?: string;
};

export type SecuritySpec = SecurityNone | SecurityTLS | SecurityReality;

export const SECURITY_NAMES: SecuritySpec["name"][] = ["none", "tls", "reality"];

export function defaultTransport(t: TransportSpec["type"]): TransportSpec {
  switch (t) {
    case "tcp":
      return { type: "tcp" };
    case "ws":
      return { type: "ws", path: "/" };
    case "grpc":
      return { type: "grpc", serviceName: "" };
    case "xhttp":
      return { type: "xhttp", path: "/", mode: "auto" };
    case "httpupgrade":
      return { type: "httpupgrade", path: "/" };
    case "kcp":
      return { type: "kcp", headerType: "none" };
  }
}

export function defaultSecurity(name: SecuritySpec["name"]): SecuritySpec {
  switch (name) {
    case "none":
      return { name: "none" };
    case "tls":
      return { name: "tls", alpn: [] };
    case "reality":
      return { name: "reality", publicKey: "" };
  }
}

// Structured manual-node body, mirroring backend/internal/app.ManualNodeInput.
export type ManualNodeBody = {
  display_name: string;
  uuid: string;
  address: string;
  port: number;
  flow: string;
  encryption: string;
  packet_encoding: string;
  network: TransportSpec["type"];
  transport: TransportSpec;
  security: SecuritySpec["name"];
  security_cfg: SecuritySpec;
};
