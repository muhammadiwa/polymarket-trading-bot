"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useAuth } from "@/lib/auth/auth-guard";

interface NavItem {
  label: string;
  href: string;
  icon: React.ReactNode;
}

interface NavGroup {
  title: string;
  items: NavItem[];
}

function Icon({ d }: { d: string }) {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <path d={d} />
    </svg>
  );
}

const navGroups: NavGroup[] = [
  {
    title: "Overview",
    items: [
      {
        label: "Dashboard",
        href: "/",
        icon: <Icon d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />,
      },
    ],
  },
  {
    title: "Trading",
    items: [
      {
        label: "Analytics",
        href: "/analytics",
        icon: <Icon d="M18 20V10M12 20V4M6 20v-6" />,
      },
      {
        label: "Orderbook",
        href: "/orderbook",
        icon: <Icon d="M9 5H7a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V7a2 2 0 0 0-2-2h-2M9 5a2 2 0 0 0 2 2h2a2 2 0 0 0 2-2M9 5a2 2 0 0 1 2-2h2a2 2 0 0 1 2 2" />,
      },
      {
        label: "Replay",
        href: "/replay",
        icon: <Icon d="M5 3l14 9-14 9V3z" />,
      },
    ],
  },
  {
    title: "Administration",
    items: [
      {
        label: "Admin",
        href: "/admin",
        icon: <Icon d="M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z" />,
      },
      {
        label: "Health",
        href: "/admin/health",
        icon: <Icon d="M22 12h-4l-3 9L9 3l-3 9H2" />,
      },
      {
        label: "Config",
        href: "/admin/config",
        icon: <Icon d="M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />,
      },
      {
        label: "Logs",
        href: "/admin/logs",
        icon: <Icon d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />,
      },
      {
        label: "Database",
        href: "/admin/database",
        icon: <Icon d="M12 2C6.48 2 2 4.02 2 6.5S6.48 11 12 11s10-2.02 10-4.5S17.52 2 12 2zM2 9.5v3c0 2.48 4.48 4.5 10 4.5s10-2.02 10-4.5v-3M2 15.5v3c0 2.48 4.48 4.5 10 4.5s10-2.02 10-4.5v-3" />,
      },
      {
        label: "Accounts",
        href: "/admin/accounts",
        icon: <Icon d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2M12 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8z" />,
      },
      {
        label: "Backtest",
        href: "/admin/backtest",
        icon: <Icon d="M12 20V10M6 20V4M18 20v-6" />,
      },
      {
        label: "AI Suggestions",
        href: "/admin/suggestions",
        icon: <Icon d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" />,
      },
    ],
  },
];

export function Sidebar() {
  const pathname = usePathname();
  const { user } = useAuth();

  return (
    <aside className="sidebar">
      <div className="sidebar-header">
        <Link href="/" className="sidebar-logo">
          <svg width="28" height="28" viewBox="0 0 40 40" fill="none" aria-hidden="true">
            <rect x="2" y="2" width="36" height="36" rx="8" stroke="url(#sidebar-logo-grad)" strokeWidth="2" />
            <path d="M12 28L20 12L28 28" stroke="url(#sidebar-logo-grad)" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />
            <line x1="15" y1="22" x2="25" y2="22" stroke="url(#sidebar-logo-grad)" strokeWidth="2" strokeLinecap="round" />
            <defs>
              <linearGradient id="sidebar-logo-grad" x1="0" y1="0" x2="40" y2="40">
                <stop stopColor="#00d4ff" />
                <stop offset="1" stopColor="#00ff88" />
              </linearGradient>
            </defs>
          </svg>
          <span className="sidebar-brand">PQAP</span>
        </Link>
      </div>

      <nav className="sidebar-nav">
        {navGroups.map((group) => (
          <div key={group.title} className="sidebar-group">
            <div className="sidebar-group-title">{group.title}</div>
            {group.items.map((item) => {
              const isActive = pathname === item.href || (item.href !== "/" && pathname.startsWith(item.href));
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  className={`sidebar-link ${isActive ? "sidebar-link--active" : ""}`}
                >
                  <span className="sidebar-link-icon">{item.icon}</span>
                  <span className="sidebar-link-label">{item.label}</span>
                </Link>
              );
            })}
          </div>
        ))}
      </nav>

      {user && (
        <div className="sidebar-footer">
          <div className="sidebar-user">
            <div className="sidebar-user-avatar">
              {user.username.charAt(0).toUpperCase()}
            </div>
            <div className="sidebar-user-info">
              <span className="sidebar-user-name">{user.username}</span>
              <span className="sidebar-user-role">{user.role}</span>
            </div>
          </div>
        </div>
      )}
    </aside>
  );
}
