import { useEffect, useState } from "react";
import { NavLink, Outlet, useNavigate, useLocation } from "react-router-dom";
import {
  LayoutDashboard,
  ArrowLeftRight,
  Store,
  PlugZap,
  BookOpen,
  Settings2,
  LogOut,
  Hexagon,
  ChevronsLeft,
  ChevronsRight,
} from "lucide-react";
import { useAuth } from "../auth/AuthContext";

const links = [
  { to: "/panel", icon: LayoutDashboard, label: "Dashboard" },
  { to: "/panel/transactions", icon: ArrowLeftRight, label: "Transactions" },
  { to: "/panel/stores", icon: Store, label: "Stores" },
  { to: "/panel/provider", icon: PlugZap, label: "Provider" },
  { to: "/panel/docs", icon: BookOpen, label: "Docs" },
  { to: "/panel/settings", icon: Settings2, label: "Settings" },
];

const titles: Record<string, string> = {
  "/panel": "Dashboard",
  "/panel/transactions": "Transactions",
  "/panel/stores": "Stores",
  "/panel/provider": "Provider",
  "/panel/docs": "Documentation",
};

export default function Layout() {
  const { logout } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [collapsed, setCollapsed] = useState(
    () => localStorage.getItem("pg_sidebar") === "collapsed",
  );

  useEffect(() => {
    localStorage.setItem("pg_sidebar", collapsed ? "collapsed" : "expanded");
  }, [collapsed]);

  const title =
    titles[location.pathname] ??
    (location.pathname.startsWith("/panel/stores/")
      ? "Store detail"
      : "Dashboard");

  const linkClass = ({ isActive }: { isActive: boolean }) =>
    `flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-colors ${
      isActive
        ? "bg-mint/10 text-mint"
        : "text-mut hover:text-fg hover:bg-raised"
    }`;

  return (
    <div className="flex h-full flex-col md:flex-row">
      {/* Sidebar — desktop */}
      <aside
        className={`hidden md:flex md:flex-col md:border-r md:border-line md:px-3 md:py-4 transition-all duration-200 ${
          collapsed ? "md:w-16" : "md:w-56"
        }`}
      >
        <button
          onClick={() => navigate("/panel")}
          className={`mb-8 flex items-center gap-2.5 px-2 ${collapsed ? "justify-center" : ""}`}
        >
          <Hexagon size={26} className="text-mint shrink-0" />
          {!collapsed && (
            <span className="text-base font-semibold tracking-tight">
              PayGateMe
            </span>
          )}
        </button>

        <nav className="flex flex-1 flex-col gap-1">
          {links.map((l) => (
            <NavLink
              key={l.to}
              to={l.to}
              end={l.to === "/panel"}
              className={linkClass}
              title={collapsed ? l.label : undefined}
            >
              <l.icon size={18} className="shrink-0" />
              {!collapsed && l.label}
            </NavLink>
          ))}
        </nav>

        <button
          onClick={() => setCollapsed((c) => !c)}
          className="mb-1 flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium text-mut transition-colors hover:bg-raised hover:text-fg"
        >
          {collapsed ? (
            <ChevronsRight size={18} className="shrink-0" />
          ) : (
            <>
              <ChevronsLeft size={18} className="shrink-0" />
              Collapse
            </>
          )}
        </button>

        <button
          onClick={logout}
          className="flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium text-mut transition-colors hover:bg-crimson-dim hover:text-crimson"
        >
          <LogOut size={18} className="shrink-0" />
          {!collapsed && "Logout"}
        </button>
      </aside>

      {/* Main column */}
      <div className="flex min-w-0 flex-1 flex-col">
        {/* Mobile header — brand + logout (sidebar hidden on mobile) */}
        <header className="flex items-center justify-between border-b border-line-soft px-5 py-3 md:hidden">
          <button
            onClick={() => navigate("/panel")}
            className="flex items-center gap-2"
          >
            <Hexagon size={20} className="text-mint" />
            <span className="text-sm font-semibold">{title}</span>
          </button>
          <button
            onClick={logout}
            className="rounded-lg p-2 text-mut hover:bg-crimson-dim hover:text-crimson"
          >
            <LogOut size={18} />
          </button>
        </header>

        <main className="flex-1 overflow-y-auto px-5 py-6 pb-24 md:pb-6">
          <div className="mx-auto max-w-4xl animate-fade">
            <Outlet />
          </div>
        </main>
      </div>

      {/* Bottom nav — mobile */}
      <nav className="fixed inset-x-0 bottom-0 z-40 flex items-center justify-around border-t border-line bg-surface/95 pb-safe backdrop-blur-xl md:hidden">
        {links.map((l) => (
          <NavLink
            key={l.to}
            to={l.to}
            end={l.to === "/panel"}
            className={linkClass}
          >
            <l.icon size={20} />
          </NavLink>
        ))}
      </nav>
    </div>
  );
}