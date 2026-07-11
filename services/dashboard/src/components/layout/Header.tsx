"use client";

import { useAuth } from "@/lib/auth/auth-guard";
import { usePathname } from "next/navigation";

const pageTitles: Record<string, string> = {
  "/": "Dashboard",
  "/analytics": "Analytics",
  "/orderbook": "Orderbook",
  "/replay": "Replay",
  "/admin": "Admin Panel",
  "/admin/health": "System Health",
  "/admin/config": "Configuration",
  "/admin/logs": "Log Viewer",
  "/admin/database": "Database Management",
  "/admin/accounts": "Account Management",
  "/admin/accounts/new": "New Account",
  "/admin/backtest": "Backtesting",
  "/admin/suggestions": "AI Suggestions",
};

export function Header() {
  const { user, logout } = useAuth();
  const pathname = usePathname();

  const title = pageTitles[pathname] ?? "PQAP";

  return (
    <header className="app-header">
      <div className="app-header-title">
        <h1>{title}</h1>
      </div>

      <div className="app-header-actions">
        {user && (
          <>
            <div className="app-header-user">
              <span className="app-header-username">{user.username}</span>
              <span className="app-header-role">{user.role}</span>
            </div>
            <button onClick={logout} className="app-header-logout" aria-label="Sign out">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
                <polyline points="16 17 21 12 16 7" />
                <line x1="21" y1="12" x2="9" y2="12" />
              </svg>
            </button>
          </>
        )}
      </div>
    </header>
  );
}
