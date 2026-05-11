import { NavLink } from "react-router-dom";
import { IconChart, IconClipboard, IconCog, IconGrid, IconLink, IconLogout, IconShield, IconUsers } from "../../assets/icons";

const link =
  "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-[#94a3b8] transition hover:bg-[#1e1e2e] hover:text-[#e2e8f0]";
const active = "bg-[#1e1e2e] text-[#e2e8f0] border-l-2 border-[#6366f1] pl-[10px]";

export function Sidebar({
  onLogout,
}: {
  onLogout: () => void;
}) {
  return (
    <aside className="fixed left-0 top-0 z-30 flex h-full w-[240px] flex-col border-r border-[#2a2a3f] bg-[#161620] px-3 py-5">
      <div className="mb-8 flex items-center gap-2 px-2 font-semibold text-[#e2e8f0]">
        <IconShield className="h-7 w-7 text-[#6366f1]" />
        <span>xray2wg</span>
      </div>
      <nav className="flex flex-1 flex-col gap-1">
        <NavLink to="/" end className={({ isActive }) => `${link} ${isActive ? active : ""}`}>
          <IconGrid className="h-5 w-5" /> Dashboard
        </NavLink>
        <NavLink to="/subscriptions" className={({ isActive }) => `${link} ${isActive ? active : ""}`}>
          <IconLink className="h-5 w-5" /> Subscriptions
        </NavLink>
        <NavLink to="/tunnels" className={({ isActive }) => `${link} ${isActive ? active : ""}`}>
          <IconShield className="h-5 w-5" /> Tunnels
        </NavLink>
        <NavLink to="/peers" className={({ isActive }) => `${link} ${isActive ? active : ""}`}>
          <IconUsers className="h-5 w-5" /> Peers
        </NavLink>
        <NavLink to="/statistics" className={({ isActive }) => `${link} ${isActive ? active : ""}`}>
          <IconChart className="h-5 w-5" /> Statistics
        </NavLink>
        <NavLink to="/audit" className={({ isActive }) => `${link} ${isActive ? active : ""}`}>
          <IconClipboard className="h-5 w-5" /> Audit Log
        </NavLink>
      </nav>
      <div className="mt-auto flex flex-col gap-1 border-t border-[#2a2a3f] pt-4">
        <NavLink to="/settings" className={({ isActive }) => `${link} ${isActive ? active : ""}`}>
          <IconCog className="h-5 w-5" /> Settings
        </NavLink>
        <button type="button" className={`${link} w-full border-0 bg-transparent text-left`} onClick={onLogout}>
          <IconLogout className="h-5 w-5" /> Logout
        </button>
      </div>
    </aside>
  );
}
