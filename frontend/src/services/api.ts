import { useAuthStore } from "../store/auth";

const API_PREFIX = "/api/v1";

export type ApiError = {
  code: string;
  message: string;
  requestId?: string;
};

export class ApiException extends Error {
  status: number;
  code: string;
  requestId?: string;
  constructor(status: number, code: string, message: string, requestId?: string) {
    super(message);
    this.status = status;
    this.code = code;
    this.requestId = requestId;
  }
}

export async function parseError(res: Response): Promise<never> {
  try {
    const body = await res.json();
    const e = body?.error as { code?: string; message?: string; request_id?: string } | undefined;
    throw new ApiException(
      res.status,
      e?.code ?? "HTTP_ERROR",
      e?.message ?? res.statusText,
      e?.request_id,
    );
  } catch (err) {
    if (err instanceof ApiException) throw err;
    throw new ApiException(res.status, "HTTP_ERROR", res.statusText);
  }
}

/** Request options for API calls (includes AbortSignal; skipAuth is stripped before fetch). */
export type FetchOpts = RequestInit & { skipAuth?: boolean };

const cred: RequestCredentials = "include";

async function refreshTokens(): Promise<boolean> {
  const res = await fetch(`${API_PREFIX}/auth/refresh`, {
    method: "POST",
    credentials: cred,
    headers: { "Content-Type": "application/json" },
  });
  if (!res.ok) {
    useAuthStore.getState().clear();
    return false;
  }
  return true;
}

export async function apiFetch(path: string, opts: FetchOpts = {}, retried = false): Promise<Response> {
  const { headers, skipAuth, ...rest } = opts;
  const hdrs = new Headers(headers);
  if (!hdrs.has("Content-Type") && rest.body != null && typeof rest.body === "string") {
    hdrs.set("Content-Type", "application/json");
  }
  const res = await fetch(`${API_PREFIX}${path}`, { ...rest, headers: hdrs, credentials: cred });
  if (res.status === 401 && !skipAuth && !retried) {
    const ok = await refreshTokens();
    if (ok) return apiFetch(path, opts, true);
    useAuthStore.getState().clear();
  }
  return res;
}

export async function apiJson<T>(path: string, opts: FetchOpts = {}): Promise<T> {
  const res = await apiFetch(path, opts);
  if (!res.ok) await parseError(res);
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  if (!text) return undefined as T;
  return JSON.parse(text) as T;
}

export async function apiText(path: string, opts: FetchOpts = {}): Promise<string> {
  const res = await apiFetch(path, opts);
  if (!res.ok) await parseError(res);
  return res.text();
}

export async function apiOk(path: string, opts: FetchOpts = {}): Promise<void> {
  const res = await apiFetch(path, opts);
  if (!res.ok) await parseError(res);
}

