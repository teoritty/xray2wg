import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Outlet, useLocation } from "react-router-dom";
import { authApi } from "../../services/api";
import { useAuthStore } from "../../store/auth";
import { Sidebar } from "./Sidebar";
import { Topbar } from "./Topbar";

const titles: { test: RegExp; title: string }[] = [
  { test: /^\/$/, title: "Dashboard" },
  { test: /^\/subscriptions\/?$/, title: "Subscriptions" },
  { test: /^\/subscriptions\/\d+/, title: "Subscription" },
  { test: /^\/tunnels\/new/, title: "Create tunnel" },
  { test: /^\/tunnels\/\d+\/edit/, title: "Edit tunnel" },
  { test: /^\/tunnels\/\d+\/peers\/new/, title: "Add peer" },
  { test: /^\/tunnels\/\d+\/peers\/\d+\/config/, title: "Peer configuration" },
  { test: /^\/tunnels\/\d+$/, title: "Tunnel" },
  { test: /^\/tunnels\/?$/, title: "Tunnels" },
  { test: /^\/statistics/, title: "Statistics" },
  { test: /^\/settings/, title: "Settings" },
];

function titleForPath(pathname: string): string {
  for (const row of titles) {
    if (row.test.test(pathname)) return row.title;
  }
  return "xray2wg";
}

export function Shell() {
  const loc = useLocation();
  const qc = useQueryClient();
  const clear = useAuthStore((s) => s.clear);
  const signedIn = useAuthStore((s) => s.signedIn);

  const { data: health } = useQuery({
    queryKey: ["setup-status"],
    queryFn: ({ signal }) => authApi.setupStatus({ signal }),
    refetchInterval: 60_000,
  });

  return (
    <div className="flex min-h-full">
      <Sidebar
        onLogout={() => {
          void authApi
            .logout()
            .catch(() => {})
            .finally(() => {
              clear();
              void qc.invalidateQueries({ queryKey: ["session"] });
              window.location.hash = "#/login";
            });
        }}
      />
      <div className="flex min-h-full flex-1 flex-col pl-[240px]">
        {signedIn && health?.http_warning ? (
          <div className="border-b border-[#f59e0b]/40 bg-[#f59e0b]/10 px-8 py-2 text-sm text-[#f59e0b]">
            You are not using HTTPS. Put a reverse proxy with TLS in front for production.
          </div>
        ) : null}
        <Topbar title={titleForPath(loc.pathname)} />
        <main className="flex-1 p-8">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
