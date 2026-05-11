import { HashRouter, Navigate, Route, Routes } from "react-router-dom";
import { RequireAuth } from "./components/RequireAuth";
import { Shell } from "./components/layout/Shell";
import { LoginPage } from "./pages/Login";
import { Dashboard } from "./pages/Dashboard";
import { SubscriptionListPage } from "./pages/subscriptions/SubscriptionList";
import { SubscriptionDetailPage } from "./pages/subscriptions/SubscriptionDetail";
import { TunnelListPage } from "./pages/tunnels/TunnelList";
import { TunnelFormPage } from "./pages/tunnels/TunnelFormPage";
import { TunnelDetailPage } from "./pages/tunnels/TunnelDetail";
import { PeerCreatePage } from "./pages/peers/PeerCreate";
import { PeerConfigPage } from "./pages/peers/PeerConfig";
import { PeerListPage } from "./pages/peers/PeerListPage";
import { StatisticsPage } from "./pages/Statistics";
import { SettingsPage } from "./pages/Settings";

export function App() {
  return (
    <HashRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route
          element={
            <RequireAuth>
              <Shell />
            </RequireAuth>
          }
        >
          <Route index element={<Dashboard />} />
          <Route path="subscriptions" element={<SubscriptionListPage />} />
          <Route path="subscriptions/:id" element={<SubscriptionDetailPage />} />
          <Route path="peers" element={<PeerListPage />} />
          <Route path="tunnels" element={<TunnelListPage />} />
          <Route path="tunnels/new" element={<TunnelFormPage />} />
          <Route path="tunnels/:id/edit" element={<TunnelFormPage />} />
          <Route path="tunnels/:tid/peers/new" element={<PeerCreatePage />} />
          <Route path="tunnels/:tid/peers/:pid/config" element={<PeerConfigPage />} />
          <Route path="tunnels/:id" element={<TunnelDetailPage />} />
          <Route path="statistics" element={<StatisticsPage />} />
          <Route path="settings" element={<SettingsPage />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </HashRouter>
  );
}