export async function apiBlobDownload(path: string, filename: string, opts: FetchOpts = {}): Promise<void> {
  const res = await apiFetch(path, opts);
  if (!res.ok) await parseError(res);
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

// —— Types (mirror Go JSON: PascalCase fields) ——

export type HealthResponse = {
  ok: boolean;
  needs_setup: boolean;
  https: boolean;
  http_warning: boolean;
};

export type Subscription = {
  ID: number;
  Name: string;
  URL: string;
  RefreshInterval: number;
  LastFetchedAt?: string | null;
  NodeCount?: number;
  Status?: string;
  ErrorMessage?: string;
};

export const MANUAL_SUB_NAME = "__manual__";

export type VlessNode = {
  ID: number;
  SubscriptionID: number;
  DisplayName: string;
  UUID: string;
  Address: string;
  Port: number;
  Flow: string;
  Network: string;
  Security: string;
  SNI: string;
  Fingerprint: string;
  PublicKey: string;
  ShortID: string;
  SpiderX: string;
  ALPN: string;
  RawURI: string;
};

export type WgInterface = {
  ID: number;
  Name: string;
  TunName: string;
  PublicKey: string;
  ListenPort: number;
  WgAddress: string;
  DNS: string;
  MTU: number;
  SubscriptionID: number | null;
  ActiveNodeID: number | null;
  XrayPort: number;
  FWMark: number;
  Status: string;
  ErrorMessage: string;
  UptimeStarted?: string | null;
  BalancingStrategy?: string;
};

export type NodeHealthInfo = {
  alive: boolean;
  delay_ms: number;
} | null;

export type TunnelNodeEntry = {
  id: number;
  subscription_id: number;
  display_name: string;
  address: string;
  port: number;
  position: number;
  health: NodeHealthInfo;
};

export type TunnelNodesResponse = {
  strategy: "round_robin" | "least_ping";
  nodes: TunnelNodeEntry[];
};

export type SetNodesBody = {
  node_ids: number[];
  strategy: "round_robin" | "least_ping";
};

export type WgPeer = {
  ID: number;
  InterfaceID: number;
  Name: string;
  PublicKey: string;
  ClientAddress: string;
  AllowedIPs: string;
  PersistentKeepalive: number;
  LastHandshake?: string | null;
  RxBytes: number;
  TxBytes: number;
};

export type PeerWithTunnel = WgPeer & { TunnelName: string };

export type StatSnapshot = {
  SampledAt: string;
  RxRate: number;
  TxRate: number;
};

export type SummaryEvent = {
  ts: string;
  level: string;
  message: string;
};

export type SummaryResponse = {
  active_tunnels: number;
  total_peers: number;
  total_rx: number;
  total_tx: number;
  /** Interface ID (stringified) → [rx B/s, tx B/s] from DB latest sample */
  tunnel_rates: Record<string, [number, number]>;
  events: SummaryEvent[];
};

export const subscriptionsApi = {
  list: (opts: FetchOpts = {}) => apiJson<Subscription[]>("/subscriptions", opts),
  get: (id: number, opts: FetchOpts = {}) => apiJson<Subscription>(`/subscriptions/${id}`, opts),
  create: (body: { name: string; url: string; refresh_interval: number }, opts: FetchOpts = {}) =>
    apiJson<Subscription>("/subscriptions", { method: "POST", body: JSON.stringify(body), ...opts }),
  update: (s: Subscription, opts: FetchOpts = {}) =>
    apiOk(`/subscriptions/${s.ID}`, { method: "PUT", body: JSON.stringify(s), ...opts }),
  remove: (id: number, opts: FetchOpts = {}) => apiOk(`/subscriptions/${id}`, { method: "DELETE", ...opts }),
  refresh: (id: number, opts: FetchOpts = {}) => apiOk(`/subscriptions/${id}/refresh`, { method: "POST", ...opts }),
  nodes: (id: number, opts: FetchOpts = {}) => apiJson<VlessNode[]>(`/subscriptions/${id}/nodes`, opts),
  addManualNode: (body: { vless_uri: string }, opts: FetchOpts = {}) =>
    apiJson<VlessNode>("/subscriptions/manual-nodes", { method: "POST", body: JSON.stringify(body), ...opts }),
  updateManualNode: (nodeId: number, body: { vless_uri: string }, opts: FetchOpts = {}) =>
    apiJson<VlessNode>(`/subscriptions/manual-nodes/${nodeId}`, {
      method: "PUT",
      body: JSON.stringify(body),
      ...opts,
    }),
  deleteManualNode: (nodeId: number, opts: FetchOpts = {}) =>
    apiOk(`/subscriptions/manual-nodes/${nodeId}`, { method: "DELETE", ...opts }),
};

export type TunnelCreateBody = {
  name: string;
  listen_port: number;
  wg_address: string;
  dns: string;
  mtu: number;
  subscription_id: number | null;
  active_node_id: number | null;
  node_ids?: number[];
  balancing_strategy?: string;
  vless_uri: string;
};

export const tunnelsApi = {
  list: (opts: FetchOpts = {}) => apiJson<WgInterface[]>("/tunnels", opts),
  get: (id: number, opts: FetchOpts = {}) => apiJson<WgInterface>(`/tunnels/${id}`, opts),
  create: (body: TunnelCreateBody, opts: FetchOpts = {}) =>
    apiJson<WgInterface>("/tunnels", { method: "POST", body: JSON.stringify(body), ...opts }),
  update: (iface: WgInterface, opts: FetchOpts = {}) =>
    apiOk(`/tunnels/${iface.ID}`, { method: "PUT", body: JSON.stringify(iface), ...opts }),
  remove: (id: number, opts: FetchOpts = {}) => apiOk(`/tunnels/${id}`, { method: "DELETE", ...opts }),
  start: (id: number, opts: FetchOpts = {}) => apiOk(`/tunnels/${id}/start`, { method: "POST", ...opts }),
  stop: (id: number, opts: FetchOpts = {}) => apiOk(`/tunnels/${id}/stop`, { method: "POST", ...opts }),
  stats: (id: number, window: "1h" | "6h" | "24h", opts: FetchOpts = {}) =>
    apiJson<StatSnapshot[]>(`/tunnels/${id}/stats?window=${encodeURIComponent(window)}`, opts),
  getNodes: (id: number, opts: FetchOpts = {}) =>
    apiJson<TunnelNodesResponse>(`/tunnels/${id}/nodes`, opts),
  setNodes: (id: number, body: SetNodesBody, opts: FetchOpts = {}) =>
    apiOk(`/tunnels/${id}/nodes`, { method: "PUT", body: JSON.stringify(body), ...opts }),
};

export const peersApi = {
  listAll: (opts: FetchOpts = {}) => apiJson<PeerWithTunnel[]>("/peers", opts),
  list: (tid: number, opts: FetchOpts = {}) => apiJson<WgPeer[]>(`/tunnels/${tid}/peers`, opts),
  create: (
    tid: number,
    body: { name: string; public_key: string; client_address: string },
    opts: FetchOpts = {}
  ) => apiJson<WgPeer>(`/tunnels/${tid}/peers`, { method: "POST", body: JSON.stringify(body), ...opts }),
  update: (tid: number, peer: WgPeer, opts: FetchOpts = {}) =>
    apiOk(`/tunnels/${tid}/peers/${peer.ID}`, { method: "PUT", body: JSON.stringify(peer), ...opts }),
  remove: (tid: number, pid: number, opts: FetchOpts = {}) =>
    apiOk(`/tunnels/${tid}/peers/${pid}`, { method: "DELETE", ...opts }),
  config: (tid: number, pid: number, opts: FetchOpts = {}) =>
    apiText(`/tunnels/${tid}/peers/${pid}/config`, opts),
  mikrotik: (tid: number, pid: number, opts: FetchOpts = {}) =>
    apiText(`/tunnels/${tid}/peers/${pid}/mikrotik`, opts),
};

export const settingsApi = {
  get: (opts: FetchOpts = {}) => apiJson<{ server_host: string }>("/settings", opts),
  put: (server_host: string, opts: FetchOpts = {}) =>
    apiOk("/settings", { method: "PUT", body: JSON.stringify({ server_host }), ...opts }),
  password: (old_password: string, new_password: string, opts: FetchOpts = {}) =>
    apiOk("/settings/password", { method: "PUT", body: JSON.stringify({ old_password, new_password }), ...opts }),
  exportDownload: async (opts: FetchOpts = {}) => {
    await apiBlobDownload("/settings/export", "xray2wg-export.json", opts);
  },
  importData: (json: string, opts: FetchOpts = {}) => apiOk("/settings/import", { method: "POST", body: json, ...opts }),
};

export const authApi = {
  login: (password: string) =>
    apiOk("/auth/login", {
      method: "POST",
      body: JSON.stringify({ password }),
      skipAuth: true,
    }),
  bootstrap: (password: string, confirm: string) =>
    apiOk("/auth/bootstrap", {
      method: "POST",
      body: JSON.stringify({ password, confirm }),
      skipAuth: true,
    }),
  logout: () => apiOk("/auth/logout", { method: "POST" }),
  /** Session probe for cookie-based auth (HttpOnly JWT). */
  me: (opts: FetchOpts = {}) => apiJson<{ user_id: number }>("/auth/me", opts),
  /** Bootstrap / first-run probe (not Kubernetes liveness). */
  setupStatus: (opts: FetchOpts = {}) =>
    apiJson<HealthResponse>("/auth/setup-status", { skipAuth: true, ...opts }),
};

export const summaryApi = {
  get: (opts: FetchOpts = {}) => apiJson<SummaryResponse>("/stats/summary", opts),
};

export type AuditLogEntry = {
  ID: number;
  Level: string;
  Source: string;
  Message: string;
  CreatedAt: string;
};

export type AuditLogResponse = {
  items: AuditLogEntry[];
  total: number;
};

export const auditApi = {
  list: (
    params: { level?: string; search?: string; limit?: number; offset?: number },
    opts: FetchOpts = {}
  ) => {
    const q = new URLSearchParams();
    if (params.level) q.set("level", params.level);
    if (params.search) q.set("search", params.search);
    if (params.limit != null) q.set("limit", String(params.limit));
    if (params.offset != null) q.set("offset", String(params.offset));
    const qs = q.toString();
    return apiJson<AuditLogResponse>(`/audit${qs ? "?" + qs : ""}`, opts);
  },
};
